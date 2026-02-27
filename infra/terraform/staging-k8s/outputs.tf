output "cert_manager" {
  description = "cert-manager status"
  value       = module.platform.cert_manager_status
}

output "platform" {
  description = "Zenith platform (shared resources) status"
  value       = module.platform.platform_status
}

output "api" {
  description = "Zenith API status"
  value       = module.platform.api_status
}

output "landing" {
  description = "Zenith Landing status"
  value       = module.platform.landing_status
}

output "demo" {
  description = "Zenith Demo status"
  value       = module.platform.demo_status
}

output "tenant" {
  description = "Zenith Tenant status"
  value       = module.platform.tenant_status
}
