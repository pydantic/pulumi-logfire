# Logfire Go example

Prereqs:
- `LOGFIRE_API_KEY` set

Steps:
```
cd logfire-go
pulumi stack init dev --secrets-provider=passphrase # or select existing
pulumi config set --secret logfire:apiKey $LOGFIRE_API_KEY
# Self-hosted only:
# pulumi config set logfire:baseUrl https://<self-hosted-logfire>
pulumi up
```

For Logfire SaaS, the provider infers the API endpoint from the API key region. Self-hosted customers should set `logfire:baseUrl` explicitly.
