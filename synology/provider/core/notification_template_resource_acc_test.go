package core_test

import (
	"fmt"
	"testing"
	"time"

	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/synology-community/terraform-provider-synology/synology/acctest"
)

func TestAccResource_CoreNotificationTemplate_Lifecycle(t *testing.T) {
	resourceName := "synology_core_notification_template.test"
	templateName := fmt.Sprintf("tf-ntf-%d", time.Now().Unix())

	r.Test(t, r.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []r.TestStep{
			{
				Config: testAccCoreNotificationTemplateConfigCreate(templateName),
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
				Config: testAccCoreNotificationTemplateConfigUpdate(templateName),
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

func testAccCoreNotificationTemplateConfigCreate(name string) string {
	return fmt.Sprintf(`
provider "synology" {}

resource "synology_core_notification_template" "test" {
  name = %q

  settings = {
    docker_container_unexpected_exit = false
    docker_image_pull_failed         = true
  }
}
`, name)
}

func testAccCoreNotificationTemplateConfigUpdate(name string) string {
	return fmt.Sprintf(`
provider "synology" {}

resource "synology_core_notification_template" "test" {
  name = %q

  settings = {
    docker_container_unexpected_exit = true
    docker_image_pull_failed         = false
  }
}
`, name+"-updated")
}
