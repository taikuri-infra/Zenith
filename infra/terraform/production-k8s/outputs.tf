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

output "apisix" {
  description = "APISIX API Gateway status"
  value       = module.platform.apisix_status
}

output "keycloak" {
  description = "Keycloak identity provider status"
  value       = module.platform.keycloak_status
}

output "argocd" {
  description = "ArgoCD GitOps engine status"
  value       = module.platform.argocd_status
}

output "harbor" {
  description = "Harbor container registry status"
  value       = module.platform.harbor_status
}

output "temporal" {
  description = "Temporal workflow engine status"
  value       = module.platform.temporal_status
}

output "monitoring" {
  description = "Monitoring (Prometheus+Grafana) status"
  value       = module.platform.monitoring_status
}

output "velero" {
  description = "Velero backup status"
  value       = module.platform.velero_status
}

output "kyverno" {
  description = "Kyverno policy engine status"
  value       = module.platform.kyverno_status
}

output "falco" {
  description = "Falco runtime security status"
  value       = module.platform.falco_status
}

output "external_dns" {
  description = "external-dns status"
  value       = module.platform.external_dns_status
}

output "sealed_secrets" {
  description = "Sealed Secrets status"
  value       = module.platform.sealed_secrets_status
}
