# Local filesystem blob store.
resource "nexspence_blobstore" "local" {
  name = "main"
  type = "local"
  path = "./data/blobs/main"
}

# S3-compatible blob store (MinIO / AWS S3).
resource "nexspence_blobstore" "s3" {
  name = "s3-store"
  type = "s3"
  s3 {
    bucket           = "my-nexspence-blobs"
    region           = "us-east-1"
    endpoint         = "https://minio.example.com"
    access_key       = "AKIAIOSFODNN7EXAMPLE"
    secret_key       = var.s3_secret_key
    force_path_style = true
  }
}
