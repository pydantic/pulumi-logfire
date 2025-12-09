module logfire-example

go 1.25

require (
	github.com/pulumi/pulumi/sdk/v3 v3.209.0
	github.com/pydantic/pulumi-logfire/sdk v0.0.0
)

replace github.com/pydantic/pulumi-logfire/sdk => ../../sdk
