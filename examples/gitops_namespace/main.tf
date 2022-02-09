terraform {
  required_providers {
    gitops = {
      source  = "cloud-native-toolkit/gitops"
    }
  }
}

resource gitops_namespace ns {
    name = var.name
    content_dir = "${path.module}/yaml"
    server_name = var.server_name
    config = yamlencode(var.config)
    credentials = yamlencode(var.credentials)
}
