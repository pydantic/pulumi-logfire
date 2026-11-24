package logfire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"

	pfbridge "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfgen"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
)

func TestGeneratedSchema(t *testing.T) {
	const booleanType = "boolean"

	provider := Provider()
	generator, err := tfgen.NewGenerator(tfgen.GeneratorOptions{
		Package:      provider.Name,
		Version:      provider.Version,
		ProviderInfo: provider,
		Language:     tfgen.Schema,
		Root:         afero.NewMemMapFs(),
		SkipExamples: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := generator.Generate()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("dashboard documentation", func(t *testing.T) {
		description := generated.PackageSpec.Resources["logfire:index/dashboard:Dashboard"].Description
		for _, want := range []string{
			"Manages a Logfire dashboard.",
			"## Example Usage",
			"Read a dashboard definition from a file",
			"Copy of a UI dashboard",
			"## Definition comparison",
		} {
			if !strings.Contains(description, want) {
				t.Errorf("dashboard description is missing %q", want)
			}
		}
	})

	t.Run("group mappings documentation", func(t *testing.T) {
		description := generated.PackageSpec.Resources["logfire:index/groupMappings:GroupMappings"].Description
		for _, want := range []string{
			"Manages the identity-provider group mapping",
			"## Conflict with the Helm value",
			"one Pulumi stack per instance",
			`$ pulumi import logfire:index/groupMappings:GroupMappings this "instance"`,
		} {
			if !strings.Contains(description, want) {
				t.Errorf("group mappings description is missing %q", want)
			}
		}
		for _, unwanted := range []string{"Terraform", "`moved`", "group_id"} {
			if strings.Contains(description, unwanted) {
				t.Errorf("group mappings description contains %q", unwanted)
			}
		}
	})

	t.Run("project organization", func(t *testing.T) {
		project := generated.PackageSpec.Resources["logfire:index/project:Project"]
		description := project.InputProperties["organization"].Description
		if !strings.Contains(description, "resource option to block it") || strings.Contains(description, "lifecycle") {
			t.Errorf("organization description is not rewritten for Pulumi: %q", description)
		}
		if _, ok := project.Properties["organizationId"]; !ok {
			t.Error("organizationId output is missing")
		}
		if _, ok := project.InputProperties["organizationId"]; ok {
			t.Error("organizationId must not be an input")
		}
	})

	t.Run("dashboard managed edits", func(t *testing.T) {
		const property = "managedAllowEdits"
		dashboard := generated.PackageSpec.Resources["logfire:index/dashboard:Dashboard"]
		if got := dashboard.InputProperties[property].Type; got != booleanType {
			t.Errorf("%s input type = %q, want boolean", property, got)
		}
		if got := dashboard.Properties[property].Type; got != booleanType {
			t.Errorf("%s output type = %q, want boolean", property, got)
		}
		if slices.Contains(dashboard.RequiredInputs, property) {
			t.Errorf("%s input must be optional", property)
		}
		if slices.Contains(dashboard.Required, property) {
			t.Errorf("%s output must be optional", property)
		}
	})

	t.Run("channel configuration", func(t *testing.T) {
		config := generated.PackageSpec.Types["logfire:index/ChannelConfig:ChannelConfig"]
		if got := config.Properties["includeAgentPrompt"].Type; got != booleanType {
			t.Errorf("includeAgentPrompt type = %q, want boolean", got)
		}
		if !slices.Equal(config.Required, []string{"type"}) {
			t.Errorf("required inputs = %v, want [type]", config.Required)
		}
		var nodejs struct {
			RequiredOutputs []string `json:"requiredOutputs"`
		}
		requiredOutputs := config.Required
		if language, ok := config.Language["nodejs"]; ok {
			if err := json.Unmarshal(language, &nodejs); err != nil {
				t.Fatal(err)
			}
			if nodejs.RequiredOutputs != nil {
				requiredOutputs = nodejs.RequiredOutputs
			}
		}
		if !slices.Equal(requiredOutputs, []string{"type"}) {
			t.Errorf("required outputs = %v, want [type]", requiredOutputs)
		}
	})
}

func TestDashboardDocsEditRule(t *testing.T) {
	provider := Provider()
	if provider.DocRules == nil || provider.DocRules.EditRules == nil {
		t.Fatal("dashboard docs edit rule is missing")
	}

	content := []byte(`definition = file("${path.module}/dashboard.json")`)
	for _, rule := range provider.DocRules.EditRules(nil) {
		match, err := filepath.Match(rule.Path, "dashboard.md")
		if err != nil {
			t.Fatalf("invalid edit rule glob %q: %v", rule.Path, err)
		}
		if !match {
			continue
		}
		content, err = rule.Edit("dashboard.md", content)
		if err != nil {
			t.Fatalf("edit rule failed: %v", err)
		}
	}

	if !bytes.Equal(content, []byte(`definition = file("dashboard.json")`)) {
		t.Fatalf("unexpected edited content: %s", content)
	}
}

func TestImportDocsEditRules(t *testing.T) {
	tests := map[string]string{
		"alert.md":     `$ terraform import logfire_alert.example "example-project/error-alert"`,
		"dashboard.md": `$ terraform import logfire_dashboard.example "example-project/example-dashboard"`,
		"project.md":   `$ terraform import logfire_project.example "acme/example-project"`,
	}

	content := []byte(`## Example Usage

resource "logfire_project" "example" {}

## Import

Import is supported using the following syntax:

In Terraform v1.5.0 and later, the import block can be used with the id attribute, for example:

import {
  to = logfire_project.example
  id = "acme/example-project"
}

The terraform import command can be used, for example:

terraform import logfire_project.example "acme/example-project"
`)

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			got := applyDocEdits(t, path, content)
			if !strings.Contains(got, want) {
				t.Fatalf("expected import docs to contain %q, got %q", want, got)
			}
			if strings.Contains(got, "Terraform v1.5.0") || strings.Contains(got, "import {") {
				t.Fatalf("import docs should not contain Terraform import block prose, got %q", got)
			}
		})
	}
}

func applyDocEdits(t *testing.T, path string, content []byte) string {
	t.Helper()

	providerInfo := Provider()
	for _, rule := range providerInfo.DocRules.EditRules(nil) {
		match, err := filepath.Match(rule.Path, path)
		if err != nil {
			t.Fatalf("invalid edit rule glob %q: %v", rule.Path, err)
		}
		if !match {
			continue
		}
		content, err = rule.Edit(path, content)
		if err != nil {
			t.Fatalf("edit rule failed: %v", err)
		}
	}
	return string(content)
}

func TestAutonamedSlugs(t *testing.T) {
	schema, err := os.ReadFile("cmd/pulumi-resource-logfire/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	server, err := pfbridge.NewProviderServer(context.Background(), nil, Provider(),
		pfbridge.ProviderMetadata{PackageSchema: schema})
	if err != nil {
		t.Fatal(err)
	}
	check := func(t *testing.T, typ tokens.Type, logicalName string, olds resource.PropertyMap,
		autonaming *pulumirpc.CheckRequest_AutonamingOptions,
	) (string, error) {
		t.Helper()
		inputs, err := plugin.MarshalProperties(resource.PropertyMap{}, plugin.MarshalOptions{})
		if err != nil {
			t.Fatal(err)
		}
		oldInputs, err := plugin.MarshalProperties(olds, plugin.MarshalOptions{})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := server.Check(context.Background(), &pulumirpc.CheckRequest{
			Urn:        string(resource.NewURN("stack", "project", "", typ, logicalName)),
			Olds:       oldInputs,
			News:       inputs,
			RandomSeed: []byte("seed"),
			Autonaming: autonaming,
		})
		if err != nil {
			return "", err
		}
		if len(resp.Failures) > 0 {
			return "", fmt.Errorf("check failed: %s", resp.Failures[0].Reason)
		}
		checked, err := plugin.UnmarshalProperties(resp.Inputs, plugin.MarshalOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return checked["name"].StringValue(), nil
	}
	propose := func(name string) *pulumirpc.CheckRequest_AutonamingOptions {
		return &pulumirpc.CheckRequest_AutonamingOptions{
			ProposedName: name, Mode: pulumirpc.CheckRequest_AutonamingOptions_PROPOSE,
		}
	}

	suffix := regexp.MustCompile(`-[0-9a-f]{7}$`)
	for typ, fallback := range map[tokens.Type]string{
		"logfire:index/project:Project":           "project",
		"logfire:index/organization:Organization": "organization",
	} {
		// Without a pattern, the logical name becomes a slug and gets a random suffix.
		for logicalName, stem := range map[string]string{
			"myProject": "myproject", "My_Org.1": "my-org-1", "plain": "plain", "A": "a",
			strings.Repeat("long-", 12):    strings.Repeat("long-", 8) + "lo",
			strings.Repeat("a", 41) + "-b": strings.Repeat("a", 41),
			"東京":                           fallback,
		} {
			name, err := check(t, typ, logicalName, nil, nil)
			if err != nil || !suffix.MatchString(name) || strings.TrimSuffix(name, name[len(name)-8:]) != stem {
				t.Errorf("%s %q: generated name %q, error %v; want %q and a random suffix", typ, logicalName, name, err, stem)
			}
		}

		// A name in prior state is kept instead of a new one.
		kept := resource.PropertyMap{"name": resource.NewStringProperty("kept-name")}
		if name, err := check(t, typ, "named", kept, nil); err != nil || name != "kept-name" {
			t.Errorf("%s: prior name became %q, error %v", typ, name, err)
		}

		// A pattern's proposal is used as given. A valid one at the length limit passes.
		valid := strings.Repeat("a", 34) + "x1abcdef-sharedx"
		if name, err := check(t, typ, "named", nil, propose(valid)); err != nil || name != valid {
			t.Errorf("%s: proposed name %q became %q, error %v", typ, valid, name, err)
		}
		// An invalid proposal fails at preview instead of being converted. A long one becomes a
		// missing name.
		const notSlug = "lowercase"
		for proposed, value := range map[string]string{
			"My_Project-1abcdef": notSlug, "Admin": notSlug, "foo_bar": notSlug,
			"A": "length", "東京": "東京",
			strings.Repeat("a", 51): "Missing required property",
		} {
			name, err := check(t, typ, "named", nil, propose(proposed))
			if err == nil || !strings.Contains(err.Error(), value) {
				t.Errorf("%s: proposed name %q became %q, error %v; want an error containing %q",
					typ, proposed, name, err, value)
			}
		}
	}
}
