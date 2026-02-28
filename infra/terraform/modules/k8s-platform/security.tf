# =============================================================================
# Kyverno — Admission Policy Engine (5.13)
# =============================================================================

resource "helm_release" "kyverno" {
  count = var.enable_kyverno ? 1 : 0

  name             = "kyverno"
  repository       = "https://kyverno.github.io/kyverno"
  chart            = "kyverno"
  version          = var.kyverno_version
  namespace        = "kyverno"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "replicaCount"
    value = "1"
  }

  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }
}

# =============================================================================
# Falco — Runtime Security Detection (5.14)
# =============================================================================

resource "helm_release" "falco" {
  count = var.enable_falco ? 1 : 0

  name             = "falco"
  repository       = "https://falcosecurity.github.io/charts"
  chart            = "falco"
  version          = var.falco_version
  namespace        = "falco"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "driver.kind"
    value = "auto"
  }

  set {
    name  = "falcosidekick.enabled"
    value = "true"
  }

  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }
}

# =============================================================================
# Velero — Cluster Backup to Hetzner S3 (5.15)
# =============================================================================

resource "helm_release" "velero" {
  count = var.enable_velero ? 1 : 0

  name             = "velero"
  repository       = "https://vmware-tanzu.github.io/helm-charts"
  chart            = "velero"
  version          = var.velero_version
  namespace        = "velero"
  create_namespace = true
  wait             = true
  timeout          = 600

  # AWS plugin for S3-compatible storage
  set {
    name  = "initContainers[0].name"
    value = "velero-plugin-for-aws"
  }

  set {
    name  = "initContainers[0].image"
    value = "velero/velero-plugin-for-aws:v1.11.0"
  }

  set {
    name  = "initContainers[0].volumeMounts[0].name"
    value = "plugins"
  }

  set {
    name  = "initContainers[0].volumeMounts[0].mountPath"
    value = "/target"
  }

  set {
    name  = "configuration.backupStorageLocation[0].name"
    value = "default"
  }

  set {
    name  = "configuration.backupStorageLocation[0].provider"
    value = "aws"
  }

  set {
    name  = "configuration.backupStorageLocation[0].bucket"
    value = "zenith-backups"
  }

  set {
    name  = "configuration.backupStorageLocation[0].prefix"
    value = "velero"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.region"
    value = "fsn1"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.s3ForcePathStyle"
    value = "true"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.s3Url"
    value = var.s3_endpoint
  }

  set_sensitive {
    name  = "credentials.secretContents.cloud"
    value = "[default]\naws_access_key_id=${var.s3_access_key}\naws_secret_access_key=${var.s3_secret_key}\n"
  }

  # Daily backup at 03:00 UTC, 30-day retention
  set {
    name  = "schedules.daily-backup.disabled"
    value = "false"
  }

  set {
    name  = "schedules.daily-backup.schedule"
    value = "0 3 * * *"
  }

  set {
    name  = "schedules.daily-backup.template.ttl"
    value = "720h"
  }

  set {
    name  = "schedules.daily-backup.template.excludedNamespaces[0]"
    value = "velero"
  }

  # Skip CRD upgrade job (CRDs already installed, bitnami kubectl images are purged)
  set {
    name  = "upgradeCRDs"
    value = "false"
  }

  # VolumeSnapshotLocation requires a provider
  set {
    name  = "configuration.volumeSnapshotLocation[0].name"
    value = "default"
  }

  set {
    name  = "configuration.volumeSnapshotLocation[0].provider"
    value = "aws"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }
}
