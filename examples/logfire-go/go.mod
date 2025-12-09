module logfire-example

go 1.25

require (
    github.com/pulumi/pulumi/sdk/v3 v3.209.0
    github.com/pydantic/pulumi-logfire/sdk/v1 v1.0.0-alpha.0+dev
)

replace github.com/pydantic/pulumi-logfire/sdk/v1 => ../..
