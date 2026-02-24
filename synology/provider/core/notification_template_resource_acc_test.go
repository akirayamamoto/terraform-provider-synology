package core_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccResource_CoreNotificationTemplate_Lifecycle(t *testing.T) {
	providerConfig := testAccCoreProviderConfig(t)
	resourceName := "synology_core_notification_template.test"
	templateName := fmt.Sprintf("tf-ntf-%d", time.Now().Unix())

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: testAccCoreNotificationTemplateConfigCreate(providerConfig, templateName),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr(resourceName, "name", templateName),
					r.TestCheckResourceAttr(resourceName, "settings.docker_container_unexpected_exit", "false"),
					r.TestCheckResourceAttr(resourceName, "settings.docker_image_pull_failed", "true"),
					r.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCoreNotificationTemplateConfigUpdate(providerConfig, templateName),
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr(resourceName, "name", templateName+"-updated"),
					r.TestCheckResourceAttr(resourceName, "settings.docker_container_unexpected_exit", "true"),
					r.TestCheckResourceAttr(resourceName, "settings.docker_image_pull_failed", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCoreProviderConfig(t *testing.T) string {
	host := os.Getenv("SYNOLOGY_HOST")
	user := os.Getenv("SYNOLOGY_USER")
	password := os.Getenv("SYNOLOGY_PASSWORD")
	skipCertCheck := os.Getenv("SYNOLOGY_SKIP_CERT_CHECK")

	if host == "" || user == "" || password == "" {
		t.Skip("acceptance test requires SYNOLOGY_HOST, SYNOLOGY_USER, and SYNOLOGY_PASSWORD")
	}

	if skipCertCheck == "" {
		skipCertCheck = "true"
	}

	return fmt.Sprintf(`
provider "synology" {
  host            = %q
  user            = %q
  password        = %q
  skip_cert_check = %q
}
`, host, user, password, skipCertCheck)
}

func testAccCoreNotificationTemplateConfigCreate(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "synology_core_notification_template" "test" {
  name = %q

  settings = {
    docker_container_unexpected_exit = false
    docker_image_pull_failed         = true
  }
}
`, providerConfig, name)
}

func testAccCoreNotificationTemplateConfigUpdate(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "synology_core_notification_template" "test" {
  name = %q

  settings = {
    docker_container_unexpected_exit = true
    docker_image_pull_failed         = false
  }
}
`, providerConfig, name+"-updated")
}
