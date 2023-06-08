output "cluster_name" {
  description = "Cluster name"
  value       = google_container_cluster.primary.name
}

output "endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "access_token" {
  value = google_container_cluster.primary.access_token
}

output "cluster_ca_certificate" {
  value = base64decode(google_container_cluster.primary.ca_certificate)
}
