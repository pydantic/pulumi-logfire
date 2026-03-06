package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pydantic/pulumi-logfire/sdk/go/logfire"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "logfire")
		baseURL := cfg.Get("baseUrl")
		apiKey := cfg.RequireSecret("apiKey")

		providerArgs := &logfire.ProviderArgs{
			ApiKey:  apiKey.ToStringPtrOutput(),
		}
		if baseURL != "" {
			providerArgs.BaseUrl = pulumi.StringPtr(baseURL)
		}

		provider, err := logfire.NewProvider(ctx, "logfire", providerArgs)
		if err != nil {
			return err
		}

		stack := ctx.Stack()
		if len(stack) > 34 {
			stack = stack[:34]
		}

		proj, err := logfire.NewProject(ctx, "proj", &logfire.ProjectArgs{
			Name:        pulumi.StringPtr("pulumi-basic-go-" + stack),
			Description: pulumi.StringPtr("Pulumi example project"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		ch, err := logfire.NewChannel(ctx, "alerts", &logfire.ChannelArgs{
			Name:   pulumi.String("alerts-webhook"),
			Active: pulumi.BoolPtr(true),
			Config: logfire.ChannelConfigArgs{
				Type:   pulumi.String("webhook"),
				Format: pulumi.String("auto"),
				Url:    pulumi.String("https://example.invalid/webhook"),
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = logfire.NewAlert(ctx, "alert", &logfire.AlertArgs{
			ProjectId:  proj.ID(),
			Name:       pulumi.String("pulumi-alert"),
			Query:      pulumi.String("select * from records limit 1"),
			TimeWindow: pulumi.String("15m"),
			Frequency:  pulumi.String("5m"),
			ChannelIds: pulumi.StringArray{ch.ID()},
			NotifyWhen: pulumi.String("has_matches"),
			Active:     pulumi.BoolPtr(true),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		ctx.Export("projectId", proj.ID())
		return nil
	})
}
