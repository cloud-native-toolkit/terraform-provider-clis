terraform {
  required_providers {
    gitops = {
      source  = "cloud-native-toolkit/gitops"
    }
  }
}

resource gitops_module module {
    name = var.name
    namespace = var.namespace
    content_dir = "${path.module}/yaml"
    server_name = var.server_name
    layer = var.layer
    type = var.type
    config = yamlencode(var.config)
    credentials = yamlencode(var.credentials)
}
