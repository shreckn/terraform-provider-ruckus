terraform {
  required_providers {
    ruckus = {
      source  = "nshreck/ruckus"
      version = ">= 0.0.16"
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

data "ruckus_zone" "hq" {
  name = var.zone_name
}

# Create new WLANs
resource "ruckus_wlan" "accounting" {
  zone_id = data.ruckus_zone.hq.id
  name    = "Accounting WLAN"
  ssid    = "accounting-ssid"

  encryption {
    mode       = "WPA2"
    passphrase = var.psk
    algorithm  = "AES"
  }

  vlan {
    access_vlan = var.accounting_vlan
  }
}

resource "ruckus_wlan" "hr" {
  zone_id = data.ruckus_zone.hq.id
  name    = "HR WLAN"
  ssid    = "hr-ssid"

  encryption {
    mode       = "WPA2"
    passphrase = var.psk
    algorithm  = "AES"
  }

  vlan {
    access_vlan = var.hr_vlan
  }
}

# Get existing WLAN data
data "ruckus_wlans" "all_wlans" {
  zone_id = data.ruckus_zone.hq.id
}

# Create a WLAN group and add multiple WLANs to it
# Option 1: Add newly created WLANs
resource "ruckus_wlan_group" "office_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Office Department WLANs"
  description = "WLANs for office departments"

  # Add the newly created WLANs to this group
  wlan_ids = [
    ruckus_wlan.accounting.id,
    ruckus_wlan.hr.id,
  ]
}

# Option 2: Add existing WLANs by filtering them from the data source
resource "ruckus_wlan_group" "managed_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Managed WLANs"
  description = "Group of existing WLANs"

  # Add existing WLANs that match a criteria
  wlan_ids = [
    for wlan in data.ruckus_wlans.all_wlans.list :
    wlan.id if contains(lower(wlan.name), "managed")
  ]
}

# Option 3: Mixed - add both newly created and existing WLANs
resource "ruckus_wlan_group" "mixed_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Mixed WLANs Group"
  description = "Group containing both new and existing WLANs"

  wlan_ids = concat(
    [ruckus_wlan.accounting.id],
    [wlan.id for wlan in data.ruckus_wlans.all_wlans.list if wlan.name == "existing-wlan-name"]
  )
}

