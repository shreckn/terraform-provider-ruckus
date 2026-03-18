terraform {
  required_providers {
    ruckus = {
      source  = "nshreck/ruckus"
      version = "0.0.1"
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

data "ruckus_zone" "zone" {
  name = var.zone
}

resource "ruckus_wlan" "wlan" {
  zone_id     = data.ruckus_zone.zone.id
  name        = var.ssid
  ssid        = var.ssid

  security {
    mode        = "wpa2_psk"
    passphrase  = var.psk
    encryption  = "ccmp"
  }

  vlan {
    access_vlan  = var.vlan
  }

  radio {
    band             = var.band
    client_isolation = var.client_isolation
  }

  advanced {
    min_bss_rate = 6000
    ofdma        = true
  }
}

resource "ruckus_wlan_group" "corp_group" {
  zone_id = data.ruckus_zone.zone.id
  name    = "Corporate WLAN Group"
  description = "Group containing corporate WLANs"
  members = [ruckus_wlan.wlan.id]
}
