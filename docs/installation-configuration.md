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
pulumi config set logfire:baseUrl https://logfire-us.pydantic.dev
pulumi config set --secret logfire:apiKey pylf_v1_...
```

You can also use environment variables:

- `LOGFIRE_BASE_URL`
- `LOGFIRE_API_KEY`

## Configuration Reference

- `logfire:baseUrl` (string): Base URL for the Logfire API.
- `logfire:apiKey` (secret string): Bearer token for the Logfire API.
