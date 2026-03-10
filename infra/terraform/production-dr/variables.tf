variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key for server access"
  type        = string
}

variable "server_type" {
  description = "Hetzner server type for DR node"
  type        = string
  default     = "cx32" # 4 vCPU, 8GB RAM — minimal for DR standby
}

variable "k3s_version" {
  description = "k3s version to install"
  type        = string
  default     = "v1.34.3+k3s1"
}

variable "allowed_ssh_ips" {
  description = "CIDR blocks allowed to SSH into DR server"
  type        = list(string)
  default     = ["0.0.0.0/0", "::/0"]
}

# --- S3 / Object Storage ---

variable "s3_access_key" {
  description = "Hetzner S3 access key (shared with production)"
  type        = string
  sensitive   = true
}

variable "s3_secret_key" {
  description = "Hetzner S3 secret key"
  type        = string
  sensitive   = true
}

variable "s3_endpoint" {
  description = "Hetzner S3 endpoint URL"
  type        = string
  default     = "https://fsn1.your-objectstorage.com"
}

variable "velero_bucket" {
  description = "S3 bucket name for Velero backups (shared with production)"
  type        = string
  default     = "zenith-backups"
}

# --- Production primary (for CNPG replication) ---

variable "production_pg_host" {
  description = "Production PostgreSQL primary host for streaming replication"
  type        = string
  default     = ""
}

variable "production_pg_port" {
  description = "Production PostgreSQL primary port"
  type        = number
  default     = 5432
}
