# =============================================================================
# Traefik HelmChartConfig — k3s built-in Traefik customization
# =============================================================================
#
# k3s bundles Traefik as the default ingress controller. The HelmChartConfig
# CRD in kube-system allows overriding Traefik's Helm values without managing
# the chart ourselves.
#
# We need:
#   - allowCrossNamespace: true    — IngressRoutes can reference Services in
#                                    other namespaces (e.g., APISIX in apisix ns)
#   - allowExternalNameServices: true — IngressRoutes can route to ExternalName
#                                       Services (e.g., apisix-gateway-proxy)
# =============================================================================

resource "kubernetes_manifest" "traefik_config" {
  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "helm.cattle.io/v1"
    kind       = "HelmChartConfig"
    metadata = {
      name      = "traefik"
      namespace = "kube-system"
    }
    spec = {
      valuesContent = yamlencode({
        providers = {
          kubernetesCRD = {
            allowCrossNamespace       = true
            allowExternalNameServices = true
          }
        }
      })
    }
  }
}
