#!/bin/bash
set -eo pipefail

BASE_IMAGE="golang:1.23.6-alpine3.21"
IMAGE_TAG="hobbyfarm-proto-builder:latest"
PROTOC_VERSION="29.3"
PROTOC_GEN_GO_VERSION="v1.36.5"
PROTOC_GEN_GO_GRPC_VERSION="v1.5.1"

PROJECT_PATH="/github.com/hobbyfarm/gargantua"
PROTO_DIR="${PROJECT_PATH}/protos"

# build the container
docker build -t ${IMAGE_TAG} - <<EOF
FROM ${BASE_IMAGE}
ENV PATH="/go/bin:\${PATH}"

RUN \
  apk add --no-cache curl unzip && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} && \
  ARCH=\$(uname -m) && \
  if [ "\$ARCH" = "aarch64" ]; then ARCH="aarch_64"; fi && \
  curl -sLo /tmp/protoc-${PROTOC_VERSION}-linux-\${ARCH}.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-\${ARCH}.zip && \
  unzip -q /tmp/protoc-${PROTOC_VERSION}-linux-\${ARCH}.zip -d /usr/local && \
  chmod +x /usr/local/bin/protoc && \
  rm /tmp/protoc-${PROTOC_VERSION}-linux-\${ARCH}.zip
EOF

# run the container
docker run --rm \
  -v "$(pwd):${PROJECT_PATH}" \
  -v "$(pwd)/v3/protos:${PROTO_DIR}" \
  -w "${PROJECT_PATH}" \
  --entrypoint sh \
  ${IMAGE_TAG} \
  -c "\
  echo \"Compiling proto files\" && \
  protoc -I ${PROTO_DIR} \
    --go_out=./protos \
    --go_opt=paths=source_relative \
    --go-grpc_out=./protos \
    --go-grpc_opt=paths=source_relative \
    \$(find ${PROTO_DIR} -name "*.proto") && \
  echo \"Done\""