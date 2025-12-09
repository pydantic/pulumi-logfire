# Logfire Resource Provider

Manage [Pydantic Logfire](https://pydantic.dev/logfire) projects, alerts, dashboards, channels, and API tokens with Pulumi.

## Installing

This package is available for several languages/platforms:

### Node.js (JavaScript/TypeScript)

To use from JavaScript or TypeScript in Node.js, install using either `npm`:

```bash
npm install @pulumi/logfire
```

or `yarn`:

```bash
yarn add @pulumi/logfire
```

### Python

To use from Python, install using `pip`:

```bash
pip install pulumi_logfire
```

### Go

To use from Go, use `go get` to grab the latest version of the library:

```bash
go get github.com/pulumi/pulumi-logfire/sdk/go/...
```

### .NET

To use from .NET, install using `dotnet add package`:

```bash
dotnet add package Pulumi.Logfire
```

## Configuration

The following configuration points are available for the `logfire` provider:

- `logfire:baseUrl` (environment: `LOGFIRE_BASE_URL`) – Base URL for the Logfire API, e.g. `https://logfire-us.pydantic.dev`.
- `logfire:apiKey` (environment: `LOGFIRE_API_KEY`) – Bearer token for the Logfire API.

Example stack config:

```bash
pulumi config set logfire:baseUrl https://logfire-us.pydantic.dev
pulumi config set --secret logfire:apiKey pylf_v1_...
```

## Reference

For detailed reference documentation, please visit [the Pulumi registry](https://www.pulumi.com/registry/packages/logfire/api-docs/).

## Resources

- Projects (`logfire:Project`)
- Channels (`logfire:Channel`)
- Alerts (`logfire:Alert`)
- Dashboards (`logfire:Dashboard`)
- Write tokens (`logfire:WriteToken`)
- Read tokens (`logfire:ReadToken`)
