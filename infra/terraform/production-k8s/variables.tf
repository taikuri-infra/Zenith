variable "kubeconfig_path" {
  description = "Path to kubeconfig file for the production k3s cluster"
  type        = string
  default     = "~/.kube/zenith-production.yaml"
}

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

variable "enable_keda" {
  description = "Install KEDA for scale-to-zero"
  type        = bool
  default     = true
}

variable "enable_monitoring" {
  description = "Install monitoring stack"
  type        = bool
  default     = true
}
