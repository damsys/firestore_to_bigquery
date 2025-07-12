# BigQuery用のデータセット
resource "google_bigquery_dataset" "firestore_to_bigquery" {
  dataset_id  = replace(var.basename, "-", "_")
  description = "Firestore to BigQuery Demo"
  location    = var.region

  lifecycle {
    # access は別途設定するので変更を無視する
    ignore_changes = [access]
  }

  depends_on = [google_project_service.bigquery]
}

resource "google_bigquery_table" "testdata" {
  dataset_id = google_bigquery_dataset.firestore_to_bigquery.dataset_id
  table_id   = "testdata"

  table_constraints {
    primary_key {
      columns = ["ID"]
    }
  }

  schema = jsonencode([
    {
      name = "ID"
      type = "STRING"
      mode = "REQUIRED"
    },
    {
      name = "Name"
      type = "STRING"
      mode = "NULLABLE"
    },
    {
      name = "Ruby"
      type = "STRING"
      mode = "NULLABLE"
    },
    {
      name = "Age"
      type = "INTEGER"
      mode = "NULLABLE"
    },
    {
      name = "CreatedAt"
      type = "TIMESTAMP"
      mode = "NULLABLE"
    },
  ])
}
