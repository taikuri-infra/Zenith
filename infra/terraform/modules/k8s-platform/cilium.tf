# =============================================================================
# Cilium — eBPF-based CNI (replaces Flannel)
# =============================================================================
# K3s must be configured with:
#   flannel-backend: none
#   disable-network-policy: true
#
# Provides: pod networking, NetworkPolicy enforcement, Hubble observability
# =============================================================================

resource "helm_release" "cilium" {
  name             = "cilium"
  repository       = "https://helm.cilium.io/"
  chart            = "cilium"
  version          = var.cilium_version
  namespace        = "kube-system"
  wait             = true
  timeout          = 300

  # Operator
  set {
    name  = "operator.replicas"
    value = "1"
  }

  # Hubble observability
  set {
    name  = "hubble.enabled"
    value = "true"
  }

  set {
    name  = "hubble.relay.enabled"
    value = "true"
  }

  set {
    name  = "hubble.ui.enabled"
    value = "true"
  }

  # Resources — operator
  set {
    name  = "operator.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "operator.resources.requests.memory"
    value = "128Mi"
  }

  # Resources — agent
  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  # Resources — Hubble relay
  set {
    name  = "hubble.relay.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "hubble.relay.resources.requests.memory"
    value = "128Mi"
  }
}
