terraform {
}

locals {
  project = "myinfra1"
  region  = "us-central1"
  zone    = "us-central1-c"
  clustername = "my-gke-cluster"
}

inputs = {

  project = local.project
  region  = local.region
  zone    = local.zone
  clustername = local.clustername

}

generate "provider" {
  path = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents = <<EOF
provider "google" {

  project = "${local.project}"
  region  = "${local.region}"
  zone    = "${local.zone}"
}
EOF
}

generate "variables" {
  path      = "variables.tf"
  if_exists = "overwrite"
  contents  = <<EOF
variable "project" {
 type = string
}
variable "region" {
 type = string
}
variable "zone" {
 type = string
}
variable "clustername" {
 type = string
}
EOF
}

remote_state {
  backend = "gcs"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    bucket = "myinfra1-tf-state"
    prefix = "${path_relative_to_include()}/terraform.tfstate"
  }
}
