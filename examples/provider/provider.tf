# Configure the sendgrid provider using the required_providers stanza.
# You may optionally use a version directive to prevent breaking
# changes occurring unannounced.
terraform {
  required_providers {
    hashicups = {
      source = "registry.terraform.io/kenzo0107/sendgrid"
    }
  }
}

provider "sendgrid" {
  api_key = "<your api key>"
}
