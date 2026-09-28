# The project that owns the configured project API key.
data "lmnr_project" "current" {}

output "project_id" {
  value = data.lmnr_project.current.id
}
