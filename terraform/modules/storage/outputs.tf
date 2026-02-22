output "bucket_name" {
  description = "Created S3 bucket name"
  value       = aws_s3_bucket.zenith.id
}

output "bucket_arn" {
  description = "S3 bucket ARN"
  value       = aws_s3_bucket.zenith.arn
}
