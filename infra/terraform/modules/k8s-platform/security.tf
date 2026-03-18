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
# Policy Reporter — Kyverno UI Dashboard (5.13.1)
# =============================================================================

resource "helm_release" "policy_reporter" {
  count = var.enable_kyverno ? 1 : 0

  name             = "policy-reporter"
  repository       = "https://kyverno.github.io/policy-reporter"
  chart            = "policy-reporter"
  version          = var.policy_reporter_version
  namespace        = "kyverno"
  create_namespace = false
  wait             = true
  timeout          = 300

  # Enable the UI plugin
  set {
    name  = "ui.enabled"
    value = "true"
  }

  set {
    name  = "ui.plugins.kyverno"
    value = "true"
  }

  # Kyverno plugin for policy details
  set {
    name  = "kyvernoPlugin.enabled"
    value = "true"
  }

  # Resource limits
  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  set {
    name  = "ui.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "ui.resources.requests.memory"
    value = "64Mi"
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Validate Image Architecture (amd64 only in zenith-apps)
# =============================================================================

resource "kubernetes_manifest" "kyverno_validate_image_arch" {
  count = var.enable_kyverno ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "validate-image-architecture"
      annotations = {
        "policies.kyverno.io/title"       = "Validate Image Architecture"
        "policies.kyverno.io/category"    = "Supply Chain Security"
        "policies.kyverno.io/severity"    = "medium"
        "policies.kyverno.io/description" = "Images deployed to zenith-apps must be built for linux/amd64. This prevents exec format errors from wrong-architecture images."
      }
    }
    spec = {
      validationFailureAction = "Enforce"
      background              = false
      rules = [{
        name = "check-image-arch"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-apps"]
            }
          }]
        }
        exclude = {
          any = [{
            resources = {
              selector = {
                matchLabels = {
                  "zenith.dev/cold-start" = "true"
                }
              }
            }
          }]
        }
        preconditions = {
          all = [{
            key      = "{{request.operation}}"
            operator = "In"
            value    = ["CREATE", "UPDATE"]
          }]
        }
        validate = {
          foreach = [{
            list = "request.object.spec.containers"
            context = [{
              name = "imageData"
              imageRegistry = {
                reference = "{{element.image}}"
              }
            }]
            deny = {
              conditions = {
                any = [{
                  key      = "{{ imageData.configData.architecture }}"
                  operator = "NotEquals"
                  value    = "amd64"
                }]
              }
            }
            elementScope = true
          }]
          message = "One or more container images are not built for linux/amd64. Please rebuild with: docker build --platform linux/amd64"
        }
      }]
    }
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Disallow Privileged Containers
# =============================================================================

resource "kubernetes_manifest" "kyverno_disallow_privileged" {
  count = var.enable_kyverno ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "disallow-privileged-containers"
      annotations = {
        "policies.kyverno.io/title"       = "Disallow Privileged Containers"
        "policies.kyverno.io/category"    = "Pod Security"
        "policies.kyverno.io/severity"    = "high"
        "policies.kyverno.io/description" = "Privileged containers can access the host OS. This policy blocks them in application namespaces."
      }
    }
    spec = {
      validationFailureAction = "Enforce"
      background              = true
      rules = [{
        name = "deny-privileged"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-apps", "zenith-builds"]
            }
          }]
        }
        validate = {
          message = "Privileged containers are not allowed in application namespaces."
          pattern = {
            spec = {
              containers = [{
                securityContext = {
                  privileged = "false | !true"
                }
              }]
            }
          }
        }
      }]
    }
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Require Run As Non-Root
# =============================================================================

resource "kubernetes_manifest" "kyverno_require_non_root" {
  count = var.enable_kyverno ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "require-run-as-non-root"
      annotations = {
        "policies.kyverno.io/title"       = "Require Run As Non-Root"
        "policies.kyverno.io/category"    = "Pod Security"
        "policies.kyverno.io/severity"    = "high"
        "policies.kyverno.io/description" = "Containers must run as non-root user. Prevents privilege escalation attacks."
      }
    }
    spec = {
      validationFailureAction = "Audit"
      background              = true
      rules = [{
        name = "check-non-root"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-apps", "zenith-builds"]
            }
          }]
        }
        validate = {
          message = "Containers must set runAsNonRoot to true."
          pattern = {
            spec = {
              containers = [{
                securityContext = {
                  runAsNonRoot = true
                }
              }]
            }
          }
        }
      }]
    }
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Disallow Host Namespaces
# =============================================================================

resource "kubernetes_manifest" "kyverno_disallow_host_namespaces" {
  count = var.enable_kyverno ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "disallow-host-namespaces"
      annotations = {
        "policies.kyverno.io/title"       = "Disallow Host Namespaces"
        "policies.kyverno.io/category"    = "Pod Security"
        "policies.kyverno.io/severity"    = "high"
        "policies.kyverno.io/description" = "Pods must not use hostNetwork, hostPID, or hostIPC. These give direct access to the host."
      }
    }
    spec = {
      validationFailureAction = "Enforce"
      background              = true
      rules = [{
        name = "deny-host-namespaces"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-apps", "zenith-builds"]
            }
          }]
        }
        validate = {
          message = "Pods cannot use hostNetwork, hostPID, or hostIPC."
          pattern = {
            spec = {
              "=(hostNetwork)" = false
              "=(hostPID)"     = false
              "=(hostIPC)"     = false
            }
          }
        }
      }]
    }
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Require Resource Limits
# =============================================================================

resource "kubernetes_manifest" "kyverno_require_resource_limits" {
  count = var.enable_kyverno ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "require-resource-limits"
      annotations = {
        "policies.kyverno.io/title"       = "Require Resource Limits"
        "policies.kyverno.io/category"    = "Resource Management"
        "policies.kyverno.io/severity"    = "medium"
        "policies.kyverno.io/description" = "All containers must define memory limits. Prevents a single pod from consuming all node resources."
      }
    }
    spec = {
      validationFailureAction = "Audit"
      background              = true
      rules = [{
        name = "check-resource-limits"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-apps", "zenith-builds", "zenith-staging"]
            }
          }]
        }
        validate = {
          message = "All containers must have memory limits defined."
          pattern = {
            spec = {
              containers = [{
                resources = {
                  limits = {
                    memory = "?*"
                  }
                }
              }]
            }
          }
        }
      }]
    }
  }

  depends_on = [helm_release.kyverno]
}

# =============================================================================
# Kyverno Policy — Verify Cosign Image Signatures
# =============================================================================

resource "kubernetes_manifest" "kyverno_verify_image_signatures" {
  count = var.enable_kyverno && var.cosign_public_key != "" ? 1 : 0

  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "kyverno.io/v1"
    kind       = "ClusterPolicy"
    metadata = {
      name = "verify-image-signatures"
      annotations = {
        "policies.kyverno.io/title"       = "Verify Image Signatures"
        "policies.kyverno.io/category"    = "Supply Chain Security"
        "policies.kyverno.io/severity"    = "high"
        "policies.kyverno.io/description" = "Verify that images from the internal registry are signed with Cosign. Set to Audit mode until keys are generated and signing is enabled in CI."
      }
    }
    spec = {
      validationFailureAction = "Audit"
      background              = true
      rules = [{
        name = "verify-cosign-signature"
        match = {
          any = [{
            resources = {
              kinds      = ["Pod"]
              namespaces = ["zenith-staging", "zenith-apps"]
            }
          }]
        }
        verifyImages = [{
          imageReferences = ["registry.stage.freezenith.com/zenith-stage/*"]
          attestors = [{
            entries = [{
              keys = {
                publicKeys = var.cosign_public_key
              }
            }]
          }]
        }]
      }]
    }
  }

  depends_on = [helm_release.kyverno]
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
    name  = "falcosidekick.config.slack.webhookurl"
    value = var.slack_webhook_url
  }

  set {
    name  = "falcosidekick.config.slack.minimumpriority"
    value = "warning"
  }

  set {
    name  = "falcosidekick.config.slack.outputformat"
    value = "all"
  }

  # --- Telegram alerts ---
  set_sensitive {
    name  = "falcosidekick.config.telegram.token"
    value = var.telegram_bot_token
  }

  set {
    name  = "falcosidekick.config.telegram.chatid"
    value = var.telegram_chat_id
    type  = "string"
  }

  set {
    name  = "falcosidekick.config.telegram.minimumpriority"
    value = "warning"
  }

  # Falco metrics for Prometheus scraping
  set {
    name  = "falco.metrics_enabled"
    value = "true"
  }

  set {
    name  = "falco.metrics_interval"
    value = "15s"
  }

  # Loki output for Falco events (logs → Loki for dashboard)
  set {
    name  = "falcosidekick.config.loki.hostport"
    value = "http://loki.monitoring.svc.cluster.local:3100"
  }

  set {
    name  = "falcosidekick.config.loki.minimumpriority"
    value = "notice"
  }

  # --- Custom Falco Rules ---
  values = [yamlencode({
    customRules = {
      "zenith-rules.yaml" = yamlencode([
        {
          rule      = "Detect crypto mining"
          desc      = "Detect crypto mining processes"
          condition = "spawned_process and proc.name in (xmrig, minergate, minerd, cpuminer, cgminer, bfgminer, stratum)"
          output    = "Crypto miner detected (user=%user.name command=%proc.cmdline container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "CRITICAL"
          tags      = ["crypto", "mitre_execution"]
        },
        {
          rule      = "Unauthorized shell in app container"
          desc      = "Detect shell access in application containers"
          condition = "spawned_process and container and proc.name in (bash, sh, zsh, dash, csh, ksh) and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Shell spawned in app container (user=%user.name shell=%proc.name container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["shell", "mitre_execution"]
        },
        {
          rule      = "Sensitive file access in container"
          desc      = "Detect access to sensitive files"
          condition = "open_read and container and fd.name in (/etc/shadow, /etc/passwd, /etc/sudoers, /root/.ssh/authorized_keys, /root/.bash_history) and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Sensitive file read (file=%fd.name user=%user.name container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["filesystem", "mitre_discovery"]
        },
        {
          rule      = "Outbound connection to unusual port"
          desc      = "Detect outbound connections to non-standard ports from app containers"
          condition = "outbound and container and not fd.sport in (53, 80, 443, 5432, 6379, 8080, 8443, 9090, 27017) and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Unusual outbound connection (port=%fd.sport ip=%fd.sip container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["network", "mitre_command_and_control"]
        },
        {
          rule      = "Package manager in container"
          desc      = "Detect package manager usage in running containers"
          condition = "spawned_process and container and proc.name in (apt, apt-get, yum, dnf, apk, pip, npm, gem) and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Package manager run in container (command=%proc.cmdline container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["package", "mitre_persistence"]
        },
        {
          rule      = "Reverse shell detected"
          desc      = "Detect potential reverse shell connections"
          condition = "spawned_process and container and proc.name in (bash, sh, zsh, nc, ncat) and (proc.cmdline contains '/dev/tcp' or proc.cmdline contains 'nc -e' or proc.cmdline contains 'ncat -e') and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Reverse shell detected (user=%user.name command=%proc.cmdline container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "CRITICAL"
          tags      = ["reverse_shell", "mitre_execution"]
        },
        {
          rule      = "Binary written to /tmp"
          desc      = "Detect binaries dropped into tmp directories"
          condition = "((evt.type = open and evt.is_open_write = true) or evt.type = creat) and container and (fd.name startswith /tmp or fd.name startswith /var/tmp) and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Binary written to temp dir (file=%fd.name user=%user.name container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["filesystem", "mitre_defense_evasion"]
        },
        {
          rule      = "Kubernetes service account token access"
          desc      = "Detect access to service account tokens"
          condition = "open_read and container and fd.name startswith /var/run/secrets/kubernetes.io and k8s.ns.name in (zenith-apps, zenith-builds)"
          output    = "Service account token read (file=%fd.name user=%user.name container=%container.name namespace=%k8s.ns.name pod=%k8s.pod.name)"
          priority  = "WARNING"
          tags      = ["k8s", "mitre_credential_access"]
        },
      ])
    }
  })]

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
