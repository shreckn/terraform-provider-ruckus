variable "controller" {
  type        = string
  description = "Ruckus SmartZone controller hostname or IP address"
}

variable "username" {
  type        = string
  description = "Controller username"
  sensitive   = true
}

variable "password" {
  type        = string
  description = "Controller password"
  sensitive   = true
}

variable "domain" {
  type        = string
  description = "Authentication domain"
  default     = ""
}

variable "insecure" {
  type        = bool
  description = "Skip TLS verification"
  default     = false
}

variable "zone_name" {
  type        = string
  description = "Zone name where WLANs will be created"
}

variable "psk" {
  type        = string
  description = "Pre-shared key for WPA2 encryption"
  sensitive   = true
}

variable "accounting_vlan" {
  type        = number
  description = "VLAN ID for Accounting department"
}

variable "hr_vlan" {
  type        = number
  description = "VLAN ID for HR department"
}

