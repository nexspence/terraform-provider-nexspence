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
  s3 = {
    bucket           = "my-nexspence-blobs"
    region           = "us-east-1"
    endpoint         = "https://minio.example.com"
    access_key       = "AKIAIOSFODNN7EXAMPLE"
    secret_key       = var.s3_secret_key
    force_path_style = true
  }
}

# Group blob store — spread writes across two physical stores.
resource "nexspence_blobstore" "tiered" {
  name = "tiered"
  type = "group"
  group = {
    fill_policy = "write_to_first_fill"
    members     = [nexspence_blobstore.local.name, nexspence_blobstore.s3.name]
  }
}
