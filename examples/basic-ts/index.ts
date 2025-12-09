import * as pulumi from "@pulumi/pulumi";
import * as logfire from "@pulumi/logfire";

const config = new pulumi.Config("logfire");
const baseUrl = config.require("baseUrl");
const apiKey = config.requireSecret("apiKey");

const provider = new logfire.Provider("logfire", {
    baseUrl,
    apiKey,
});

const project = new logfire.Project("proj", {
    name: "pulumi-basic",
    description: "Pulumi example project",
}, { provider });

const channel = new logfire.Channel("alerts", {
    name: "alerts-webhook",
    active: true,
    config: {
        type: "webhook",
        format: "auto",
        url: "https://example.invalid/webhook",
    },
}, { provider });

new logfire.Alert("alert", {
    projectId: project.id,
    name: "pulumi-alert",
    query: "select * from records limit 1",
    timeWindow: "15m",
    frequency: "5m",
    channelIds: [channel.id],
    notifyWhen: "has_matches",
    active: true,
}, { provider });

export const projectId = project.id;
