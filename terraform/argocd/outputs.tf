output "cluster_name" {
  description = "Cluster name"
  value       = google_container_cluster.primary.name
}

output "endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "ca_cert" {
  value = base64decode(google_container_cluster.primary.master_auth.0.cluster_ca_certificate)
}

output "access_token" {
  value     = data.google_client_config.current.access_token
  sensitive = true
}
