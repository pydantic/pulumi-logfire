#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Ensure generation uses the repository toolchain, not global binaries.
if command -v mise >/dev/null 2>&1; then
  eval "$(mise env)"
fi

major_version="$(awk -F': *' '$1=="major-version" {print $2; exit}' .ci-mgmt.yaml | tr -d '[:space:]')"
if [[ -z "$major_version" ]]; then
  echo "[pre-commit] Could not determine major-version from .ci-mgmt.yaml"
  exit 1
fi

# Use the repository default provider version and ensure it stays aligned with CI major stream.
provider_version="$(sed -nE 's/^PROVIDER_VERSION \?= ([^[:space:]]+).*$/\1/p' Makefile | head -n1)"
if [[ -z "$provider_version" ]]; then
  echo "[pre-commit] Could not determine PROVIDER_VERSION default from Makefile"
  exit 1
fi
if [[ "$provider_version" != "${major_version}."* ]]; then
  echo "[pre-commit] Makefile PROVIDER_VERSION (${provider_version}) does not match .ci-mgmt.yaml major-version (${major_version})"
  exit 1
fi

echo "[pre-commit] Regenerating schema and SDKs (PROVIDER_VERSION=${provider_version}) to verify generated files are in sync..."

# Force regeneration even if local caches/binaries exist.
rm -f .make/schema bin/pulumi-tfgen-logfire
make schema PULUMI_CONVERT=0 PROVIDER_VERSION="$provider_version"
make generate_sdks PULUMI_CONVERT=0 PROVIDER_VERSION="$provider_version"

if ! git diff --quiet -- \
  provider/cmd/pulumi-resource-logfire/schema.json \
  provider/cmd/pulumi-resource-logfire/schema-embed.json \
  provider/cmd/pulumi-resource-logfire/bridge-metadata.json \
  sdk; then
  echo "[pre-commit] Generated artifacts changed after regeneration."
  echo "[pre-commit] Run the following and re-commit:"
  echo "  git add provider/cmd/pulumi-resource-logfire/schema.json provider/cmd/pulumi-resource-logfire/schema-embed.json provider/cmd/pulumi-resource-logfire/bridge-metadata.json sdk"
  exit 1
fi
