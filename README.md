# Terraform Provider for Ruckus SmartZone (v7.1.1)

> A community Terraform provider that manages **Ruckus SmartZone / vSZ** configuration via the public REST API (`/wsg/api/public/{api_version}`) using **serviceTicket** authentication. Targets **SmartZone 7.1.1.0.551** by default, and supports earlier/later API versions via a provider argument. This is a partially AI generated codebase, as I am still learning Go. I apologize to any Go aficionados who may read this and cringe at my code style. I welcome any contributions to improve the code quality, add features, or fix bugs. Please see the [Contributing](https://github.com/nshreck/terraform-provider-ruckus/blob/main/README.md#%E2%80%8D%E2%80%8D-contributing) section below.

---

## ✨ Features

- **Provider auth** with `serviceTicket` (logon, reuse in query string). [1](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)  
- **Data sources**
  - `ruckus_zone` — look up a Zone by name.  
- **Resources**
  - `ruckus_wlan` — create/update/delete WLANs, including:
 - Security modes (Open, WPA2‑PSK, WPA3‑SAE, WPA2/WPA3 mixed, 802.1X, Web‑auth, **Hotspot (WISPr)**). [3](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)
    - VLAN (access VLAN, **Dynamic VLAN** via RADIUS attributes). [4](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)
    - Radio/band selection & client isolation. [3](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/README.md)
    - Tunneling (Ruckus GRE, Soft‑GRE, IPsec) via profile reference. [5](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider)[6](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/README.md?plain=1) 
- Built on HashiCorp’s **Terraform Plugin Framework** (modern, strongly‑typed provider SDK). [7](https://github.com/hashicorp/terraform-plugin-framework)[8](https://support.ruckuswireless.com/documents/2819-virtual-smartzone-getting-started-guide-vsz-vsz-d)

> 📘 **Docs references:**  
> • Ruckus SmartZone public API entry points & Version Matrix: Developer Central and v7.1.1 public API guides. [9](https://developer.ruckuswireless.com/)[2](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)  
> • Controller exposes local **OpenAPI** at `https://{host}:8443/wsg/apiDoc/openapi` to verify request/response shapes for your exact build. [2](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)

---

## 🔧 Requirements

- **Terraform** v1.4+ (tested with v1.5–v1.7)
- **Go** 1.21+ to build from source (provider uses the Plugin Framework). [8](https://support.ruckuswireless.com/documents/2819-virtual-smartzone-getting-started-guide-vsz-vsz-d)
- A SmartZone/vSZ controller reachable over HTTPS with an admin/API account (read/write for resource operations).  
- SmartZone firmware **7.1.1.0.551** (default `api_version = "v13_1"`). You can override `api_version` for other supported versions in 7.1.1’s matrix. [2](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)

---

## 🚀 Install

## From Terraform Registry

```hcl
terraform {
  required_providers {
    ruckus = {
      source  = "nshreck/ruckus"
      version = ">= 0.0.29"
    }
  }
}

```
### From Source

```bash
git clone https://github.com/nshreck/terraform-provider-ruckus.git
cd terraform-provider-ruckus
go build -o terraform-provider-ruckus
```

#### Recommended: development overrides

`dev_overrides` points Terraform at your build directly, with no plugin
directory layout and no `terraform init` step. Create `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "nshreck/ruckus" = "/absolute/path/to/terraform-provider-ruckus"
  }

  # Everything else installs from its origin registry as normal.
  # Omitting this leaves dev_overrides as the only installation method.
  direct {}
}
```

The path is the **directory** holding the compiled binary, not the binary
itself. Run `terraform plan` from your test configuration; Terraform prints
a warning confirming the override is active. Remove the block and run
`terraform init -upgrade` when returning to a released version.

To avoid touching a shared `~/.terraformrc`, put the block in a separate
file and set `TF_CLI_CONFIG_FILE=/path/to/dev.tfrc`.

#### Alternative: local plugin directory

```bash
PLATFORM="$(go env GOOS)_$(go env GOARCH)"
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/nshreck/ruckus/0.1.0/${PLATFORM}
mkdir -p "${PLUGIN_DIR}"
cp terraform-provider-ruckus "${PLUGIN_DIR}/"
```

Deriving `PLATFORM` from `go env` keeps this correct on Apple Silicon
(`darwin_arm64`), Intel macOS (`darwin_amd64`), and Linux alike.

## 🧑‍🤝‍🧑 Contributing
Contributions are welcome! Please open an issue or submit a pull request with improvements, bug fixes, or new features.
