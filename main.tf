terraform {
  required_version = ">= 1.10.2"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.33.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.7.0"
    }
  }
}

provider "google" {
  region = var.region
}

data "google_project" "project" {
}

locals {
  project        = data.google_project.project.name
  project_number = data.google_project.project.number
}
