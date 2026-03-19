variable "username" {
  type = string
}
variable "password" {
  sensitive = true
  type = string
}
variable "domain" {
  default = ""
  type = string
}
variable "insecure" {
  default = false
  type = bool
}
variable "psk" {
  sensitive = true
  type = string
}
variable "group_name" {
  type = string
}
variable "controller" {
  type = string
}
variable "zones" {
  type = list(string)
}
variable "ssid" {
  type = string
}
variable "vlan" {
  type = number
}