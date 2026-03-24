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
	"path"

	// Allow embedding bridge-metadata.json in the provider.
	_ "embed"

	pfbridge "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
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
		Config:       map[string]*tfbridge.SchemaInfo{},
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

	prov.MustApplyAutoAliases()
	prov.SetAutonaming(255, "-")

	return prov
}
