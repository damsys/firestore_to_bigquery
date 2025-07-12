resource "google_firestore_database" "database" {
  name        = var.basename
  location_id = var.region
  type        = "FIRESTORE_NATIVE"

  # Just for demo use
  delete_protection_state = "DELETE_PROTECTION_DISABLED"

  depends_on = [google_project_service.firestore]
}
