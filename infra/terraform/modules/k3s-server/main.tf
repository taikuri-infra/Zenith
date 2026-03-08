terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.49"
    }
  }
}

# SSH key for server access
resource "hcloud_ssh_key" "zenith" {
  name       = "${var.name}-ssh-key"
  public_key = var.ssh_public_key

  lifecycle {
    ignore_changes = [public_key]
  }
}

# Firewall
resource "hcloud_firewall" "zenith" {
  name = "${var.name}-firewall"

  # SSH
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = var.ssh_allowed_ips
  }

  # HTTP
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # HTTPS
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # k3s API (only from allowed IPs)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "6443"
    source_ips = var.ssh_allowed_ips
  }

  # kubelet
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "10250"
    source_ips = var.ssh_allowed_ips
  }
}

# Server
resource "hcloud_server" "zenith" {
  name        = var.name
  server_type = var.server_type
  image       = var.image
  location    = var.location
  ssh_keys    = [hcloud_ssh_key.zenith.id]

  firewall_ids = [hcloud_firewall.zenith.id]

  labels = merge({
    "zenith.dev/managed-by"  = "terraform"
    "zenith.dev/environment" = var.environment
    "zenith.dev/role"        = var.role
  }, var.extra_labels)

  user_data = var.user_data
}
