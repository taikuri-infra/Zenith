# =============================================================================
# NetworkPolicies — Default Deny + Allow Rules
# =============================================================================
#
# Strategy: deny all ingress by default in application namespaces,
# then allow only what's needed. This prevents lateral movement if
# a pod is compromised.
#
# System namespaces (kube-system, kyverno, cnpg-system) are excluded
# because they need broad access to function.
# =============================================================================

# --- Default Deny Ingress for zenith-staging ---

resource "kubernetes_network_policy_v1" "deny_all_zenith_staging" {
  metadata {
    name      = "default-deny-ingress"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}

# Allow Traefik → zenith-api, zenith-landing, zenith-web, zenith-mc
resource "kubernetes_network_policy_v1" "allow_traefik_to_apps_staging" {
  metadata {
    name      = "allow-traefik-ingress"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {}

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "kube-system"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow zenith-api → zenith-postgres (DB access)
resource "kubernetes_network_policy_v1" "allow_api_to_db_staging" {
  metadata {
    name      = "allow-api-to-postgres"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {
      match_labels = {
        "cnpg.io/cluster" = "zenith-postgres"
      }
    }

    ingress {
      from {
        pod_selector {
          match_labels = {
            app = "zenith-api"
          }
        }
      }

      ports {
        port     = "5432"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow APISIX → zenith-api (gateway proxy)
resource "kubernetes_network_policy_v1" "allow_apisix_to_api_staging" {
  metadata {
    name      = "allow-apisix-to-api"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {
      match_labels = {
        app = "zenith-api"
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "apisix"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow Prometheus → zenith-postgres metrics (CNPG exporter on port 9187)
resource "kubernetes_network_policy_v1" "allow_prometheus_to_cnpg_staging" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-prometheus-to-postgres"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {
      match_labels = {
        "cnpg.io/cluster" = "zenith-postgres"
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "monitoring"
          }
        }
      }

      ports {
        port     = "9187"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow cloudflared tunnel → zenith-staging (MC admin panel)
resource "kubernetes_network_policy_v1" "allow_tunnel_to_staging" {
  metadata {
    name      = "allow-cloudflare-tunnel"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {}

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "cloudflare-tunnel"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# --- Default Deny Ingress for zenith-apps (customer apps) ---

resource "kubernetes_network_policy_v1" "deny_all_zenith_apps" {
  metadata {
    name      = "default-deny-ingress"
    namespace = "zenith-apps"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}

# Allow Traefik → customer apps (HTTP traffic)
resource "kubernetes_network_policy_v1" "allow_traefik_to_customer_apps" {
  metadata {
    name      = "allow-traefik-ingress"
    namespace = "zenith-apps"
  }

  spec {
    pod_selector {}

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "kube-system"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# --- Default Deny Ingress for monitoring ---

resource "kubernetes_network_policy_v1" "deny_all_monitoring" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "default-deny-ingress"
    namespace = "monitoring"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}

# Allow cloudflared tunnel → monitoring services (Grafana, Prometheus, etc.)
resource "kubernetes_network_policy_v1" "allow_tunnel_to_monitoring" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-cloudflare-tunnel"
    namespace = "monitoring"
  }

  spec {
    pod_selector {}

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "cloudflare-tunnel"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow Falco → Loki (falcosidekick sends events to Loki in monitoring)
resource "kubernetes_network_policy_v1" "allow_falco_to_monitoring" {
  count = var.enable_monitoring && var.enable_falco ? 1 : 0

  metadata {
    name      = "allow-falco-to-loki"
    namespace = "monitoring"
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/name" = "loki"
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "falco"
          }
        }
      }

      ports {
        port     = "3100"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow OTel Collector → Tempo (traces pipeline)
resource "kubernetes_network_policy_v1" "allow_otel_to_tempo" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-otel-to-tempo"
    namespace = "monitoring"
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/name" = "tempo"
      }
    }

    ingress {
      from {
        namespace_selector {}
      }

      ports {
        port     = "4317"
        protocol = "TCP"
      }

      ports {
        port     = "4318"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}

# Allow Prometheus scraping from monitoring namespace (self + other namespaces)
resource "kubernetes_network_policy_v1" "allow_prometheus_scrape" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-prometheus-scrape"
    namespace = "monitoring"
  }

  spec {
    pod_selector {}

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "monitoring"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}

# --- Default Deny Ingress for zenith-builds ---

resource "kubernetes_network_policy_v1" "deny_all_zenith_builds" {
  metadata {
    name      = "default-deny-ingress"
    namespace = "zenith-builds"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}

# --- Default Deny Ingress for cloudflare-tunnel ---

resource "kubernetes_network_policy_v1" "deny_all_cloudflare_tunnel" {
  metadata {
    name      = "default-deny-ingress"
    namespace = "cloudflare-tunnel"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}
