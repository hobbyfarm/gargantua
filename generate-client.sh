#!/usr/bin/env bash
set -eo pipefail

SCRIPT_PATH=$(dirname "$(realpath "$0")")
cd "$SCRIPT_PATH"

API_PATH="v3/pkg/apis"
PROPERTY_PATH="v3/pkg/property"
CLIENT_PATH="v3/pkg/client"

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