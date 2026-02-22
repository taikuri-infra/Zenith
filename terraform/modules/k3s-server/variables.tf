variable "name" {
  description = "Server name"
  type        = string
}

variable "server_type" {
  description = "Hetzner server type (cx22, cx32, cx42, etc.)"
  type        = string
  default     = "cx22"
}

variable "image" {
  description = "OS image"
  type        = string
  default     = "ubuntu-24.04"
}

variable "location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "nbg1"
}

variable "environment" {
  description = "Environment name (staging, production)"
  type        = string
}

variable "role" {
  description = "Server role (management, cluster, all-in-one)"
  type        = string
  default     = "all-in-one"
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
}

variable "ssh_allowed_ips" {
  description = "IPs allowed for SSH and k3s API access"
  type        = list(string)
  default     = ["0.0.0.0/0", "::/0"]
}

variable "user_data" {
  description = "Cloud-init user data script"
  type        = string
  default     = ""
}

variable "extra_labels" {
  description = "Additional labels for the server"
  type        = map(string)
  default     = {}
}
