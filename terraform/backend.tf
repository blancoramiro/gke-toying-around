# Generated by Terragrunt. Sig: nIlQXj57tbuaRZEa
terraform {
  backend "gcs" {
    bucket = "myinfra1-tf-state"
    prefix = "./terraform.tfstate"
  }
}
