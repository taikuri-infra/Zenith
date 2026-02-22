terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Hetzner Object Storage is S3-compatible
# We use the AWS provider with a custom endpoint
resource "aws_s3_bucket" "zenith" {
  bucket = var.bucket_name

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
    Project     = "zenith"
  }
}
