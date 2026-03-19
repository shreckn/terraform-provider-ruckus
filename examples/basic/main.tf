terraform {
  required_providers {
    ruckus = {
      source  = "nshreck/ruckus"
      version =  ">= 0.0.8"
    }
  }
}
provider "ruckus" {
  host                 = var.controller
  username             = var.username
  password             = var.password
  domain               = var.domain
  api_version          = "v13_1"
  insecure_skip_verify = var.insecure
}

data "ruckus_zone" "zones" {
  for_each    = toset(var.zones)
  name    = each.value
}

resource "ruckus_wlan" "wlan" {
  for_each    = toset(var.zones)
  zone_id     = data.ruckus_zone.zones[each.value].id
  name        = var.ssid
  ssid        = var.ssid

  encryption {
    mode        = "WPA2"
    passphrase  = var.psk
    algorithm  = "AES"
  }

  vlan {
    access_vlan  = var.vlan
  }
}

resource "ruckus_wlan_group" "group" {
  for_each    = toset(var.zones)
  zone_id     = data.ruckus_zone.zones[each.value].id
  name    = var.group_name
}