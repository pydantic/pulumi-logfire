package logfire

import (
	"bytes"
	"path/filepath"
	"testing"
)

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
