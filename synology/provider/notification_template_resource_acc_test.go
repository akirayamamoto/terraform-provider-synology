package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResource_CoreNotificationTemplate_Lifecycle(t *testing.T) {
	resourceName := "synology_core_notification_template.test"
	templateName := fmt.Sprintf("tf-ntf-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCoreNotificationTemplateConfigCreate(templateName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", templateName),
					resource.TestCheckResourceAttr(resourceName, "settings.docker_container_unexpected_exit", "false"),
					resource.TestCheckResourceAttr(resourceName, "settings.docker_image_pull_failed", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCoreNotificationTemplateConfigUpdate(templateName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", templateName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "settings.docker_container_unexpected_exit", "true"),
					resource.TestCheckResourceAttr(resourceName, "settings.docker_image_pull_failed", "false"),
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
