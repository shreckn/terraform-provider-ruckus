# Terraform Provider for Ruckus SmartZone (v7.1.1)

> A community Terraform provider that manages **Ruckus SmartZone / vSZ** configuration via the public REST API (`/wsg/api/public/{api_version}`) using **serviceTicket** authentication. Targets **SmartZone 7.1.1.0.551** by default, and supports earlier/later API versions via a provider argument. [1](https://docs.devnetexperttraining.com/static-docs/Terraform/docs/plugin/framework/acctests.html)[2](https://github.com/hashicorp/terraform-plugin-go)

---

This was mostly Vibe-Coded. I apologize to any Go aficionados who may read this.

---

## ✨ Features

- **Provider auth** with `serviceTicket` (logon, reuse in query string). [1](https://docs.devnetexperttraining.com/static-docs/Terraform/docs/plugin/framework/acctests.html)  
- **Data sources**
  - `ruckus_zone` — look up a Zone by name.  
- **Resources**
  - `ruckus_wlan` — create/update/delete WLANs, including:
 - Security modes (Open, WPA2‑PSK, WPA3‑SAE, WPA2/WPA3 mixed, 802.1X, Web‑auth, **Hotspot (WISPr)**). [3](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/README.md)
    - VLAN (access VLAN, **Dynamic VLAN** via RADIUS attributes). [4](https://oneuptime.com/blog/post/2026-02-23-terraform-plugin-framework/view)
    - Radio/band selection & client isolation. [3](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/README.md)
    - Tunneling (Ruckus GRE, Soft‑GRE, IPsec) via profile reference. [5](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider)[6](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/README.md?plain=1)
    - Advanced knobs (e.g., min basic rate, OFDMA) and a **`custom_json`** escape hatch to pass raw fields if needed for your firmware.  
- Built on HashiCorp’s **Terraform Plugin Framework** (modern, strongly‑typed provider SDK). [7](https://github.com/RagaviCommscope/Virtual-SmartZone)[8](https://support.ruckuswireless.com/documents/2819-virtual-smartzone-getting-started-guide-vsz-vsz-d)

> 📘 **Docs references:**  
> • Ruckus SmartZone public API entry points & Version Matrix: Developer Central and v7.1.1 public API guides. [9](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework)[2](https://github.com/hashicorp/terraform-plugin-go)  
> • Controller exposes local **OpenAPI** at `https://{host}:8443/wsg/apiDoc/openapi` to verify request/response shapes for your exact build. [2](https://github.com/hashicorp/terraform-plugin-go)

---

## 🔧 Requirements

- **Terraform** v1.4+ (tested with v1.5–v1.7)
- **Go** 1.21+ to build from source (provider uses the Plugin Framework). [8](https://support.ruckuswireless.com/documents/2819-virtual-smartzone-getting-started-guide-vsz-vsz-d)
- A SmartZone/vSZ controller reachable over HTTPS with an admin/API account (read/write for resource operations).  
- SmartZone firmware **7.1.1.0.551** (default `api_version = "v13_1"`). You can override `api_version` for other supported versions in 7.1.1’s matrix. [2](https://github.com/hashicorp/terraform-plugin-go)

---

## 🚀 Install

Until this provider is published to the Terraform Registry, install from source:

```bash
git clone https://github.com/nshreck/terraform-provider-ruckus.git
cd terraform-provider-ruckus
go build -o terraform-provider-ruckus

# Local dev install layout (Linux/macOS example)
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/nshreck/ruckus/0.1.0/darwin_amd64
cp terraform-provider-ruckus ~/.terraform.d/plugins/registry.terraform.io/nshreck/ruckus/0.1.0/darwin_amd64/
