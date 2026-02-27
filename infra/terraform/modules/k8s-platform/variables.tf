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
