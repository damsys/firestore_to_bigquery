variable "region" {
  description = "region"
  type        = string
  default     = "asia-northeast1"
}

variable "basename" {
  description = "basename for resources"
  type        = string
  default     = "f2bdemo"
}

variable "rules" {
  description = "Kind to Table mappings"
  type = map(object({
    table  = string
    fields = list(string)
  }))
  default = {
    "Testdata" = {
      table  = "f2bdemo.testdata"
      fields = ["Name", "Ruby", "Age", "CreatedAt"]
    }
  }
}
