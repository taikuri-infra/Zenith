# =============================================================================
# Zenith Tenant — per-customer MC + Web deployments (optional)
# =============================================================================

resource "helm_release" "zenith_tenant" {
  count = var.enable_tenants ? 1 : 0

  name      = "zenith-tenant"
  chart     = var.chart_repository != "" ? "zenith-tenant" : var.tenant_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.tenant_values_file)]

  set_sensitive {
    name  = "registry.host"
    value = var.registry_host
  }

  set_sensitive {
    name  = "registry.username"
    value = var.registry_username
  }

  set_sensitive {
    name  = "registry.password"
    value = var.registry_password
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}
