#!/usr/bin/env bash
set -euo pipefail

for file in "$@"; do
  case "$file" in
    *.json)
      perl -0pi -e 's/std\.File\(ctx, map\[string\]interface\{\}\{/std.File(ctx, \\u0026std.FileArgs{/g; s/\\"input\\": \\"dashboard\.json\\",/Input: \\"dashboard.json\\",/g; s/Definition: invokeFile\.Result,/Definition: pulumi.String(invokeFile.Result),/g' "$file"
      ;;
    *)
      perl -0pi -e 's/std\.File\(ctx, map\[string\]interface\{\}\{/std.File(ctx, \&std.FileArgs{/g; s/"input": "dashboard\.json",/Input: "dashboard.json",/g; s/Definition: invokeFile\.Result,/Definition: pulumi.String(invokeFile.Result),/g' "$file"
      ;;
  esac
done
