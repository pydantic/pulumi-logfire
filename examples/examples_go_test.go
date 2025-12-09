// Copyright 2024, Pulumi Corporation.  All rights reserved.
//go:build go || all
// +build go all

package examples

import (
	"path/filepath"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func TestLogfireGo(t *testing.T) {
	opts := getGoBaseOptions(t).With(integration.ProgramTestOptions{
		Dir: filepath.Join(getCwd(t), "logfire-go"),
	})

	integration.ProgramTest(t, &opts)
}
