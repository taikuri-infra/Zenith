# =============================================================================
# Zenith Platform — shared infrastructure (secrets, postgres, certs, middleware)
# =============================================================================

resource "helm_release" "zenith_platform" {
  name             = "zenith-platform"
  chart            = var.chart_repository != "" ? "zenith-platform" : var.platform_chart_path
  version          = var.chart_repository != "" ? var.chart_version : null
  namespace        = var.platform_namespace
  create_namespace = true
  wait             = true
  timeout          = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.platform_values_file)]

  set_sensitive {
    name  = "secrets.jwtSecret"
    value = var.jwt_secret
  }

  set_sensitive {
    name  = "secrets.adminEmail"
    value = var.admin_email
  }

  set_sensitive {
    name  = "secrets.adminPassword"
    value = var.admin_password
  }

  set_sensitive {
    name  = "secrets.githubWebhookSecret"
    value = var.github_webhook_secret
  }

  set_sensitive {
    name  = "secrets.secretsEncryptionKey"
    value = var.secrets_encryption_key
  }

  set_sensitive {
    name  = "secrets.keycloakAdminUser"
    value = "admin"
  }

  set_sensitive {
    name  = "secrets.keycloakAdminPassword"
    value = var.keycloak_admin_password
  }

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

  depends_on = [kubernetes_manifest.cluster_issuer, helm_release.cnpg]
}

# =============================================================================
# Zenith API — Go API server
# =============================================================================

resource "helm_release" "zenith_api" {
  name      = "zenith-api"
  chart     = var.chart_repository != "" ? "zenith-api" : var.api_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.api_values_file)]

  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# Zenith Landing — Next.js landing page
# =============================================================================

resource "helm_release" "zenith_landing" {
  name      = "zenith-landing"
  chart     = var.chart_repository != "" ? "zenith-landing" : var.landing_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.landing_values_file)]

  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# Zenith Demo — MC + Web demo instances (optional)
# =============================================================================

resource "helm_release" "zenith_demo" {
  count = var.enable_demo ? 1 : 0

  name      = "zenith-demo"
  chart     = var.chart_repository != "" ? "zenith-demo" : var.demo_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.demo_values_file)]

  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}
