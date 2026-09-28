data "lmnr_llm_profile" "openai" {
  name = "openai"
}

output "openai_models" {
  value = data.lmnr_llm_profile.openai.models
}
