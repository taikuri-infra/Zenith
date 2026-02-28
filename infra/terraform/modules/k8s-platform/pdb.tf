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
