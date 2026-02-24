package acctest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/synology-community/terraform-provider-synology/synology/provider"
)

// ProtoV5ProviderFactories returns a muxed ProviderServer that uses the provider code from this repo (SDK and plugin-framework).
// Used to set ProtoV5ProviderFactories in a resource.TestStep within an acceptance test.
func ProtoV6ProviderFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"synology": providerserver.NewProtocol6WithError(provider.New()()),
	}
}

func TestAccPreCheck(t *testing.T) {
	variables := []string{
		"SYNOLOGY_HOST",
		"SYNOLOGY_USER",
		"SYNOLOGY_PASSWORD",
	}

	for _, variable := range variables {
		if os.Getenv(variable) == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}
