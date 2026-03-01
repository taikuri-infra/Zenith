# =============================================================================
# PodDisruptionBudgets — HA for infrastructure services (5.22)
# =============================================================================

locals {
  # NOTE: keycloak, apisix-etcd, external-dns, keda, and CNPG clusters
  # already have PDBs created by their own Helm charts.
  # Only add PDBs here for services that don't create their own.
  pdb_services = {
    apisix = {
      namespace    = "apisix"
      match_labels = { "app.kubernetes.io/name" = "apisix" }
    }
    argocd-server = {
      namespace    = "argocd"
      match_labels = { "app.kubernetes.io/name" = "argocd-server" }
    }
    cnpg-operator = {
      namespace    = "cnpg-system"
      match_labels = { "app.kubernetes.io/name" = "cloudnative-pg" }
    }
    prometheus = {
      namespace    = "monitoring"
      match_labels = { "app.kubernetes.io/name" = "prometheus" }
    }
    grafana = {
      namespace    = "monitoring"
      match_labels = { "app.kubernetes.io/name" = "grafana" }
    }
    temporal-frontend = {
      namespace    = "temporal"
      match_labels = { "app.kubernetes.io/component" = "frontend" }
    }
    temporal-history = {
      namespace    = "temporal"
      match_labels = { "app.kubernetes.io/component" = "history" }
    }
    harbor-core = {
      namespace    = "harbor"
      match_labels = { "component" = "core" }
    }
    harbor-registry = {
      namespace    = "harbor"
      match_labels = { "component" = "registry" }
    }
    cert-manager-webhook = {
      namespace    = "cert-manager"
      match_labels = { "app.kubernetes.io/component" = "webhook" }
    }
    alertmanager = {
      namespace    = "monitoring"
      match_labels = { "app.kubernetes.io/name" = "alertmanager" }
    }
    loki = {
      namespace    = "monitoring"
      match_labels = { "app.kubernetes.io/name" = "loki" }
    }
  }
}

resource "kubernetes_pod_disruption_budget_v1" "infra" {
  for_each = local.pdb_services

  metadata {
    name      = "${each.key}-pdb"
    namespace = each.value.namespace
  }

  spec {
    min_available = "1"

    selector {
      match_labels = each.value.match_labels
    }
  }
}
