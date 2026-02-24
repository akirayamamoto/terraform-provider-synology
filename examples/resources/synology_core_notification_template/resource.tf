resource "synology_core_notification_template" "custom" {
  name = "Terraform Managed Template"

  settings = {
    docker_container_unexpected_exit = false
    route_down                       = true
  }
}
