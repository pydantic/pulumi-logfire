# Contributing to the Pulumi ecosystem

Do you want to contribute to Pulumi? Awesome! We are so happy to have you.
We have a few tips and housekeeping items to help you get up and running.

## Code of Conduct

Please make sure to read and observe our [Code of Conduct](./CODE-OF-CONDUCT.md)

## Community Expectations

Please read about our [contribution guidelines here.](https://github.com/pulumi/pulumi/blob/master/CONTRIBUTING.md#communications)

## Setting up your development environment

### Pulumi prerequisites

Please refer to the [main Pulumi repo](https://github.com/pulumi/pulumi/)'s [CONTRIBUTING.md file](
https://github.com/pulumi/pulumi/blob/master/CONTRIBUTING.md#developing) for details on how to get set up with Pulumi.

## Committing Generated Code

You must generate and check in the schema and SDKs on each pull request containing provider changes.

1. Install [pre-commit](https://pre-commit.com/) and run: `pre-commit install`
1. Regenerate committed artifacts before opening a PR:
   - `make schema PULUMI_CONVERT=0`
   - `make generate_sdks PULUMI_CONVERT=0`
1. Open a pull request containing all generated changes.
1. If a large number of unrelated diffs are produced, run `go mod tidy` in `provider/` and regenerate.

The local pre-commit hook runs generation checks only when relevant files are staged. Use `SKIP=generated-artifacts git commit ...` to bypass it for one commit.

## Running Integration Tests

The examples and integration tests in this repository will create and destroy real
cloud resources while running. Before running these tests, make sure that you have
configured access to your cloud provider with Pulumi.

*TODO: Add any steps you need to take to run integration tests here*
