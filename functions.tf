resource "google_cloudfunctions2_function" "database_to_bigquery" {
  name        = "${var.basename}-database-to-bigquery"
  location    = var.region
  description = "Firestore から BigQuery へのデータ転送関数"

  build_config {
    runtime     = "go123"
    entry_point = "firestoreToBigQuery"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.source.name
      }
    }
  }

  service_config {
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
    service_account_email = google_service_account.database_to_bigquery.email

    environment_variables = {
      EXPORT_CONFIG = jsonencode({
        rules = var.rules
      })
    }
  }

  depends_on = [
    google_project_service.cloudfunctions,
    google_project_service.cloudbuild,
  ]
}

data "archive_file" "source" {
  type             = "zip"
  output_file_mode = "0644"
  output_path      = "functions.zip"
  source_dir       = "${path.module}/functions"
  excludes = [
    "**/*_test.go",
  ]
}

resource "google_storage_bucket" "source" {
  name     = "${local.project_number}-${var.basename}-gcf-source"
  location = var.region

  depends_on = [google_project_service.storage]
}

resource "google_storage_bucket_object" "source" {
  # https://github.com/hashicorp/terraform-provider-google/issues/1938
  name   = "functions_${data.archive_file.source.output_md5}.zip"
  bucket = google_storage_bucket.source.name
  source = data.archive_file.source.output_path

  lifecycle {
    create_before_destroy = true
  }
}
