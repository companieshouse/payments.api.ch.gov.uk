# Define all hardcoded local variable and local variables looked up from data resources
locals {
  stack_name                  = "payments-service" # this must match the stack name the service deploys into
  name_prefix                 = "${local.stack_name}-${var.environment}"
  global_prefix               = "global-${var.environment}"
  service_name                = "payments-api"
  container_port              = "3055"
  eric_port                   = "10000"
  docker_repo                 = "payments.api.ch.gov.uk"
  kms_alias                   = "alias/${var.aws_profile}/environment-services-kms"
  lb_listener_rule_priority   = 50
  lb_listener_paths           = ["/payments", "/payments/*", "/payments/*/refunds", "/private/payments/*", "/private/payments/*/external-journey", "/callback/payments/govpay/*", "/callback/payments/paypal/orders/*", "/admin/payments/bulk-refunds/*", "/admin/payments/bulk-refunds" ]
  healthcheck_path            = "/healthcheck"
  healthcheck_matcher         = "200"
  vpc_name                    = local.stack_secrets["vpc_name"]
  s3_config_bucket            = data.vault_generic_secret.shared_s3.data["config_bucket_name"]
  app_environment_filename    = "payments.api.ch.gov.uk.env"
  eric_environment_filename   = "eric.env"
  use_set_environment_files   = var.use_set_environment_files
  application_subnet_ids      = data.aws_subnets.application.ids

  stack_secrets              = jsondecode(data.vault_generic_secret.stack_secrets.data_json)
  application_subnet_pattern = local.stack_secrets["application_subnet_pattern"]

  ssm_global_version_map = [
    for sec in data.aws_ssm_parameter.global_secret : {
      name = "GLOBAL_${var.ssm_version_prefix}${replace(upper(basename(sec.name)), "-", "_")}", value = sec.version
    }
  ]

  ssm_service_version_map = [
    for sec in module.secrets.secrets : {
      name = "${replace(upper(local.service_name), "-", "_")}_${var.ssm_version_prefix}${replace(upper(basename(sec.name)), "-", "_")}", value = sec.version
    }
  ]

  task_environment = concat(local.ssm_global_version_map,local.ssm_service_version_map,[
    { "name" : "PORT", "value" : local.container_port }
  ])
}

