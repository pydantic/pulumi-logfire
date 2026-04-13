#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

managed_files=(
  ".config/mise.test.toml"
  ".config/mise.toml"
  ".github/actions/download-prerequisites/action.yml"
  ".github/actions/download-provider/action.yml"
  ".github/actions/download-sdk/action.yml"
  ".github/actions/upload-prerequisites/action.yml"
  ".github/actions/upload-sdk/action.yml"
  ".github/workflows/build_provider.yml"
  ".github/workflows/build_sdk.yml"
  ".github/workflows/license.yml"
  ".github/workflows/lint.yml"
  ".github/workflows/main-post-build.yml"
  ".github/workflows/main.yml"
  ".github/workflows/nightly-test.yml"
  ".github/workflows/prerelease.yml"
  ".github/workflows/prerequisites.yml"
  ".github/workflows/publish.yml"
  ".github/workflows/pull-request.yml"
  ".github/workflows/release.yml"
  ".github/workflows/resync-build.yml"
  ".github/workflows/run-acceptance-tests.yml"
  ".github/workflows/test.yml"
  ".github/workflows/upgrade-bridge.yml"
  ".github/workflows/upgrade-provider.yml"
  ".github/workflows/verify-release.yml"
)

copy_managed_file() {
  local relative_path="$1"
  local src="$tmp_dir/$relative_path"
  local dest="$root_dir/$relative_path"

  mkdir -p "$(dirname "$dest")"
  if [[ -e "$src" ]]; then
    rsync -a "$src" "$dest"
  else
    rm -f "$dest"
  fi
}

require_pattern() {
  local pattern="$1"
  local file="$2"

  if ! rg -q --fixed-strings "$pattern" "$file"; then
    echo "expected pattern '$pattern' in $file" >&2
    exit 1
  fi
}

generate_upstream_ci() {
  (
    cd "$root_dir"
    go run github.com/pulumi-labs/ci-mgmt/provider-ci@master generate --skip-migrations --out "$tmp_dir"
  )
}

patch_main_workflow() {
  local file="$root_dir/.github/workflows/main.yml"

  require_pattern "  publish:" "$file"
  require_pattern "    name: publish" "$file"
  require_pattern "  tag_release_if_labeled_needs_release:" "$file"

  perl -0pi -e 's/\n  publish:\n    name: publish/\n  publish:\n    if: \$\{\{ false \}\}\n    name: publish/' "$file"
  perl -0pi -e 's/\n  tag_release_if_labeled_needs_release:.*?(?=\n  test:)/\n/s' "$file"
}

patch_publish_workflow() {
  local file="$root_dir/.github/workflows/publish.yml"

  require_pattern "  clean_up_release_labels:" "$file"
  require_pattern "  verify_release:" "$file"

  perl -0pi -e 's/\n  clean_up_release_labels:.*?(?=\n  verify_release:)/\n/s' "$file"
}

patch_test_workflow() {
  local file="$root_dir/.github/workflows/test.yml"

  require_pattern 'ref: ${{ env.PR_COMMIT_SHA }}' "$file"
  perl -0pi -e 's/ref: \$\{\{ env\.PR_COMMIT_SHA \}\}/ref: \$\{\{ env.PR_COMMIT_SHA || github.sha \}\}/' "$file"
}

patch_prerequisites_workflow() {
  local file="$root_dir/.github/workflows/prerequisites.yml"

  require_pattern 'run: make schema' "$file"
  perl -0pi -e 's/run: make schema$/run: make schema PULUMI_CONVERT=0/m' "$file"
}

write_custom_nightly_workflow() {
  cat >"$root_dir/.github/workflows/nightly-acceptance.yml" <<'EOF'
name: Nightly Acceptance Tests

on:
  workflow_dispatch: {}
  schedule:
    - cron: "47 5 * * *"

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  prerequisites:
    permissions:
      contents: read
      pull-requests: write
    uses: ./.github/workflows/prerequisites.yml
    secrets: inherit
    with:
      default_branch: main
      is_pr: false
      is_automated: false

  build_provider:
    needs: prerequisites
    permissions:
      contents: read
    uses: ./.github/workflows/build_provider.yml
    secrets: inherit
    with:
      version: ${{ needs.prerequisites.outputs.version }}
      matrix: |
        {
          "platform": [
            {"os": "linux", "arch": "amd64"}
          ]
        }

  build_sdk:
    name: build_sdk
    needs: prerequisites
    permissions:
      contents: write
    uses: ./.github/workflows/build_sdk.yml
    secrets: inherit
    with:
      version: ${{ needs.prerequisites.outputs.version }}

  test:
    needs:
      - prerequisites
      - build_provider
      - build_sdk
    permissions:
      contents: read
    uses: ./.github/workflows/test.yml
    secrets: inherit
    with:
      version: ${{ needs.prerequisites.outputs.version }}
EOF
}

generate_upstream_ci

for relative_path in "${managed_files[@]}"; do
  copy_managed_file "$relative_path"
done

patch_main_workflow
patch_publish_workflow
patch_prerequisites_workflow
patch_test_workflow
write_custom_nightly_workflow
