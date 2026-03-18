# =============================================================================
# V3 Identity & Data Operators — Keycloak Operator, Redis Operator, MongoDB Operator
# =============================================================================
# P2 migration: Operator-managed Keycloak, Redis, and MongoDB
# Enable with enable_v3_operators = true
# =============================================================================

# --- Keycloak Operator (P2-01) ---

resource "helm_release" "keycloak_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "keycloak-operator"
  repository       = "oci://quay.io/keycloak/keycloak-operator"
  chart            = "keycloak-operator"
  version          = var.keycloak_operator_version
  namespace        = "keycloak"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }
}

# --- Keycloak CRD instance (P2-02) ---
# Points to existing CNPG database for zero-downtime migration

resource "kubernetes_manifest" "keycloak_instance" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "k8s.keycloak.org/v2alpha1"
    kind       = "Keycloak"
    metadata = {
      name      = "zenith-keycloak"
      namespace = "keycloak"
    }
    spec = {
      instances = 1
      hostname = {
        hostname = "auth.${var.cluster_domain}"
      }
      db = {
        vendor   = "postgres"
        host     = "keycloak-db-rw.keycloak.svc.cluster.local"
        port     = 5432
        database = "keycloak"
        usernameSecret = {
          name = "keycloak-db-credentials"
          key  = "username"
        }
        passwordSecret = {
          name = "keycloak-db-credentials"
          key  = "password"
        }
      }
      http = {
        httpEnabled = true
      }
      proxy = {
        headers = "xforwarded"
      }
      resources = {
        requests = {
          cpu    = "200m"
          memory = "512Mi"
        }
        limits = {
          memory = "1Gi"
        }
      }
    }
  }

  depends_on = [helm_release.keycloak_operator]
}

# --- Keycloak DB Credentials Secret (for Operator CRD) ---

resource "kubernetes_secret" "keycloak_db_credentials" {
  count = var.enable_v3_operators ? 1 : 0

  metadata {
    name      = "keycloak-db-credentials"
    namespace = "keycloak"
  }

  data = {
    username = "keycloak"
    password = var.keycloak_db_password
  }
}

# --- KeycloakRealmImport CRDs (P2-03, P2-04) ---
# Import existing realms. These are applied as CRDs and the operator handles import.

resource "kubernetes_manifest" "keycloak_realm_zenith" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "k8s.keycloak.org/v2alpha1"
    kind       = "KeycloakRealmImport"
    metadata = {
      name      = "zenith-realm"
      namespace = "keycloak"
    }
    spec = {
      keycloakCRName = "zenith-keycloak"
      realm = {
        realm   = "zenith"
        enabled = true
        clients = [{
          clientId                  = "zenith-api"
          publicClient              = true
          directAccessGrantsEnabled = true
          redirectUris              = ["https://app.${var.cluster_domain}/*", "https://admin.${var.cluster_domain}/*"]
          webOrigins                = ["https://app.${var.cluster_domain}", "https://admin.${var.cluster_domain}"]
        }]
      }
    }
  }

  depends_on = [kubernetes_manifest.keycloak_instance]
}

# --- Redis Operator — OpsTree (P2-08) ---

resource "helm_release" "redis_operator" {
  count = var.enable_redis_operator || var.enable_v3_operators ? 1 : 0

  name             = "redis-operator"
  repository       = "https://ot-container-kit.github.io/helm-charts/"
  chart            = "redis-operator"
  version          = var.redis_operator_version
  namespace        = "redis-operator"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }
}

# --- MongoDB Operator — Percona (P2-13) ---

resource "helm_release" "mongodb_operator" {
  count = var.enable_mongodb_operator || var.enable_v3_operators ? 1 : 0

  name             = "psmdb-operator"
  repository       = "https://percona.github.io/percona-helm-charts"
  chart            = "psmdb-operator"
  version          = var.mongodb_operator_version
  namespace        = "mongodb-operator"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "resources.requests.memory"
    value = "64Mi"
  }

  # Disable multicluster service discovery (crashes with stale KEDA metrics API)
  set {
    name  = "env[0].name"
    value = "DISABLE_TELEMETRY"
  }

  set {
    name  = "env[0].value"
    value = "true"
  }
}
