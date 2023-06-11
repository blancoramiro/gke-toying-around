terraform {
  backend "gcs" {
    bucket = "myinfra1-tf-state"
    prefix = "terraform/state/argocd"
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
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.8.0"
    }
  }
}

provider "google" {

  project = "myinfra1"
  region  = "us-central1"
  zone    = "us-central1-c"
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.primary.endpoint}"
  cluster_ca_certificate = base64decode(data.google_container_cluster.primary.master_auth.0.cluster_ca_certificate)
  token                  = data.google_client_config.current.access_token
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.primary.endpoint}"
    cluster_ca_certificate = base64decode(data.google_container_cluster.primary.master_auth.0.cluster_ca_certificate)
    token                  = data.google_client_config.current.access_token
  }
}

data "google_client_config" "current" {
}

data "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"
}

resource "kubernetes_namespace" "argocd" {

  metadata {
    name = "argocd"
  }
}

resource "helm_release" "argocd" {
  depends_on = [data.google_container_cluster.primary, kubernetes_namespace.argocd]
  name       = "argocd"
  namespace  = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "5.36.1"
  values     = [file("${path.module}/../../argocd-values.yaml")]
}
