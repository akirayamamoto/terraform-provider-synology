package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResource_CoreNotificationTemplate_Lifecycle(t *testing.T) {
	resourceName := "synology_core_notification_template.test"
	templateName := fmt.Sprintf("tf-ntf-%s", strconv.FormatInt(time.Now().UnixNano(), 36))

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
