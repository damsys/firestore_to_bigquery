resource "google_service_account" "database_to_bigquery" {
  account_id   = "${var.basename}-database-to-bigquery"
  display_name = "Database to BigQuery Service Account (Demo)"

  depends_on = [google_project_service.iam]
}

resource "google_bigquery_dataset_iam_binding" "firestore_to_bigquery" {
  dataset_id = google_bigquery_dataset.firestore_to_bigquery.dataset_id
  role       = "roles/bigquery.dataEditor"

  members = [
    "serviceAccount:${google_service_account.database_to_bigquery.email}",
  ]
}

resource "google_cloud_run_service_iam_member" "firestore_to_bigquery_invoker" {
  location = google_cloudfunctions2_function.database_to_bigquery.location
  service  = google_cloudfunctions2_function.database_to_bigquery.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.database_to_bigquery.email}"
}

resource "google_project_iam_member" "firestore_to_bigquery_event_receiver" {
  project = local.project
  role    = "roles/eventarc.eventReceiver"
  member  = "serviceAccount:${google_service_account.database_to_bigquery.email}"
}

resource "google_service_account_iam_member" "pubsub_firestore_to_bigquery" {
  service_account_id = google_service_account.database_to_bigquery.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${local.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"

  depends_on = [google_project_service.pubsub]
}

resource "google_pubsub_topic_iam_member" "firestore_to_bigquery_pubsub_publisher" {
  topic  = google_pubsub_topic.testdata.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${google_service_account.database_to_bigquery.email}"
}
