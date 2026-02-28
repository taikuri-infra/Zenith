# =============================================================================
# ArgoCD — GitOps Engine (5.10)
# =============================================================================

resource "helm_release" "argocd" {
  count = var.enable_argocd ? 1 : 0

  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "server.replicas"
    value = "1"
  }

  set {
    name  = "server.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "server.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "repoServer.replicas"
    value = "1"
  }

  set {
    name  = "repoServer.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "repoServer.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "controller.resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "controller.resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "server.extensions.enabled"
    value = "true"
  }

  set {
    name  = "configs.params.server\\.insecure"
    value = "true"
  }

  # --- Git repository credentials for ArgoCD ---
  set {
    name  = "configs.repositories.zenith.url"
    value = "https://github.com/taikuri-infra/Zenith.git"
  }

  set {
    name  = "configs.repositories.zenith.type"
    value = "git"
  }

  set {
    name  = "configs.repositories.zenith.username"
    value = "argocd"
  }

  set_sensitive {
    name  = "configs.repositories.zenith.password"
    value = var.github_token
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
}

resource "helm_release" "argocd_image_updater" {
  count = var.enable_argocd ? 1 : 0

  name       = "argocd-image-updater"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argocd-image-updater"
  version    = var.argocd_image_updater_version
  namespace  = "argocd"
  wait       = true
  timeout    = 300

  set {
    name  = "config.registries[0].name"
    value = "harbor"
  }

  set {
    name  = "config.registries[0].prefix"
    value = var.registry_host
  }

  set {
    name  = "config.registries[0].api_url"
    value = "https://${var.registry_host}"
  }

  set {
    name  = "config.registries[0].default"
    value = "true"
  }

  set {
    name  = "config.registries[0].credentials"
    value = "secret:argocd/harbor-image-updater-creds#token"
  }

  depends_on = [helm_release.argocd]
}

# =============================================================================
# Image Updater — Harbor credentials for internal registry
# =============================================================================

resource "kubernetes_secret_v1" "harbor_image_updater_creds" {
  count = var.enable_argocd ? 1 : 0

  metadata {
    name      = "harbor-image-updater-creds"
    namespace = "argocd"
  }

  data = {
    token = "${var.registry_username}:${var.registry_password}"
  }

  depends_on = [helm_release.argocd]
}

# =============================================================================
# ArgoCD TLS — Certificate + IngressRoute (5.10b)
# =============================================================================

resource "kubernetes_manifest" "argocd_certificate" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "Certificate"
    metadata = {
      name      = "argocd-tls"
      namespace = "argocd"
    }
    spec = {
      secretName = "argocd-tls"
      issuerRef = {
        name = "letsencrypt-prod"
        kind = "ClusterIssuer"
      }
      dnsNames = [
        "argocd.${var.cluster_domain}",
      ]
    }
  }

  depends_on = [helm_release.argocd]
}

resource "kubernetes_manifest" "argocd_ingressroute" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "traefik.io/v1alpha1"
    kind       = "IngressRoute"
    metadata = {
      name      = "argocd"
      namespace = "argocd"
    }
    spec = {
      entryPoints = ["websecure"]
      routes = [{
        match = "Host(`argocd.${var.cluster_domain}`)"
        kind  = "Rule"
        services = [{
          name = "argocd-server"
          port = 80
        }]
      }]
      tls = {
        secretName = "argocd-tls"
      }
    }
  }

  depends_on = [helm_release.argocd]
}

# =============================================================================
# ArgoCD — AppProject for tenant isolation (5.10c)
# =============================================================================

resource "kubernetes_manifest" "argocd_project_tenant_apps" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "AppProject"
    metadata = {
      name      = "tenant-apps"
      namespace = "argocd"
    }
    spec = {
      description = "Tenant application deployments — restricted to zenith-* namespaces"
      sourceRepos = [
        "https://github.com/taikuri-infra/Zenith.git",
      ]
      destinations = [{
        server    = "https://kubernetes.default.svc"
        namespace = "zenith-*"
      }]
      clusterResourceWhitelist = []
    }
  }

  depends_on = [helm_release.argocd]
}

# =============================================================================
# ArgoCD — Root Application (App-of-Apps bootstrap) (5.10d)
# =============================================================================

resource "kubernetes_manifest" "argocd_root_application" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "zenith-apps"
      namespace = "argocd"
      finalizers = [
        "resources-finalizer.argocd.argoproj.io",
      ]
    }
    spec = {
      project = "default"
      source = {
        repoURL        = "https://github.com/taikuri-infra/Zenith.git"
        targetRevision = var.argocd_target_revision
        path           = "infra/argocd/staging"
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "argocd"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
      }
    }
  }

  depends_on = [helm_release.argocd]
}
