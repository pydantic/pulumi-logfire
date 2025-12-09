// Copyright 2024, Pulumi Corporation.  All rights reserved.
//go:build dotnet || all
// +build dotnet all

package examples

import (
	"path/filepath"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func TestLogfireDotnet(t *testing.T) {
	opts := getCSBaseOptions(t).With(integration.ProgramTestOptions{
		Dir: filepath.Join(getCwd(t), "logfire-dotnet"),
	})

	integration.ProgramTest(t, &opts)
}
