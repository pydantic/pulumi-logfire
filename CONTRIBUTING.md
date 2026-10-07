# Contributing

This provider is generated from `pydantic/terraform-provider-logfire` with the
Pulumi Terraform bridge.

## Local Checks

```bash
python3 scripts/test_build_workflows.py
(cd provider/shim && go test ./...)
(cd provider && go test ./...)
make schema
make generate_sdks
make build_go build_nodejs build_python
```

Normal updates use `PULUMI_CONVERT=0` by default, matching CI. To regenerate
converted examples, set `PULUMI_CONVERT=1` explicitly.

Install the local generated-artifact hook if you want it:

```bash
pre-commit install
```

The workflow checks cover build cache invalidation, newly generated SDK files,
and retries after failed Windows signing. They use temporary workspaces and do
not require cloud credentials.

## Updating Terraform Provider

Update the Terraform provider module in both `provider/` and `provider/shim/`,
run `go mod tidy` in both directories, then regenerate schema and SDKs.

## Release

Tag `main` with `vX.Y.Z`, push the tag, and watch the `release` workflow. Verify
the GitHub release, npm package, PyPI package, and `sdk/vX.Y.Z` Go tag.
