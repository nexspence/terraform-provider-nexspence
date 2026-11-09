terraform {
  required_providers {
    nexspence = {
      source  = "nexspence/nexspence"
      version = ">= 0.5.0"
    }
  }
}

provider "nexspence" {
  url   = "https://nexspence.example.com"
  token = var.nexspence_token
}
