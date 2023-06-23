dependency "vpc" {
  config_path = "../deploy"
  skip_outputs = true
}

include "root" {
  path = find_in_parent_folders()
}
