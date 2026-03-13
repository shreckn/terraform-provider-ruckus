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

resource "ruckus_wlan" "corp" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Corp-Staff"
  ssid        = "Corp-Staff"
  description = "Corp secure WLAN"

  security {
    mode        = "wpa2_psk"
    passphrase  = var.corp_psk
    encryption  = "ccmp"
  }

  vlan {
    access_vlan  = 120
    dynamic_vlan = false
  }

  radio {
    band             = "5"
    client_isolation = true
  }

  advanced {
    min_bss_rate = 6000
    ofdma        = true
  }
}