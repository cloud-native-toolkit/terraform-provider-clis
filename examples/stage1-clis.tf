
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
    "gitu"
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
  bin_dir = data.clis_check.clis.bin_dir
}
