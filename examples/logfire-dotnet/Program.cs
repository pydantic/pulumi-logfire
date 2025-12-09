using System.Collections.Generic;
using Pulumi;
using Pulumi.Logfire;
using Pulumi.Logfire.Inputs;

return await Deployment.RunAsync(() =>
{
    var config = new Pulumi.Config("logfire");
    var baseUrl = config.Require("baseUrl");
    var apiKey = config.RequireSecret("apiKey");

    var provider = new Provider("logfire", new ProviderArgs
    {
        BaseUrl = baseUrl,
        ApiKey = apiKey,
    });

    var project = new Project("proj", new ProjectArgs
    {
        Name = "pulumi-basic",
        Description = "Pulumi example project",
    }, new CustomResourceOptions { Provider = provider });

    var channel = new Channel("alerts", new ChannelArgs
    {
        Name = "alerts-webhook",
        Active = true,
        Config = new ChannelConfigArgs
        {
            Type = "webhook",
            Format = "auto",
            Url = "https://example.invalid/webhook",
        },
    }, new CustomResourceOptions { Provider = provider });

    _ = new Alert("alert", new AlertArgs
    {
        ProjectId = project.Id,
        Name = "pulumi-alert",
        Query = "select * from records limit 1",
        TimeWindow = "15m",
        Frequency = "5m",
        ChannelIds = { channel.Id },
        NotifyWhen = "has_matches",
        Active = true,
    }, new CustomResourceOptions { Provider = provider });

    return new Dictionary<string, object?>
    {
        ["projectId"] = project.Id,
    };
});
