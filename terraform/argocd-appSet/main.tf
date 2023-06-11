terraform {
  backend "gcs" {
    bucket = "myinfra1-tf-state"
    prefix = "terraform/state"
  }
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.17.0"
    }
  }
}

provider "google" {

  project = "myinfra1"
  region  = "us-central1"
  zone    = "us-central1-c"
}

provider "kubernetes" {
  host                   = "https://${google_container_cluster.primary.endpoint}"
  cluster_ca_certificate = base64decode(google_container_cluster.primary.master_auth.0.cluster_ca_certificate)
  token                  = data.google_client_config.current.access_token
}

data "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"

}

resource "kubernetes_manifest" "argocd-appSet" {
  depends_on = [helm_release.argocd]
  manifest   = yamldecode(file("${path.module}/../../appSet.yaml"))
}
