terraform {
  required_providers {
    sendgrid = {
      source = "registry.terraform.io/kenzo0107/sendgrid"
    }
  }
}

provider "sendgrid" {
}
