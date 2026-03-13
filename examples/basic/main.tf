terraform {
  required_providers {
    ruckus = {
      source  = "shreckn/ruckus"
      version = "0.0.1"
    }
  }
}

provider "ruckus" {
  host                 = "https://sz.example.com:8443"
  username             = var.username
  password             = var.password
  domain               = "System"
  api_version          = "v13_1"
  insecure_skip_verify = true
}

data "ruckus_zone" "hq" {
  name = "HQ"
}

resource "ruckus_wlan" "guest" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Guest WiFi"
  ssid        = "Guest-WLAN"
  description = "Guest access"
}
