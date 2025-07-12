resource "google_eventarc_trigger" "database_to_bigquery" {
  for_each = var.rules

  name     = "${var.basename}-database-to-bigquery-${lower(each.key)}"
  location = var.region

  matching_criteria {
    attribute = "type"
    value     = "google.cloud.firestore.document.v1.written"
  }

  matching_criteria {
    attribute = "database"
    value     = google_firestore_database.database.name
  }

  matching_criteria {
    attribute = "document"
    value     = "${each.key}/{id}"
    operator  = "match-path-pattern"
  }

  event_data_content_type = "application/protobuf"

  destination {
    cloud_run_service {
      service = google_cloudfunctions2_function.database_to_bigquery.name
      region  = google_cloudfunctions2_function.database_to_bigquery.location
    }
  }

  service_account = google_service_account.database_to_bigquery.email

  depends_on = [
    google_project_service.eventarc,
    google_cloud_run_service_iam_member.firestore_to_bigquery_invoker,
    google_project_iam_member.firestore_to_bigquery_event_receiver,
  ]
}
