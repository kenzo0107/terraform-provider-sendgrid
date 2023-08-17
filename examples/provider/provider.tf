terraform {
  required_providers {
    hashicups = {
      source = "kenzo0107/sendgrid"
    }
  }
}

provider "sendgrid" {
  api_key = "<your api key>"
}
