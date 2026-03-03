# --- Helm Chart Source (shared across all zenith charts) ---

variable "chart_repository" {
  description = "OCI repository for zenith charts (e.g. oci://registry.stage.freezenith.com/zenith-stage)"
  type        = string
  default     = ""
}

variable "chart_version" {
  description = "Chart version to deploy (required when using OCI repository)"
  type        = string
  default     = ""
}

# --- Per-chart local paths (used when chart_repository is empty) ---

variable "platform_chart_path" {
  description = "Local path to the zenith-platform Helm chart"
  type        = string
  default     = ""
}

variable "api_chart_path" {
  description = "Local path to the zenith-api Helm chart"
  type        = string
  default     = ""
}

variable "landing_chart_path" {
  description = "Local path to the zenith-landing Helm chart"
  type        = string
  default     = ""
}

variable "demo_chart_path" {
  description = "Local path to the zenith-demo Helm chart"
  type        = string
  default     = ""
}

variable "tenant_chart_path" {
  description = "Local path to the zenith-tenant Helm chart"
  type        = string
  default     = ""
}

# --- Per-chart values files ---

variable "platform_values_file" {
  description = "Path to the zenith-platform values file"
  type        = string
}

variable "api_values_file" {
  description = "Path to the zenith-api values file"
  type        = string
}

variable "landing_values_file" {
  description = "Path to the zenith-landing values file"
  type        = string
}

variable "demo_values_file" {
  description = "Path to the zenith-demo values file"
  type        = string
  default     = ""
}

variable "tenant_values_file" {
  description = "Path to the zenith-tenant values file"
  type        = string
  default     = ""
}

# --- Registry Credentials (for imagePullSecret) ---

variable "registry_host" {
  description = "Container registry host (e.g. registry.stage.freezenith.com)"
  type        = string
  default     = ""
}

variable "registry_username" {
  description = "Container registry username"
  type        = string
  sensitive   = true
  default     = ""
}

variable "registry_password" {
  description = "Container registry password"
  type        = string
  sensitive   = true
  default     = ""
}

# --- Monitoring ---

variable "monitoring_chart_path" {
  description = "Path to the monitoring Helm chart"
  type        = string
  default     = ""
}

variable "monitoring_values_file" {
  description = "Path to the monitoring values file"
  type        = string
  default     = ""
}

variable "monitoring_domain" {
  description = "Base domain for monitoring IngressRoutes (grafana.<domain>, prometheus.<domain>)"
  type        = string
  default     = ""
}

# --- Platform Config ---

variable "platform_namespace" {
  description = "Kubernetes namespace for the platform"
  type        = string
}

variable "cert_issuer_email" {
  description = "Email for Let's Encrypt certificate issuer"
  type        = string
  default     = "admin@freezenith.com"
}

# --- Secrets ---

variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}

variable "admin_email" {
  description = "Admin user email"
  type        = string
  sensitive   = true
}

variable "admin_password" {
  description = "Admin user password"
  type        = string
  sensitive   = true
}

variable "github_webhook_secret" {
  description = "HMAC secret for verifying GitHub webhook signatures"
  type        = string
  sensitive   = true
  default     = ""
}

variable "secrets_encryption_key" {
  description = "64-char hex (32 bytes) AES-256-GCM key for encrypting app secrets"
  type        = string
  sensitive   = true
  default     = ""
}

variable "resend_api_key" {
  description = "Resend API key for email verification"
  type        = string
  sensitive   = true
  default     = ""
}

variable "google_client_id" {
  description = "Google OAuth client ID for login"
  type        = string
  sensitive   = true
  default     = ""
}

# --- cert-manager ---

variable "cert_manager_version" {
  description = "cert-manager Helm chart version"
  type        = string
  default     = "v1.17.2"
}

# --- Feature Flags ---

variable "enable_keda" {
  description = "Install KEDA for scale-to-zero"
  type        = bool
  default     = false
}

variable "enable_monitoring" {
  description = "Install monitoring stack (Prometheus + Grafana + Loki)"
  type        = bool
  default     = false
}

variable "enable_demo" {
  description = "Deploy demo MC + Web instances"
  type        = bool
  default     = false
}

variable "enable_tenants" {
  description = "Deploy tenant MC + Web instances"
  type        = bool
  default     = false
}

# --- Hetzner CSI ---

variable "hcloud_token" {
  description = "Hetzner Cloud API token for CSI driver volume provisioning"
  type        = string
  sensitive   = true
}

variable "hcloud_csi_version" {
  description = "Hetzner CSI driver Helm chart version"
  type        = string
  default     = "2.20.0"
}

# --- CloudNativePG ---

variable "enable_cnpg" {
  description = "Install CloudNativePG operator for PostgreSQL"
  type        = bool
  default     = false
}

variable "cnpg_version" {
  description = "CloudNativePG Helm chart version"
  type        = string
  default     = "0.23.0"
}

# --- Kong ---

variable "enable_kong" {
  description = "Install Kong API Gateway"
  type        = bool
  default     = false
}

variable "kong_version" {
  description = "Kong Ingress Controller Helm chart version"
  type        = string
  default     = "0.16.0"
}

# --- KEDA ---

variable "keda_version" {
  description = "KEDA Helm chart version"
  type        = string
  default     = "2.16.0"
}

variable "keda_http_addon_version" {
  description = "KEDA HTTP Add-on Helm chart version"
  type        = string
  default     = "0.9.0"
}

# --- V2 Variables ---

variable "environment" {
  description = "Environment name (staging or production)"
  type        = string
  default     = "staging"
}

variable "domain" {
  description = "The root domain used for DNS routing and certificates (e.g. freezenith.com)"
  type        = string
}

variable "cluster_domain" {
  description = "The specific environment domain used for ingress (e.g. stage.freezenith.com)"
  type        = string
}

variable "enable_sealed_secrets" {
  description = "Enable Bitnami Sealed Secrets"
  type        = bool
  default     = true
}

variable "sealed_secrets_version" {
  description = "Sealed Secrets chart version"
  type        = string
  default     = "2.16.1"
}

variable "enable_keycloak" {
  description = "Enable Keycloak identity provider"
  type        = bool
  default     = true
}

variable "enable_kyverno" {
  description = "Enable Kyverno admission controller"
  type        = bool
  default     = true
}

variable "kyverno_version" {
  description = "Kyverno chart version"
  type        = string
  default     = "3.7.1"
}

variable "enable_falco" {
  description = "Enable Falco runtime security"
  type        = bool
  default     = true
}

variable "falco_version" {
  description = "Falco chart version"
  type        = string
  default     = "4.18.0"
}

variable "enable_velero" {
  description = "Enable Velero cluster backups"
  type        = bool
  default     = true
}

variable "velero_version" {
  description = "Velero chart version"
  type        = string
  default     = "11.4.0"
}

variable "prometheus_stack_version" {
  description = "Kube Prometheus Stack version"
  type        = string
  default     = "61.3.1"
}

variable "s3_access_key" {
  description = "Hetzner Object Storage Access Key"
  type        = string
  sensitive   = true
}

variable "s3_secret_key" {
  description = "Hetzner Object Storage Secret Key"
  type        = string
  sensitive   = true
}

variable "s3_endpoint" {
  description = "Hetzner Object Storage Endpoint"
  type        = string
  default     = "https://fsn1.your-objectstorage.com"
}

# --- Keycloak ---

variable "keycloak_version" {
  description = "Keycloak Helm chart version"
  type        = string
  default     = "25.2.0"
}

variable "keycloak_db_password" {
  description = "Password for Keycloak CNPG database"
  type        = string
  sensitive   = true
}

variable "keycloak_admin_password" {
  description = "Keycloak admin console password"
  type        = string
  sensitive   = true
}

variable "keycloak_db_storage_size" {
  description = "Storage size for Keycloak dedicated CNPG cluster"
  type        = string
  default     = "10Gi"
}

variable "free_db_storage_size" {
  description = "Storage size for the shared free-tier CNPG cluster"
  type        = string
  default     = "20Gi"
}

# --- APISIX ---

variable "enable_apisix" {
  description = "Enable APISIX API Gateway (replaces Kong)"
  type        = bool
  default     = true
}

variable "apisix_version" {
  description = "APISIX Helm chart version"
  type        = string
  default     = "2.13.0"
}

variable "apisix_ingress_version" {
  description = "APISIX Ingress Controller chart version"
  type        = string
  default     = "0.14.0"
}

# --- external-dns ---

variable "enable_external_dns" {
  description = "Enable external-dns for automatic Cloudflare DNS"
  type        = bool
  default     = true
}

variable "external_dns_version" {
  description = "external-dns Helm chart version"
  type        = string
  default     = "9.0.3"
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token for DNS-01 and external-dns"
  type        = string
  sensitive   = true
}

# --- ArgoCD ---

variable "enable_argocd" {
  description = "Enable ArgoCD GitOps engine"
  type        = bool
  default     = true
}

variable "argocd_version" {
  description = "ArgoCD Helm chart version"
  type        = string
  default     = "7.3.11"
}

variable "argocd_image_updater_version" {
  description = "ArgoCD Image Updater chart version"
  type        = string
  default     = "0.11.0"
}

variable "argocd_target_revision" {
  description = "Git branch/tag ArgoCD watches for app manifests"
  type        = string
  default     = "staging"
}

variable "github_token" {
  description = "GitHub personal access token for ArgoCD repo access"
  type        = string
  sensitive   = true
  default     = ""
}

# --- Harbor ---

variable "enable_harbor" {
  description = "Enable Harbor container registry"
  type        = bool
  default     = true
}

variable "harbor_version" {
  description = "Harbor Helm chart version"
  type        = string
  default     = "1.15.1"
}

variable "customer_registry_host" {
  description = "Public-facing registry host for pro-tier customers (e.g. hub.stage.freezenith.com)"
  type        = string
  default     = "hub.stage.freezenith.com"
}

# --- Temporal ---

variable "enable_temporal" {
  description = "Enable Temporal workflow engine"
  type        = bool
  default     = true
}

variable "temporal_version" {
  description = "Temporal Helm chart version"
  type        = string
  default     = "0.45.0"
}

variable "temporal_db_user" {
  description = "Temporal database user"
  type        = string
  sensitive   = true
}

variable "temporal_db_password" {
  description = "Temporal database password"
  type        = string
  sensitive   = true
}

# --- Observability (Loki / Tempo / OTel) ---

variable "loki_version" {
  description = "Loki Helm chart version"
  type        = string
  default     = "6.6.4"
}

variable "tempo_version" {
  description = "Tempo Helm chart version"
  type        = string
  default     = "1.10.1"
}

variable "otel_collector_version" {
  description = "OpenTelemetry Collector Helm chart version"
  type        = string
  default     = "0.96.0"
}
