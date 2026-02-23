output "cert_manager_status" {
  description = "cert-manager release status"
  value       = helm_release.cert_manager.status
}

output "zenith_status" {
  description = "Zenith platform release status"
  value       = helm_release.zenith.status
}

output "zenith_version" {
  description = "Zenith chart version deployed"
  value       = helm_release.zenith.version
}

output "keda_status" {
  description = "KEDA release status"
  value       = var.enable_keda ? helm_release.keda[0].status : "disabled"
}

output "monitoring_status" {
  description = "Monitoring release status"
  value       = var.enable_monitoring ? helm_release.monitoring[0].status : "disabled"
}
