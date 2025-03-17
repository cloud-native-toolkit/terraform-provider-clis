# Terraform Provider CLIs

Terraform provider to populate a Cloud Native Toolkit gitops repo with a new ArgoCD
application. Currently, this provider is a simple wrapper for the `gitops-namespace`
and `gitops-module` commands in the [igc](https://github.com/cloud-native-toolkit/ibm-garage-cloud-cli) cli.

The provider serves two purposes with this version:

- Provides more terraform-friendly integration with other terraform modules. This provider removes the need for null_resource resources to call the cli.
- Performs mutex locking on the git username to prevent concurrent access of the git apis and associated rate limits.

## Usage

### Provider configuration

Add the gitops module to the required providers list:

```hcl
terraform {
  required_providers {
    gitops = {
      source  = "cloud-native-toolkit/gitops"
    }
  }
}
```

Configure the provider block:

```hcl
provider "gitops" {
  username = var.git_username
  token = var.git_token
  bin_dir  = module.setup_clis.bin_dir
}
```

**Note:** `username` and `token` are both optional parameters. `bin_dir` should point to the directory where the `igc` cli can be found.

### Gitops Namespace resource

The Gitops Namespace resource will add namespace configuration to the repo.

```hcl
resource gitops_namespace ns {
    name = var.name
    content_dir = "${path.module}/yaml"
    server_name = var.server_name
    config = yamlencode(var.config)
    credentials = yamlencode(var.credentials)
}
```

### Gitops Module resource

The Gitops Module resource will add gitops module/application configutation to the repo.

```hcl
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
```

This entry provides a replacement for the following null_resource configuration:

```hcl
resource null_resource setup_gitops {
  depends_on = [null_resource.create_yaml]

  triggers = {
    name = local.name
    namespace = var.namespace
    yaml_dir = local.yaml_dir
    server_name = var.server_name
    layer = local.layer
    type = local.type
    git_credentials = yamlencode(var.git_credentials)
    gitops_config   = yamlencode(var.gitops_config)
    bin_dir = local.bin_dir
  }

  provisioner "local-exec" {
    command = "${self.triggers.bin_dir}/igc gitops-module '${self.triggers.name}' -n '${self.triggers.namespace}' --contentDir '${self.triggers.yaml_dir}' --serverName '${self.triggers.server_name}' -l '${self.triggers.layer}' --type '${self.triggers.type}'"

    environment = {
      GIT_CREDENTIALS = nonsensitive(self.triggers.git_credentials)
      GITOPS_CONFIG   = self.triggers.gitops_config
    }
  }

  provisioner "local-exec" {
    when = destroy
    command = "${self.triggers.bin_dir}/igc gitops-module '${self.triggers.name}' -n '${self.triggers.namespace}' --delete --contentDir '${self.triggers.yaml_dir}' --serverName '${self.triggers.server_name}' -l '${self.triggers.layer}' --type '${self.triggers.type}'"

    environment = {
      GIT_CREDENTIALS = nonsensitive(self.triggers.git_credentials)
      GITOPS_CONFIG   = self.triggers.gitops_config
    }
  }
}
```

## Development

### Build the application

Run the following command to build the provider

```shell
make build
```

## Test sample configuration

First, build and install the provider.

```shell
make install
```

Then, run the following command to initialize the workspace and apply the sample configuration.

```shell
terraform init && terraform apply
```