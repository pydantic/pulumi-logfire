package shim

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"
	upstream "github.com/pydantic/terraform-provider-logfire/shim"
)

// New returns the upstream provider constructor for the bridge.
func New(version string) func() provider.Provider {
	return upstream.New(version)
}
