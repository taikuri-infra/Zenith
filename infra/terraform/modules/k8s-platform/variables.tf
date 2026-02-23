# --- Helm Chart Paths ---

variable "zenith_chart_path" {
  description = "Path to the zenith Helm chart"
  type        = string
}

variable "zenith_values_file" {
  description = "Path to the zenith values file (e.g. values-staging.yaml)"
  type        = string
}

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

variable "db_password" {
  description = "PostgreSQL password"
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
