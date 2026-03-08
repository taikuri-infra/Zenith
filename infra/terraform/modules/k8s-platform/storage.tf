# =============================================================================
# Hetzner CSI Driver — Block Storage Provisioner (5.3)
# =============================================================================

resource "helm_release" "hcloud_csi" {
  name       = "hcloud-csi"
  repository = "https://charts.hetzner.cloud"
  chart      = "hcloud-csi"
  version    = var.hcloud_csi_version
  namespace  = "kube-system"
  wait       = true
  timeout    = 300

  set_sensitive {
    name  = "controller.hcloudToken.value"
    value = var.hcloud_token
  }

  set {
    name  = "storageClasses[0].name"
    value = "hcloud-volumes"
  }

  set {
    name  = "storageClasses[0].defaultStorageClass"
    value = "false"
  }

  set {
    name  = "storageClasses[0].reclaimPolicy"
    value = "Retain"
  }
}

# =============================================================================
# CloudNativePG — PostgreSQL Operator (5.4)
# =============================================================================

resource "helm_release" "cnpg" {
  count = var.enable_cnpg ? 1 : 0

  name             = "cnpg"
  repository       = "https://cloudnative-pg.github.io/charts"
  chart            = "cloudnative-pg"
  version          = var.cnpg_version
  namespace        = "cnpg-system"
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

  set {
    name  = "resources.limits.cpu"
    value = "200m"
  }

  set {
    name  = "resources.limits.memory"
    value = "256Mi"
  }

  # V2: Inherit cert-manager annotations and app labels
  set {
    name  = "config.data.INHERITED_ANNOTATIONS"
    value = "cert-manager.io/*"
  }

  set {
    name  = "config.data.INHERITED_LABELS"
    value = "app.kubernetes.io/*"
  }

  depends_on = [helm_release.cert_manager]
}

# =============================================================================
# CNPG Cluster — Dedicated PostgreSQL for Keycloak (5.5)
# =============================================================================

resource "kubernetes_namespace" "keycloak" {
  count = var.enable_keycloak ? 1 : 0
  metadata {
    name = "keycloak"
  }
}

resource "kubernetes_secret" "cnpg_s3_credentials_keycloak" {
  count = var.enable_keycloak ? 1 : 0

  metadata {
    name      = "cnpg-s3-credentials"
    namespace = "keycloak"
  }

  data = {
    ACCESS_KEY_ID     = var.s3_access_key
    ACCESS_SECRET_KEY = var.s3_secret_key
  }

  depends_on = [kubernetes_namespace.keycloak]
}

resource "kubernetes_manifest" "cnpg_keycloak" {
  count = var.enable_keycloak ? 1 : 0

  # CNPG operator injects default PostgreSQL parameters server-side
  computed_fields = ["spec.postgresql.parameters"]

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "keycloak-pg"
      namespace = "keycloak"
    }
    spec = {
      instances             = var.environment == "production" ? 3 : 2
      primaryUpdateStrategy = "unsupervised"

      storage = {
        storageClass = "hcloud-volumes"
        size         = var.keycloak_db_storage_size
      }

      postgresql = {
        parameters = {
          max_connections      = "100"
          shared_buffers       = "128MB"
          effective_cache_size = "256MB"
          work_mem             = "4MB"
          maintenance_work_mem = "64MB"
        }
      }

      bootstrap = {
        initdb = {
          database = "keycloak"
          owner    = "keycloak"
        }
      }

      backup = {
        barmanObjectStore = {
          destinationPath = "s3://zenith-backups/keycloak-wal/"
          endpointURL     = var.s3_endpoint
          s3Credentials = {
            accessKeyId = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_KEY_ID"
            }
            secretAccessKey = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_SECRET_KEY"
            }
          }
          wal = {
            compression = "gzip"
            maxParallel = 4
          }
        }
        retentionPolicy = "14d"
      }

      monitoring = {
        enablePodMonitor = true
      }

      priorityClassName = "infra-critical"
    }
  }

  depends_on = [
    helm_release.cnpg,
    helm_release.hcloud_csi,
    kubernetes_namespace.keycloak,
    kubernetes_priority_class.infra_critical,
  ]
}

# =============================================================================
# CNPG Cluster — Shared PostgreSQL for Free-tier customers (5.6)
# =============================================================================

resource "kubernetes_namespace" "zenith_shared" {
  metadata {
    name = "zenith-shared"
  }
}

resource "kubernetes_secret" "cnpg_s3_credentials_shared" {
  metadata {
    name      = "cnpg-s3-credentials"
    namespace = "zenith-shared"
  }

  data = {
    ACCESS_KEY_ID     = var.s3_access_key
    ACCESS_SECRET_KEY = var.s3_secret_key
  }

  depends_on = [kubernetes_namespace.zenith_shared]
}

resource "kubernetes_manifest" "cnpg_free" {
  # CNPG operator injects default PostgreSQL parameters server-side
  computed_fields = ["spec.postgresql.parameters"]

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "free-pg"
      namespace = "zenith-shared"
    }
    spec = {
      instances             = var.environment == "production" ? 3 : 2
      primaryUpdateStrategy = "unsupervised"
      enableSuperuserAccess = true

      storage = {
        storageClass = "hcloud-volumes"
        size         = var.free_db_storage_size
      }

      postgresql = {
        parameters = {
          max_connections      = "200"
          shared_buffers       = "256MB"
          effective_cache_size = "512MB"
          work_mem             = "4MB"
          maintenance_work_mem = "64MB"
          statement_timeout    = "30000"
        }
      }

      bootstrap = {
        initdb = {
          database = "zenith_platform"
          owner    = "zenith_admin"
          postInitSQL = [
            "CREATE ROLE temporal WITH LOGIN PASSWORD '${var.temporal_db_password}' CREATEDB",
            "CREATE DATABASE temporal OWNER temporal",
            "CREATE DATABASE temporal_visibility OWNER temporal",
          ]
        }
      }

      backup = {
        barmanObjectStore = {
          destinationPath = "s3://zenith-backups/free-pg-wal/"
          endpointURL     = var.s3_endpoint
          s3Credentials = {
            accessKeyId = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_KEY_ID"
            }
            secretAccessKey = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_SECRET_KEY"
            }
          }
          wal = {
            compression = "gzip"
            maxParallel = 4
          }
        }
        retentionPolicy = "14d"
      }

      monitoring = {
        enablePodMonitor = true
      }

      priorityClassName = "infra-critical"
    }
  }

  depends_on = [
    helm_release.cnpg,
    helm_release.hcloud_csi,
    kubernetes_namespace.zenith_shared,
    kubernetes_priority_class.infra_critical,
    kubernetes_secret.cnpg_s3_credentials_shared,
  ]
}

# =============================================================================
# CNPG Scheduled Backups — daily base backups to S3
# =============================================================================

resource "kubernetes_manifest" "cnpg_backup_keycloak" {
  count = var.enable_keycloak ? 1 : 0

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "ScheduledBackup"
    metadata = {
      name      = "keycloak-pg-daily"
      namespace = "keycloak"
    }
    spec = {
      schedule             = "0 0 2 * * *"
      backupOwnerReference = "self"
      cluster = {
        name = "keycloak-pg"
      }
      method = "barmanObjectStore"
      target = "prefer-standby"
    }
  }

  depends_on = [kubernetes_manifest.cnpg_keycloak]
}

resource "kubernetes_manifest" "cnpg_backup_free_pg" {
  count = var.enable_cnpg ? 1 : 0

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "ScheduledBackup"
    metadata = {
      name      = "free-pg-daily"
      namespace = "zenith-shared"
    }
    spec = {
      schedule             = "0 0 2 * * *"
      backupOwnerReference = "self"
      cluster = {
        name = "free-pg"
      }
      method = "barmanObjectStore"
      target = "prefer-standby"
    }
  }

  depends_on = [kubernetes_manifest.cnpg_free]
}
