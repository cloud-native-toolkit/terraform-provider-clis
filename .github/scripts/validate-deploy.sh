#!/usr/bin/env bash

BIN_DIR=$(cat .bin_dir)

export PATH="${BIN_DIR}:${PATH}"

ls -l "${BIN_DIR}"

if ! jq --version; then
  echo "jq not found" >&2
  exit 1
else
  echo "jq cli found"
fi

if ! yq3 --version; then
  echo "yq3 not found" >&2
  exit 1
else
  echo "yq3 cli found"
fi

if ! yq4 --version; then
  echo "yq4 not found" >&2
  exit 1
else
  echo "yq4 cli found"
fi

if ! igc --version; then
  echo "igc not found" >&2
  exit 1
else
  echo "igc cli found"
fi

if ! helm version --short; then
  echo "helm not found" >&2
  exit 1
else
  echo "helm cli found"
fi

if ! argocd version --client; then
  echo "argocd not found" >&2
  exit 1
else
  echo "argocd cli found"
fi

if ! rosa version; then
  echo "rosa cli not configured properly" >&2
  exit 1
else
  echo "rosa cli configured properly"
fi

if ! gh version; then
  echo "gh cli not configured properly" >&2
  exit 1
else
  echo "gh cli configured properly"
fi

if ! kubeseal --version; then
  echo "kubeseal cli not configured properly" >&2
  exit 1
else
  echo "kubeseal cli configured properly"
fi

if ! oc version --client=true; then
  echo "oc cli not configured properly" >&2
  exit 1
else
  echo "oc cli configured properly"
fi

if ! kubectl version --client=true; then
  echo "kubectl cli not configured properly" >&2
  exit 1
else
  echo "kubectl cli configured properly"
fi

if ! ibmcloud version; then
  echo "ibmcloud cli not configured properly" >&2
  exit 1
else
  echo "ibmcloud cli configured properly"
fi

if ! ibmcloud plugin show infrastructure-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud is plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud is plugin configured properly"
fi

if ! ibmcloud plugin show observe-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud ob plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud ob plugin configured properly"
fi

if ! ibmcloud plugin show kubernetes-service 1> /dev/null 2> /dev/null; then
  echo "ibmcloud ks plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud ks plugin configured properly"
fi

if ! ibmcloud plugin show container-registry 1> /dev/null 2> /dev/null; then
  echo "ibmcloud cr plugin not configured properly" >&2
  exit 1
else
  echo "ibmcloud cr plugin configured properly"
fi

if ! kustomize version; then
  echo "kustomize cli not configured properly" >&2
  exit 1
else
  echo "kustomize cli configured properly"
fi

if ! gitu --version; then
  echo "gitu cli not found" >&2
  exit 1
else
  echo "gitu cli found"
fi

if ! openshift-install version; then
  echo "openshift-install cli not found" >&2
  exit 1
else
  echo "openshift-install cli found"
fi

if ! operator-sdk version; then
  echo "operator-sdk cli not found" >&2
  exit 1
else
  echo "operator-sdk cli found"
fi
