# WLAN Group with Members Example

This example demonstrates how to use the updated `ruckus_wlan_group` resource to add existing WLANs to newly created WLAN groups.

## Overview

The Ruckus provider now supports the `wlan_ids` argument on the `ruckus_wlan_group` resource, allowing you to:

- Create a new WLAN group and add existing WLANs to it
- Add multiple WLANs (both newly created and pre-existing) to a group
- Manage group membership directly through the group resource

## Usage

### Add Newly Created WLANs to a Group

```hcl
resource "ruckus_wlan" "accounting" {
  zone_id = data.ruckus_zone.hq.id
  name    = "Accounting WLAN"
  ssid    = "accounting-ssid"
  
  encryption {
    mode       = "WPA2"
    passphrase = var.psk
  }
}

resource "ruckus_wlan_group" "office_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Office Department WLANs"
  description = "WLANs for office departments"
  
  # Add the newly created WLAN to this group
  wlan_ids = [ruckus_wlan.accounting.id]
}
```

### Add Existing WLANs from Data Source

```hcl
data "ruckus_wlans" "all_wlans" {
  zone_id = data.ruckus_zone.hq.id
}

resource "ruckus_wlan_group" "managed_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Managed WLANs"
  
  # Add existing WLANs that match certain criteria
  wlan_ids = [
    for wlan in data.ruckus_wlans.all_wlans.list :
    wlan.id if contains(lower(wlan.name), "managed")
  ]
}
```

### Mix New and Existing WLANs

```hcl
resource "ruckus_wlan_group" "mixed_wlans" {
  zone_id     = data.ruckus_zone.hq.id
  name        = "Mixed WLANs Group"
  
  wlan_ids = concat(
    [ruckus_wlan.accounting.id],  # Newly created
    [ruckus_wlan.hr.id],          # Newly created
    # Add more existing WLANs as needed
  )
}
```

## Resource Arguments

The `ruckus_wlan_group` resource supports the following arguments:

- `zone_id` (Required) - ID of the zone to which the WLAN Group belongs
- `name` (Required) - Name of the WLAN Group
- `description` (Optional) - Description of the WLAN Group
- `wlan_ids` (Optional) - List of WLAN IDs to add to this group

## Resource Attributes

In addition to the arguments, the following attributes are available:

- `id` (Computed) - Unique identifier of the WLAN Group
- `members` (Computed) - List of WLAN IDs that are members of this group (read from API)

## How It Works

1. The WLAN group is created on the controller
2. Each WLAN ID specified in `wlan_ids` is added to the group via the membership API endpoint
3. The `members` attribute reflects all WLANs currently in the group (as returned by the controller)

## Notes

- The `wlan_ids` attribute can be used with both newly created WLANs and existing WLANs
- WLANs can be added to a group at creation time or modified during updates
- The `members` attribute is computed and shows the actual state from the controller
- If a WLAN is already in another group, it will be moved to the specified group

## Error Handling

If adding a WLAN to the group fails, the error will be reported and the provider will not proceed with reading the resource. Common issues:

- Invalid WLAN ID
- WLAN does not exist in the specified zone
- Network or authentication issues with the controller

## Running the Example

1. Set up your variables:
   ```bash
   export TF_VAR_controller="controller.example.com"
   export TF_VAR_username="admin"
   export TF_VAR_password="password"
   export TF_VAR_zone_name="default"
   export TF_VAR_psk="PreSharedKey123"
   export TF_VAR_accounting_vlan="100"
   export TF_VAR_hr_vlan="101"
   ```

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Plan the changes:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

## See Also

- [Ruckus WLAN Resource](../../README.md)
- [Ruckus WLAN Data Source](../../README.md)

