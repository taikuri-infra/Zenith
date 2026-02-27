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

output "apisix_status" {
  description = "APISIX API Gateway release status"
  value       = var.enable_apisix ? helm_release.apisix[0].status : "disabled"
}

output "keda_status" {
  description = "KEDA release status"
  value       = var.enable_keda ? helm_release.keda[0].status : "disabled"
}

output "monitoring_status" {
  description = "Monitoring (Prometheus Stack) release status"
  value       = var.enable_monitoring ? helm_release.prometheus_stack[0].status : "disabled"
}

output "keycloak_status" {
  description = "Keycloak release status"
  value       = var.enable_keycloak ? helm_release.keycloak[0].status : "disabled"
}

output "argocd_status" {
  description = "ArgoCD release status"
  value       = var.enable_argocd ? helm_release.argocd[0].status : "disabled"
}

output "harbor_status" {
  description = "Harbor registry release status"
  value       = var.enable_harbor ? helm_release.harbor[0].status : "disabled"
}

output "temporal_status" {
  description = "Temporal workflow engine release status"
  value       = var.enable_temporal ? helm_release.temporal[0].status : "disabled"
}

output "velero_status" {
  description = "Velero backup release status"
  value       = var.enable_velero ? helm_release.velero[0].status : "disabled"
}

output "kyverno_status" {
  description = "Kyverno policy engine release status"
  value       = var.enable_kyverno ? helm_release.kyverno[0].status : "disabled"
}

output "falco_status" {
  description = "Falco runtime security release status"
  value       = var.enable_falco ? helm_release.falco[0].status : "disabled"
}

output "external_dns_status" {
  description = "external-dns release status"
  value       = var.enable_external_dns ? helm_release.external_dns[0].status : "disabled"
}

output "sealed_secrets_status" {
  description = "Sealed Secrets release status"
  value       = var.enable_sealed_secrets ? helm_release.sealed_secrets[0].status : "disabled"
}
