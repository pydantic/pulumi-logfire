package logfire

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfgen"
)

func TestGeneratedSchema(t *testing.T) {
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

	t.Run("channel configuration", func(t *testing.T) {
		config := generated.PackageSpec.Types["logfire:index/ChannelConfig:ChannelConfig"]
		if got := config.Properties["includeAgentPrompt"].Type; got != "boolean" {
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
