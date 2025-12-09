import pulumi
from pulumi_logfire import Provider, Project, Channel, Alert
from pulumi import Config

cfg = Config('logfire')
base_url = cfg.require('baseUrl')
api_key = cfg.require_secret('apiKey')

provider = Provider('logfire', base_url=base_url, api_key=api_key)

proj = Project('proj', name='pulumi-basic', description='Pulumi example project', opts=pulumi.ResourceOptions(provider=provider))

chan = Channel('alerts',
               name='alerts-webhook',
               active=True,
               config={
                   'type': 'webhook',
                   'format': 'auto',
                   'url': 'https://example.invalid/webhook',
               },
               opts=pulumi.ResourceOptions(provider=provider))

Alert('alert',
      project_id=proj.id,
      name='pulumi-alert',
      query='select * from records limit 1',
      time_window='15m',
      frequency='5m',
      channel_ids=[chan.id],
      notify_when='has_matches',
      active=True,
      opts=pulumi.ResourceOptions(provider=provider))

pulumi.export('projectId', proj.id)
