#!/usr/bin/env bash

BIN_DIR=$(cat .bin_dir)

ls -l "${BIN_DIR}"

if ! "${BIN_DIR}/jq" --version; then
  echo "jq not found" >&2
  exit 1
else
  echo "jq cli found"
fi

if ! "${BIN_DIR}/yq3" --version; then
  echo "yq3 not found" >&2
  exit 1
else
  echo "yq3 cli found"
fi

if ! "${BIN_DIR}/yq4" --version; then
  echo "yq4 not found" >&2
  exit 1
else
  echo "yq4 cli found"
fi

if ! "${BIN_DIR}/igc" --version; then
  echo "igc not found" >&2
  exit 1
else
  echo "igc cli found"
fi

if ! "${BIN_DIR}/helm" version --short; then
  echo "helm not found" >&2
  exit 1
else
  echo "helm cli found"
fi

if ! "${BIN_DIR}/argocd" version --client; then
  echo "argocd not found" >&2
  exit 1
else
  echo "argocd cli found"
fi

if ! "${BIN_DIR}/rosa" version; then
  echo "rosa cli not configured properly" >&2
  exit 1
else
  echo "rosa cli configured properly"
fi

if ! "${BIN_DIR}/gh" version; then
  echo "gh cli not configured properly" >&2
  exit 1
else
  echo "gh cli configured properly"
fi

s

if ! "${BIN_DIR}/kubeseal" --version; then
  echo "kubeseal cli not configured properly" >&2
  exit 1
else
  echo "kubeseal cli configured properly"
fi

if ! "${BIN_DIR}/oc" version --client=true; then
  echo "oc cli not configured properly" >&2
  exit 1
else
  echo "oc cli configured properly"
fi

if ! "${BIN_DIR}/kubectl" version --client=true; then
  echo "kubectl cli not configured properly" >&2
  exit 1
else
  echo "kubectl cli configured properly"
fi

if ! "${BIN_DIR}/ibmcloud" version; then
  echo "ibmcloud cli not configured properly" >&2
  exit 1
else
  echo "ibmcloud cli configured properly"
fi

if ! "${BIN_DIR}/ibmcloud" plugin show infrastructure-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud is plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud is plugin configured properly"
fi

if ! "${BIN_DIR}/ibmcloud" plugin show observe-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud ob plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud ob plugin configured properly"
fi

if ! "${BIN_DIR}/ibmcloud" plugin show kubernetes-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud ks plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud ks plugin configured properly"
fi

if ! "${BIN_DIR}/ibmcloud" plugin show container-registry 1> /dev/null 2> /dev/null; then
  echo "ibmcloud cr plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud cr plugin configured properly"
fi

if ! "${BIN_DIR}/kustomize" version; then
  echo "kustomize cli not configured properly" >&2
  exit 1
else
  echo "kustomize cli configured properly"
fi

if ! "${BIN_DIR}/gitu" --version; then
  echo "gitu cli not found" >&2
  exit 1
else
  echo "gitu cli found"
fi

#if ! "${BIN_DIR}/openshift-install" version; then
#  echo "openshift-install cli not found" >&2
#  exit 1
#else
#  echo "openshift-install cli found"
#fi
