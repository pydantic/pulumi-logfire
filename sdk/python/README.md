# Logfire Resource Provider

Manage [Pydantic Logfire](https://pydantic.dev/logfire) projects, alerts, dashboards, channels, and API tokens with Pulumi.

## Installing

This package is available for several languages/platforms:

### Node.js (JavaScript/TypeScript)

To use from JavaScript or TypeScript in Node.js, install using either `npm`:

```bash
npm install @pydantic/pulumi-logfire
```

or `yarn`:

```bash
yarn add @pydantic/pulumi-logfire
```

### Python

To use from Python, install using `pip`:

```bash
pip install pydantic-pulumi-logfire
```

Import it in code as `pulumi_logfire`.

### Go

To use from Go, use `go get` to grab the latest version of the library:

```bash
go get github.com/pydantic/pulumi-logfire/sdk/go/logfire
```

## Configuration

The following configuration points are available for the `logfire` provider:

- `logfire:baseUrl` (environment: `LOGFIRE_BASE_URL`) – Optional override for the Logfire API base URL. If omitted, the provider uses `LOGFIRE_BASE_URL` or infers the SaaS endpoint from the API key region. Self-hosted customers should set this explicitly.
- `logfire:apiKey` (environment: `LOGFIRE_API_KEY`) – Bearer token for the Logfire API.

Example stack config:

```bash
pulumi config set --secret logfire:apiKey pylf_v2_us_...
# Self-hosted only:
# pulumi config set logfire:baseUrl https://<self-hosted-logfire>
```

For Logfire SaaS, the provider infers `https://logfire-us.pydantic.dev` or `https://logfire-eu.pydantic.dev` from the API key region. If you set `logfire:baseUrl` or `LOGFIRE_BASE_URL`, that value is used instead.

## Reference

For detailed reference documentation, please visit [the Pulumi registry](https://www.pulumi.com/registry/packages/logfire/api-docs/).

## Resources

- Projects (`logfire:Project`)
- Channels (`logfire:Channel`)
- Alerts (`logfire:Alert`)
- Dashboards (`logfire:Dashboard`)
- Write tokens (`logfire:WriteToken`)
- Read tokens (`logfire:ReadToken`)
- Organizations (`logfire:Organization`)
