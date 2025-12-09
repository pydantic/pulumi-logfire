# Logfire .NET example

Prereqs:
- `LOGFIRE_BASE_URL` and `LOGFIRE_API_KEY` set
- `pulumi-resource-logfire` on PATH (or set `PULUMI_PLUGIN_PATH` to repo bin)

Steps:
```
cd logfire-dotnet
pulumi stack init dev --secrets-provider=passphrase # or select existing
pulumi config set logfire:baseUrl $LOGFIRE_BASE_URL
pulumi config set --secret logfire:apiKey $LOGFIRE_API_KEY
pulumi up
```
