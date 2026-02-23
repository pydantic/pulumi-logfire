#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Ensure generation uses the repository toolchain, not global binaries.
if command -v mise >/dev/null 2>&1; then
  eval "$(mise env)"
fi

echo "[pre-commit] Regenerating schema and SDKs to verify generated files are in sync..."
make schema PULUMI_CONVERT=0
make generate_sdks PULUMI_CONVERT=0

if ! git diff --quiet -- provider/cmd/pulumi-resource-logfire/schema.json provider/cmd/pulumi-resource-logfire/schema-embed.json sdk; then
  echo "[pre-commit] Generated artifacts changed after regeneration."
  echo "[pre-commit] Run the following and re-commit:"
  echo "  git add provider/cmd/pulumi-resource-logfire/schema.json provider/cmd/pulumi-resource-logfire/schema-embed.json sdk"
  exit 1
fi
