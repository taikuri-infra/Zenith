# =============================================================================
# Zenith Production DR — Disaster Recovery Cluster (Helsinki)
# =============================================================================
#
# A dormant k3s cluster in Hetzner Helsinki (hel1) for business continuity.
# Components:
#   - Minimal k3s single-node cluster
#   - Velero BackupStorageLocation (shared S3 bucket with production)
#   - CNPG standby cluster (streaming replication from production primary)
#
# This cluster is NOT active. It only becomes primary via dr-failover.sh.
#
# Usage:
#   terraform init
#   terraform plan -var-file=terraform.tfvars
#   terraform apply -var-file=terraform.tfvars
#
# =============================================================================

terraform {
  required_version = ">= 1.5"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

# --- SSH Key ---

resource "hcloud_ssh_key" "dr" {
  name       = "zenith-dr"
  public_key = var.ssh_public_key
}

# --- DR Server (Helsinki) ---

resource "hcloud_server" "dr" {
  name        = "zenith-dr"
  server_type = var.server_type
  image       = "ubuntu-24.04"
  location    = "hel1" # Helsinki datacenter
  ssh_keys    = [hcloud_ssh_key.dr.id]

  labels = {
    environment = "dr"
    project     = "zenith"
  }

  user_data = <<-USERDATA
    #!/bin/bash
    set -euo pipefail

    # Install k3s (dormant — no workloads scheduled initially)
    curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="${var.k3s_version}" sh -s - \
      --disable=traefik \
      --write-kubeconfig-mode=644

    # Wait for k3s to be ready
    until kubectl get nodes; do sleep 5; done

    # Install Helm
    curl -sfL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

    # Install Velero CLI
    VELERO_VERSION="v1.14.0"
    curl -fsSL "https://github.com/vmware-tanzu/velero/releases/download/$${VELERO_VERSION}/velero-$${VELERO_VERSION}-linux-amd64.tar.gz" | \
      tar xz -C /tmp && mv /tmp/velero-$${VELERO_VERSION}-linux-amd64/velero /usr/local/bin/

    # Create S3 credentials for Velero
    cat > /tmp/s3-credentials <<EOF2
    [default]
    aws_access_key_id=${var.s3_access_key}
    aws_secret_access_key=${var.s3_secret_key}
    EOF2

    # Install Velero with S3 backend (shared bucket with production)
    velero install \
      --provider aws \
      --plugins velero/velero-plugin-for-aws:v1.10.0 \
      --bucket "${var.velero_bucket}" \
      --backup-location-config "region=fsn1,s3ForcePathStyle=true,s3Url=${var.s3_endpoint}" \
      --secret-file /tmp/s3-credentials \
      --use-volume-snapshots=false
    rm -f /tmp/s3-credentials

    # Install CNPG operator
    helm repo add cnpg https://cloudnative-pg.github.io/charts
    helm install cnpg cnpg/cloudnative-pg --namespace cnpg-system --create-namespace

    echo "DR cluster bootstrap complete"
  USERDATA
}

# --- Firewall ---

resource "hcloud_firewall" "dr" {
  name = "zenith-dr"

  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "22"
    source_ips = var.allowed_ssh_ips
  }

  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "6443"
    source_ips = var.allowed_ssh_ips
  }

  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }
}

resource "hcloud_firewall_attachment" "dr" {
  firewall_id = hcloud_firewall.dr.id
  server_ids  = [hcloud_server.dr.id]
}
