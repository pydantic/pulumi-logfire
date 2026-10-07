#!/usr/bin/env bash
set -euo pipefail
unset NODE_AUTH_TOKEN NPM_TOKEN PYPI_USERNAME PYPI_PASSWORD

allow_nonpublishing_local=false
if [[ "${1:-}" == "--allow-nonpublishing-local-version" ]]; then
  allow_nonpublishing_local=true
  shift
fi

if [[ $# -ne 4 ]]; then
  cat >&2 <<'USAGE'
usage: rehearse_package_publisher.sh [--allow-nonpublishing-local-version] \
  VERSION NODE_ARCHIVE PYTHON_ARCHIVE OUTPUT_PREFIX

The default mode enforces publishable, exact Node and Python versions. The flag
only permits an input version with a +local suffix in a PR artifact rehearsal;
those artifacts must never be passed to a registry upload.
USAGE
  exit 2
fi

version="$1"
node_archive="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
python_archive="$(cd "$(dirname "$3")" && pwd)/$(basename "$3")"
output_prefix="$4"

case "$output_prefix" in
  /*) ;;
  *) output_prefix="$(pwd)/$output_prefix" ;;
esac
mkdir -p "$(dirname "$output_prefix")"

if [[ "$allow_nonpublishing_local" == true && "$version" != *+* ]]; then
  echo '--allow-nonpublishing-local-version requires VERSION to contain a +local suffix' >&2
  exit 2
fi
if [[ "$allow_nonpublishing_local" == false && "$version" == *+* ]]; then
  echo 'publish rehearsal rejects +local versions; use the explicit nonpublishing PR mode' >&2
  exit 2
fi

workspace="$(mktemp -d)"
trap 'rm -rf "$workspace"' EXIT
mkdir -p "$workspace/sdk/nodejs" "$workspace/sdk/python" "$workspace/npm-wrapper"

# Match the artifact extraction steps in pulumi-package-publisher v0.0.23.
tar -zxf "$node_archive" -C "$workspace/sdk/nodejs"
tar -zxf "$python_archive" -C "$workspace/sdk/python"

node_expected_name='@pydantic/pulumi-logfire'
python_expected_name='pydantic-pulumi-logfire'

node_actual="v$(jq -er .version "$workspace/sdk/nodejs/bin/package.json")"
node_expected="$(pulumictl convert-version --version "$version" --language javascript)"
node_name="$(jq -er .name "$workspace/sdk/nodejs/bin/package.json")"
printf 'node input=%s expected=%s actual=%s name=%s\n' \
  "$version" "$node_expected" "$node_actual" "$node_name" |
  tee "${output_prefix}-versions.txt"
test "$node_name" = "$node_expected_name"
test "$node_actual" = "$node_expected"

# Exercise Pulumi CLI v3.209.0's actual publish-sdk implementation. Its npm
# subprocess sees this wrapper. Reads go to the real public npm registry,
# `whoami` is answered locally, and the sole write operation gains --dry-run.
# No npm token or user npmrc is made available to either process.
real_npm="$(command -v npm)"
npm_call_log="${output_prefix}-npm-wrapper-calls.txt"
: > "$npm_call_log"
cat > "$workspace/npm-wrapper/npm" <<'WRAPPER'
#!/usr/bin/env bash
set -euo pipefail
{
  printf '%q ' "$@"
  printf '\n'
} >> "$NPM_WRAPPER_LOG"

case "${1:-}" in
  whoami)
    printf 'publisher-rehearsal\n'
    ;;
  publish)
    printf '__intercepted_publish_with_dry_run__\n' >> "$NPM_WRAPPER_LOG"
    exec env -u NODE_AUTH_TOKEN -u NPM_TOKEN \
      NPM_CONFIG_USERCONFIG=/dev/null \
      NPM_CONFIG_REGISTRY=https://registry.npmjs.org \
      NPM_CONFIG_ACCESS=public \
      "$REAL_NPM" "$@" --dry-run --json
    ;;
  info|view)
    exec env -u NODE_AUTH_TOKEN -u NPM_TOKEN \
      NPM_CONFIG_USERCONFIG=/dev/null \
      NPM_CONFIG_REGISTRY=https://registry.npmjs.org \
      "$REAL_NPM" "$@"
    ;;
  *)
    printf 'refusing unexpected npm subcommand: %s\n' "${1:-<empty>}" >&2
    exit 64
    ;;
esac
WRAPPER
chmod +x "$workspace/npm-wrapper/npm"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
expected_pulumi="$(cd "$repo_root" && bash scripts/get-versions.sh | sed -n 's/^PULUMI_VERSION_MISE=//p')"
test -n "$expected_pulumi"
pulumi_bin="$(mise exec "pulumi@$expected_pulumi" -- which pulumi)"
test -x "$pulumi_bin"
pulumi_version="$("$pulumi_bin" version)"
test "$pulumi_version" = "v$expected_pulumi"
printf 'pulumi cli=%s path=%s\n' "$pulumi_version" "$pulumi_bin" |
  tee -a "${output_prefix}-versions.txt"

env -u NODE_AUTH_TOKEN -u NPM_TOKEN \
  PATH="$workspace/npm-wrapper:$PATH" \
  REAL_NPM="$real_npm" \
  NPM_WRAPPER_LOG="$npm_call_log" \
  NPM_CONFIG_USERCONFIG=/dev/null \
  NPM_CONFIG_REGISTRY=https://registry.npmjs.org \
  "$pulumi_bin" package publish-sdk nodejs --path "$workspace/sdk/nodejs/bin" \
    2>&1 | tee "${output_prefix}-pulumi-publish-sdk.log"

grep -Eq '^whoami( |$)' "$npm_call_log"
grep -Eq '^info( |$)' "$npm_call_log"

node_bare_version="${node_actual#v}"
if "$real_npm" view "$node_name@$node_bare_version" version \
  --userconfig=/dev/null --registry=https://registry.npmjs.org >/dev/null 2>&1; then
  if grep -Eq '^publish( |$)' "$npm_call_log"; then
    echo 'Pulumi attempted to republish an existing npm version' >&2
    exit 1
  fi
  node_expected_boundary='registry version exists; Pulumi no-op'
else
  grep -Eq '^publish( |$)' "$npm_call_log"
  grep -Fxq '__intercepted_publish_with_dry_run__' "$npm_call_log"
  node_expected_boundary='registry version absent; npm publish intercepted with --dry-run'
fi
printf 'node boundary=%s\n' "$node_expected_boundary" |
  tee -a "${output_prefix}-versions.txt"

# Independently inspect npm's packlist and normalized manifest for both the
# existing-version no-op and the dry-run publish branch.
(
  cd "$workspace/sdk/nodejs/bin"
  env -u NODE_AUTH_TOKEN -u NPM_TOKEN \
    NPM_CONFIG_USERCONFIG=/dev/null \
    NPM_CONFIG_REGISTRY=https://registry.npmjs.org \
    "$real_npm" pack --dry-run --json
) > "${output_prefix}-npm-pack-dry-run.json"

python_expected="$(pulumictl convert-version --version "$version" --language python)"
if [[ "$allow_nonpublishing_local" == true ]]; then
  python_expected="${python_expected}+${version#*+}"
fi

python_dist=("$workspace"/sdk/python/bin/dist/*)
test "${#python_dist[@]}" -gt 0

# Match the action's isolated Twine install, then replace only its upload
# boundary with metadata and long-description verification. Run with no upload
# credentials.
python3 -m venv "$workspace/twine-venv"
"$workspace/twine-venv/bin/python" -m pip install \
  --disable-pip-version-check --quiet --upgrade twine
env -u PYPI_USERNAME -u PYPI_PASSWORD \
  "$workspace/twine-venv/bin/python" -m twine check "${python_dist[@]}" |
  tee "${output_prefix}-twine-check.txt"

python3 - \
  "$workspace/sdk/python/bin/pyproject.toml" \
  "$python_expected_name" \
  "$python_expected" \
  "${python_dist[@]}" > "${output_prefix}-python-distributions.txt" <<'PY'
import email
import pathlib
import sys
import tarfile
import tomllib
import zipfile

pyproject_path = pathlib.Path(sys.argv[1])
expected_name = sys.argv[2]
expected_version = sys.argv[3]
dist_paths = [pathlib.Path(raw) for raw in sys.argv[4:]]

with pyproject_path.open("rb") as stream:
    project = tomllib.load(stream)["project"]
if project["name"] != expected_name or project["version"] != expected_version:
    raise SystemExit(
        f"pyproject mismatch: expected {expected_name} {expected_version}, "
        f"got {project['name']} {project['version']}"
    )

counts = {"wheel": 0, "sdist": 0}
for path in dist_paths:
    if path.suffix == ".whl":
        counts["wheel"] += 1
        with zipfile.ZipFile(path) as zf:
            metadata_name = next(
                name for name in zf.namelist() if name.endswith(".dist-info/METADATA")
            )
            message = email.message_from_bytes(zf.read(metadata_name))
    elif path.name.endswith(".tar.gz"):
        counts["sdist"] += 1
        with tarfile.open(path) as tf:
            metadata_member = next(
                member for member in tf.getmembers() if member.name.endswith("/PKG-INFO")
            )
            extracted = tf.extractfile(metadata_member)
            assert extracted is not None
            message = email.message_from_bytes(extracted.read())
    else:
        raise SystemExit(f"unexpected distribution: {path.name}")
    actual_name = message["Name"]
    actual_version = message["Version"]
    print(f"{path.name}\t{actual_name}\t{actual_version}")
    if actual_name != expected_name or actual_version != expected_version:
        raise SystemExit(
            f"distribution mismatch in {path.name}: expected "
            f"{expected_name} {expected_version}, got {actual_name} {actual_version}"
        )

if counts != {"wheel": 1, "sdist": 1}:
    raise SystemExit(f"expected one wheel and one sdist, inspected {counts}")
PY

printf 'python input=%s expected=%s name=%s mode=%s\n' \
  "$version" "$python_expected" "$python_expected_name" \
  "$([[ "$allow_nonpublishing_local" == true ]] && echo nonpublishing-local || echo publishable-exact)" |
  tee -a "${output_prefix}-versions.txt"

echo 'rehearsal complete: registry reads and local dry-runs only; no credentials loaded'
