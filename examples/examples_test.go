// Copyright 2024, Pulumi Corporation.  All rights reserved.
package examples

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func getJSBaseOptions(t *testing.T) integration.ProgramTestOptions {
	t.Helper()
	base := getBaseOptions(t)
	baseJS := base.With(integration.ProgramTestOptions{
		Dependencies: []string{
			"@pulumi/logfire",
		},
	})

	return baseJS
}

func getPythonBaseOptions(t *testing.T) integration.ProgramTestOptions {
	t.Helper()
	base := getBaseOptions(t)
	basePython := base.With(integration.ProgramTestOptions{
		Dependencies: []string{
			filepath.Join("..", "sdk", "python", "bin"),
		},
	})

	return basePython
}

func getGoBaseOptions(t *testing.T) integration.ProgramTestOptions {
	t.Helper()
	base := getBaseOptions(t)
	baseJS := base.With(integration.ProgramTestOptions{
		Dependencies: []string{
			filepath.Join("..", "sdk", "go"),
		},
	})

	return baseJS
}

func getCSBaseOptions(t *testing.T) integration.ProgramTestOptions {
	t.Helper()
	base := getBaseOptions(t)
	baseJS := base.With(integration.ProgramTestOptions{
		Dependencies: []string{
			"Pulumi.Logfire",
		},
	})

	return baseJS
}

func getCwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	return cwd
}

func getBaseOptions(t *testing.T) integration.ProgramTestOptions {
	t.Helper()
	return integration.ProgramTestOptions{
		Config: integration.ConfigMap{
			"logfire:baseUrl": os.Getenv("LOGFIRE_BASE_URL"),
			"logfire:apiKey":  os.Getenv("LOGFIRE_API_KEY"),
		},
	}
}
