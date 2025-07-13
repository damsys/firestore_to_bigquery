resource "google_pubsub_topic" "testdata" {
  name = "${var.basename}-ftb-testdata"

  message_retention_duration = "3600s"

  depends_on = [google_project_service.pubsub]
}

resource "google_pubsub_subscription" "testdata" {
  name  = "${var.basename}-ftb-testdata"
  topic = google_pubsub_topic.testdata.name

  bigquery_config {
    table               = "${google_bigquery_table.testdata.project}.${google_bigquery_table.testdata.dataset_id}.${google_bigquery_table.testdata.table_id}"
    use_table_schema    = true
    drop_unknown_fields = true

    service_account_email = google_service_account.database_to_bigquery.email
  }

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "300s"
  }
}
