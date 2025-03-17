# Copyright (c) 2025 Cloud-Native Toolkit
# SPDX-License-Identifier: MIT

data "clis_check" "clis" {
  clis = [
    "yq",
    "jq",
    "igc",
    "helm",
    "argocd",
    "rosa",
    "kubeseal",
    "oc",
    "kubectl",
    "ibmcloud",
    "ibmcloud-is",
    "ibmcloud-ob",
    "ibmcloud-ks",
    "ibmcloud-cr",
    "kustomize",
    "gitu",
    "gh",
    "glab",
    "openshift-install-4.10",
    "operator-sdk"
  ]
}

data "clis_check" "clis2" {
  clis = [
    "yq",
    "argocd",
    "kubeseal",
    "oc",
    "kubectl",
    "helm"
  ]
}

resource "local_file" "bin_output" {
  filename = "${path.cwd}/.bin_dir_out"

  content = data.clis_check.clis.bin_dir
}
