---
title: Installation & Configuration
meta_desc: Install and configure the Pulumi Logfire provider.
layout: package
---

# Installation

## Node.js (TypeScript/JavaScript)

```bash
npm install @pydantic/pulumi-logfire
```

## Python

```bash
pip install pulumi_logfire
```

## Go

```bash
go get github.com/pydantic/pulumi-logfire/sdk/go/...
```

## Provider Configuration

Set provider config with Pulumi config values (recommended):

```bash
pulumi config set --secret logfire:apiKey pylf_v2_us_...
# Self-hosted only:
# pulumi config set logfire:baseUrl https://<self-hosted-logfire>
```

You can also use environment variables:

- `LOGFIRE_API_KEY`
- `LOGFIRE_BASE_URL` (optional override; self-hosted customers should set this)

For Logfire SaaS, the provider infers `https://logfire-us.pydantic.dev` or `https://logfire-eu.pydantic.dev` from the API key region. If you set `logfire:baseUrl` or `LOGFIRE_BASE_URL`, that value is used instead.

## Configuration Reference

- `logfire:baseUrl` (string): Optional override for the Logfire API base URL. If omitted, the provider uses `LOGFIRE_BASE_URL` or infers the SaaS endpoint from the API key region. Self-hosted customers should set this explicitly.
- `logfire:apiKey` (secret string): Bearer token for the Logfire API.
