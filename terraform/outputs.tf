output "gcloud_connect" {
  description = "gcloud cli add cluster"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.primary.name} --region=${google_container_cluster.primary.location}"
}
output "cluster_name" {
  description = "cluster name"
  value       = google_container_cluster.primary.name
}
output "cluster_location" {
  description = "cluster location"
  value       = google_container_cluster.primary.location
}
