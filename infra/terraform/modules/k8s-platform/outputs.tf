output "cert_manager_status" {
  description = "cert-manager release status"
  value       = helm_release.cert_manager.status
}

output "platform_status" {
  description = "Zenith platform (shared resources) release status"
  value       = helm_release.zenith_platform.status
}

output "api_status" {
  description = "Zenith API release status"
  value       = helm_release.zenith_api.status
}

output "landing_status" {
  description = "Zenith Landing release status"
  value       = helm_release.zenith_landing.status
}

output "demo_status" {
  description = "Zenith Demo release status"
  value       = var.enable_demo ? helm_release.zenith_demo[0].status : "disabled"
}

output "tenant_status" {
  description = "Zenith Tenant release status"
  value       = var.enable_tenants ? helm_release.zenith_tenant[0].status : "disabled"
}

output "kong_status" {
  description = "Kong API Gateway release status"
  value       = var.enable_kong ? helm_release.kong[0].status : "disabled"
}

output "keda_status" {
  description = "KEDA release status"
  value       = var.enable_keda ? helm_release.keda[0].status : "disabled"
}

output "monitoring_status" {
  description = "Monitoring release status"
  value       = var.enable_monitoring ? helm_release.monitoring[0].status : "disabled"
}
