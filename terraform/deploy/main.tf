terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
  }
}

resource "google_service_account" "default" {
  account_id   = "service-account-gke"
  display_name = "Service Account GKE nodes"
}

resource "google_container_cluster" "primary" {
  name     = var.clustername
  location = var.region

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1

  logging_service = "none"

}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  name       = "my-node-pool"
  location   = var.region
  cluster    = google_container_cluster.primary.name
  node_count = 2
  node_locations = [
    var.zone,
  ]


  node_config {
    preemptible  = true
    machine_type = "e2-medium"

    service_account = google_service_account.default.email

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

resource "google_artifact_registry_repository" "myrepo" {
  location      = var.region
  repository_id = "my-repository"
  description   = "example docker repository"
  format        = "DOCKER"

}

resource "google_project_iam_member" "allow_image_pull" {
  project = var.project
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.default.email}"
}
