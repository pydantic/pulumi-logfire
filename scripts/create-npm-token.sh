#!/usr/bin/env bash
set -euo pipefail

# Creates a granular npm token for CI publishing and updates the repository
# NPM_TOKEN secret that .github/workflows/publish.yml reads.
#
# Run `npm login` first. The command prompts for your password and OTP, so it
# needs a terminal. Rotate before the token expires, which is 90 days from
# creation.

REPO="pydantic/pulumi-logfire"
PACKAGE="@pydantic/pulumi-logfire"
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT
TOKEN_FILE="$WORKDIR/token.txt"

command -v gh >/dev/null || { echo "gh CLI is required."; exit 1; }
command -v npm >/dev/null || { echo "npm CLI is required."; exit 1; }
command -v script >/dev/null || { echo "script command is required to preserve interactive npm prompts."; exit 1; }

echo "Checking GitHub authentication..."
gh auth status --hostname github.com >/dev/null

echo "Checking npm authentication..."
if ! npm whoami --registry https://registry.npmjs.org/ >/dev/null 2>&1; then
    echo "Not logged in to npm. Run 'npm login' first." >&2
    exit 1
fi

echo "Creating npm granular access token..."
echo ""

# Use `script` to preserve a TTY so npm can prompt for the password and OTP.
script -q "$TOKEN_FILE" sh -c "cd '$WORKDIR' && npm token create \
  --name 'pulumi-logfire CI publish $(date +%Y-%m-%d)' \
  --expires 90 \
  --packages '$PACKAGE' \
  --packages-and-scopes-permission read-write \
  --bypass-2fa"

echo ""

TOKEN=$(
  awk -F '│' '
    tolower($2) ~ /^[[:space:]]*token[[:space:]]*$/ {
      value = $3
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      print value
      exit
    }
  ' "$TOKEN_FILE" | tr -d '\r'
)

if [ -z "$TOKEN" ]; then
  TOKEN=$(grep -Eo 'npm_[A-Za-z0-9_=-]+' "$TOKEN_FILE" | head -1 || true)
fi

if [ -z "$TOKEN" ]; then
  echo "Failed to extract token from output." >&2
  exit 1
fi

echo "Updating NPM_TOKEN secret in $REPO..."
printf '%s' "$TOKEN" | gh secret set NPM_TOKEN --repo "$REPO" >/dev/null

echo "Done. NPM_TOKEN updated in $REPO."
