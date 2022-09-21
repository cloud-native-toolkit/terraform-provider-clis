
data clis_check clis {
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
    "glab"
  ]
}

data clis_check clis2 {
  clis = [
    "yq",
    "argocd",
    "kubeseal",
    "oc",
    "kubectl"
  ]
}

resource local_file bin_output {
  filename = "${path.cwd}/.bin_dir_out"

  content = data.clis_check.clis.bin_dir
}
