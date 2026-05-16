# Contributing

This provider is generated from `pydantic/terraform-provider-logfire` with the
Pulumi Terraform bridge.

## Local Checks

```bash
(cd provider/shim && go test ./...)
(cd provider && go test ./...)
make schema PULUMI_CONVERT=0
make generate_sdks PULUMI_CONVERT=0
make build_go build_nodejs build_python PULUMI_CONVERT=0
```

Use `PULUMI_CONVERT=0` for normal provider updates. It keeps example conversion
from creating unrelated diffs and matches CI.

Install the local generated-artifact hook if you want it:

```bash
pre-commit install
```

## Updating Terraform Provider

Update the Terraform provider module in both `provider/` and `provider/shim/`,
run `go mod tidy` in both directories, then regenerate schema and SDKs.

## Release

Tag `main` with `vX.Y.Z`, push the tag, and watch the `release` workflow. Verify
the GitHub release, npm package, PyPI package, and `sdk/vX.Y.Z` Go tag.
