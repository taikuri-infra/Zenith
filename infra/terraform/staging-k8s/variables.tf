variable "kubeconfig_path" {
  description = "Path to kubeconfig file for the staging k3s cluster"
  type        = string
  default     = "~/.kube/zenith-staging.yaml"
}

# --- Registry ---

variable "registry_host" {
  description = "Harbor registry host"
  type        = string
  default     = "registry.stage.freezenith.com"
}

variable "registry_username" {
  description = "Harbor robot account username"
  type        = string
  sensitive   = true
}

variable "registry_password" {
  description = "Harbor robot account password"
  type        = string
  sensitive   = true
}

variable "zenith_chart_version" {
  description = "Zenith Helm chart version to deploy"
  type        = string
  default     = "0.2.0"
}

# --- App Secrets ---

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

