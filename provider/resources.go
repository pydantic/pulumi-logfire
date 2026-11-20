// Copyright 2016-2024, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logfire

import (
	"bytes"
	"context"
	"path"
	"regexp"
	"strings"

	// Allow embedding bridge-metadata.json in the provider.
	_ "embed"

	pfbridge "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge/info"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge/tokens"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"

	providerversion "github.com/pydantic/pulumi-logfire/provider/pkg/version"
	"github.com/pydantic/pulumi-logfire/provider/shim"
)

// all of the token components used below.
const (
	// This variable controls the default name of the package in the package
	// registries for nodejs and python:
	mainPkg = "logfire"
	// modules:
	mainMod = "index" // the logfire module

	projectImportSection = "## Import\n\n" +
		"Import by organization name and project name:\n\n" +
		"```console\n" +
		"$ terraform import logfire_project.example \"acme/example-project\"\n" +
		"```\n\n" +
		"Single project UUID/name imports are also supported."
	alertImportSection = "## Import\n\n" +
		"Import by project name and alert name:\n\n" +
		"```console\n" +
		"$ terraform import logfire_alert.example \"example-project/error-alert\"\n" +
		"```\n\n" +
		"Project UUID/alert UUID imports are also supported."
	dashboardImportSection = "## Import\n\n" +
		"Import by project name and dashboard slug:\n\n" +
		"```console\n" +
		"$ terraform import logfire_dashboard.example \"example-project/example-dashboard\"\n" +
		"```\n\n" +
		"Project UUID/dashboard UUID/slug imports are also supported."
)

//go:embed cmd/pulumi-resource-logfire/bridge-metadata.json
var metadata []byte

// Provider returns additional overlaid schema and metadata associated with the provider.
func Provider() tfbridge.ProviderInfo {
	// Create a Pulumi provider mapping
	prov := tfbridge.ProviderInfo{
		// Instantiate the Terraform provider
		//nolint:lll
		P: pfbridge.ShimProvider(shim.New(providerversion.Version)()),

		Name:    "logfire",
		Version: providerversion.Version,
		// DisplayName is a way to be able to change the casing of the provider name when being
		// displayed on the Pulumi registry
		DisplayName: "Logfire",
		// Change this to your personal name (or a company name) that you would like to be shown in
		// the Pulumi Registry if this package is published there.
		Publisher: "Pydantic",
		// LogoURL is optional but useful to help identify your package in the Pulumi Registry
		// if this package is published there.
		//
		// You may host a logo on a domain you control or add an PNG logo (100x100) for your package
		// in your repository and use the raw content URL for that file as your logo URL.
		LogoURL: "https://avatars.githubusercontent.com/u/110818415?s=200&v=4",
		// PluginDownloadURL is an optional URL used to download the Provider
		// for use in Pulumi programs
		// e.g. https://github.com/org/pulumi-provider-name/releases/download/v${VERSION}/
		PluginDownloadURL: "github://api.github.com/pydantic/pulumi-logfire",
		Description:       "A Pulumi package for creating and managing logfire cloud resources.",
		// category/cloud tag helps with categorizing the package in the Pulumi Registry.
		// For all available categories, see `Keywords` in
		// https://www.pulumi.com/docs/guides/pulumi-packages/schema/#package.
		Keywords:   []string{"logfire", "category/cloud"},
		License:    "Apache-2.0",
		Homepage:   "https://pydantic.dev/logfire",
		Repository: "https://github.com/pydantic/pulumi-logfire",
		// The GitHub Org for the provider - defaults to `terraform-providers`. Note that this should
		// match the TF provider module's require directive, not any replace directives.
		GitHubOrg:    "pydantic",
		MetadataInfo: tfbridge.NewProviderMetadata(metadata),
		DocRules: &tfbridge.DocRuleInfo{
			EditRules: func(defaults []tfbridge.DocsEdit) []tfbridge.DocsEdit {
				return append([]tfbridge.DocsEdit{{
					Path: "dashboard.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						content = bytes.ReplaceAll(content, []byte("Terraform workflow"), []byte("Workflow"))
						content = bytes.ReplaceAll(content, []byte("Terraform then warns"), []byte("Pulumi then warns"))
						content = bytes.ReplaceAll(content,
							[]byte("Use `file()` to read a dashboard definition"),
							[]byte("Read a dashboard definition from a file"))
						return bytes.ReplaceAll(
							content,
							[]byte(`file("${path.module}/dashboard.json")`),
							[]byte(`file("dashboard.json")`),
						), nil
					},
				}, {
					Path: "frontend_application.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						return bytes.ReplaceAll(content,
							[]byte("Terraform keeps the configured name"),
							[]byte("Pulumi keeps the configured name")), nil
					},
				}, {
					Path: "alert.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						// Keep API values out of the bridge's property-name rewriter.
						content = bytes.ReplaceAll(content, []byte("`instances_changed`"), []byte(`'instances\_changed'`))
						content = bytes.ReplaceAll(content, []byte("`no_longer_matching`"), []byte(`'no\_longer\_matching'`))
						content = withoutExample(content, "\n# Track each failing service as its own occurrence.", "\n```")
						return replaceImportSection(content, alertImportSection), nil
					},
				}, {
					Path: "channel.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						content = withoutExample(content, "\nvariable \"opsgenie_auth_key\" {", "\nvariable \"pagerduty_routing_key\" {")
						return withoutExample(content,
							"\nresource \"logfire_channel\" \"opsgenie_eu\" {",
							"\nresource \"logfire_channel\" \"pagerduty\" {"), nil
					},
				}, {
					Path: "dashboard.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						return replaceImportSection(content, dashboardImportSection), nil
					},
				}, {
					Path: "project.md",
					Edit: func(_ string, content []byte) ([]byte, error) {
						return replaceImportSection(content, projectImportSection), nil
					},
				}}, defaults...)
			},
		},
		Config: map[string]*tfbridge.SchemaInfo{},
		// If extra types are needed for configuration, they can be added here.
		ExtraTypes: map[string]schema.ComplexTypeSpec{},
		JavaScript: &tfbridge.JavaScriptInfo{
			PackageName: "@pydantic/pulumi-logfire",
			// RespectSchemaVersion ensures the SDK is generated linking to the correct version of the provider.
			RespectSchemaVersion: true,
		},
		Python: &tfbridge.PythonInfo{
			PackageName: "pulumi_logfire",
			// RespectSchemaVersion ensures the SDK is generated linking to the correct version of the provider.
			RespectSchemaVersion: true,
			// Enable modern PyProject support in the generated Python SDK.
			PyProject: struct{ Enabled bool }{true},
		},
		Golang: &tfbridge.GolangInfo{
			// Set where the SDK is going to be published to.
			ImportBasePath: path.Join(
				"github.com/pydantic/pulumi-logfire/sdk/",
				tfbridge.GetModuleMajorVersion(providerversion.Version),
				"go",
				mainPkg,
			),
			// Opt in to all available code generation features.
			GenerateResourceContainerTypes: true,
			GenerateExtraInputTypes:        true,
			// RespectSchemaVersion ensures the SDK is generated linking to the correct version of the provider.
			RespectSchemaVersion: true,
		},
	}

	// MustComputeTokens maps all resources and datasources from the upstream provider into Pulumi.
	//
	// tokens.SingleModule puts every upstream item into your provider's main module.
	//
	// You shouldn't need to override anything, but if you do, use the [tfbridge.ProviderInfo.Resources]
	// and [tfbridge.ProviderInfo.DataSources].
	prov.MustComputeTokens(tokens.SingleModule("logfire_", mainMod,
		tokens.MakeStandard(mainPkg)))

	prov.Resources["logfire_channel"].Fields = map[string]*tfbridge.SchemaInfo{
		"config": {
			Elem: &tfbridge.SchemaInfo{
				Fields: map[string]*tfbridge.SchemaInfo{
					"base_url":             {MarkAsOptional: tfbridge.True()},
					"include_agent_prompt": {MarkAsOptional: tfbridge.True()},
				},
			},
		},
	}

	prov.Resources["logfire_dashboard"].Fields = map[string]*tfbridge.SchemaInfo{
		"managed_allow_edits": {MarkAsOptional: tfbridge.True()},
	}

	for res, fallback := range map[string]string{"logfire_project": "project", "logfire_organization": "organization"} {
		prov.Resources[res].Fields = map[string]*tfbridge.SchemaInfo{"name": slugAutoName(fallback)}
	}

	prov.MustApplyAutoAliases()
	prov.SetAutonaming(255, "-")

	return prov
}

// withoutExample drops examples whose nested arguments the legacy converter misgenerates.
func withoutExample(content []byte, start, next string) []byte {
	prefix, example, found := bytes.Cut(content, []byte(start))
	if !found {
		return content
	}
	_, suffix, found := bytes.Cut(example, []byte(next))
	if !found {
		return content
	}
	return bytes.Join([][]byte{prefix, suffix}, []byte(next))
}

func replaceImportSection(content []byte, section string) []byte {
	const heading = "## Import\n"

	start := bytes.Index(content, []byte(heading))
	if start == -1 {
		return content
	}

	end := len(content)
	searchStart := start + len(heading)
	if next := bytes.Index(content[searchStart:], []byte("\n## ")); next != -1 {
		end = searchStart + next
	}

	out := make([]byte, 0, len(content)-end+start+len(section))
	out = append(out, content[:start]...)
	out = append(out, []byte(section)...)
	out = append(out, content[end:]...)
	return out
}

// Logfire project and organization names are lowercase slugs of at most 50 characters.
const maxSlugLen = 50

var invalidSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugAutoName generates a valid slug when a resource omits `name`. By default the bridge appends
// "-" and seven random characters to the logical name, so that part keeps at most 42 characters. A
// logical name with no usable characters, such as "東京", falls back to the resource kind.
//
// An autonaming pattern's proposal is used as given, so the provider's validation rejects an invalid
// one at preview, as it does a configured name. Converting it could map two names to one or turn a
// rejected name into a reserved one, and a delete-before-create replacement would delete the old
// resource before Logfire refused the new name. For the same reason a proposal over 50 characters
// fails at preview, where the bridge reports it as a missing `name`.
func slugAutoName(fallback string) *tfbridge.SchemaInfo {
	return &tfbridge.SchemaInfo{
		Name: "name",
		Default: &tfbridge.DefaultInfo{
			AutoNamed: true,
			ComputeDefault: func(ctx context.Context, opts tfbridge.ComputeDefaultOptions) (interface{}, error) {
				options := tfbridge.AutoNameOptions{Separator: "-", Maxlen: maxSlugLen, Randlen: 7}
				options.Transform = func(name string) string {
					s := slugify(name)
					if len(s) > maxSlugLen-8 {
						s = strings.TrimRight(s[:maxSlugLen-8], "-")
					}
					if s == "" {
						return fallback
					}
					return s
				}
				if opts.Autonaming != nil && opts.Autonaming.Mode == info.ComputeDefaultAutonamingModePropose {
					options.Transform = nil
				}
				return tfbridge.ComputeAutoNameDefault(ctx, options, opts)
			},
		},
	}
}

// slugify lowercases name and turns each run of other characters into one hyphen. The result is
// ASCII, so byte offsets are character offsets.
func slugify(name string) string {
	return strings.Trim(invalidSlugChars.ReplaceAllString(strings.ToLower(name), "-"), "-")
}
