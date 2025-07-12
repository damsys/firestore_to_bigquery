# https://github.com/terraform-linters/tflint/blob/master/docs/user-guide/config.md
tflint {
  required_version = ">= 0.56"
}

# https://github.com/terraform-linters/tflint-ruleset-terraform/blob/main/docs/rules/README.md
plugin "terraform" {
    enabled = true
    preset  = "recommended"
}

rule "terraform_comment_syntax" {
  enabled = true
}

rule "terraform_naming_convention" {
  enabled = true
}

rule "terraform_standard_module_structure" {
  enabled = true
}

rule "terraform_unused_required_providers" {
  enabled = true
}

rule "terraform_module_version" {
  enabled = true
  # モジュールのバージョンは固定で指定する。
  exact = true
}
