#!/usr/bin/env bash
set -eo pipefail

# k8s.io/client-go v0.32.1 for kubernetes release 1.32
KUBERNETES_CLIENT_GO_VERSION="v0.32.1"

SCRIPT_PATH=$(dirname "$(realpath "$0")")
cd "$SCRIPT_PATH"

API_PATH="v3/pkg/apis"
PROPERTY_PATH="v3/pkg/property"
CLIENT_PATH="v3/pkg/client"

echo "Downloading kube_codegen ${KUBERNETES_CLIENT_GO_VERSION}"
curl -LOs https://raw.githubusercontent.com/kubernetes/code-generator/refs/tags/${KUBERNETES_CLIENT_GO_VERSION}/kube_codegen.sh
source kube_codegen.sh

mkdir -p ${CLIENT_PATH}/clientset
mkdir -p ${CLIENT_PATH}/listers
mkdir -p ${CLIENT_PATH}/informers

kube::codegen::gen_helpers \
  "$API_PATH"

kube::codegen::gen_client \
  --output-pkg "github.com/hobbyfarm/gargantua/${CLIENT_PATH}" \
  --output-dir "$CLIENT_PATH" \
  --boilerplate ./hack/boilerplate.go.txt \
  --clientset-name clientset \
  --versioned-name versioned \
  --listers-name listers \
  --informers-name informers \
  --with-watch \
  "$API_PATH"

kube::codegen::gen_helpers \
  "$PROPERTY_PATH"