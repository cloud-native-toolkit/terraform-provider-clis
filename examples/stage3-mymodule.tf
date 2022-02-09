
module cntk_module {
  source = "./gitops_module"

  name = "my-module"
  namespace = module.cntk_namespace.name
  server_name = module.gitops.server_name
  config = module.gitops.gitops_config
  credentials = module.gitops.git_credentials
}

module another_module {
  source = "./gitops_module"

  name = "another-module"
  namespace = module.cntk_namespace.name
  server_name = module.gitops.server_name
  config = module.gitops.gitops_config
  credentials = module.gitops.git_credentials
}
