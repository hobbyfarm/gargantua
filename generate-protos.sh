#!/bin/bash

PROJECT_PATH="/github.com/hobbyfarm/gargantua"
PROTO_DIR="${PROJECT_PATH}/protos"

docker run --rm \
  -v "$(pwd):${PROJECT_PATH}" \
  -v "$(pwd)/protos:${PROTO_DIR}" \
  -w "${PROJECT_PATH}" \
  --entrypoint bash \
  golang:latest \
  -c "apt-get update && apt-get install -y protobuf-compiler && \
  export PATH=\$PATH:/go/bin && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
  find ${PROTO_DIR} -type f -name '*.proto' -exec bash -c ' \
    protoc -I ${PROTO_DIR} \
    --go_out=./protos \
    --go_opt=paths=source_relative \
    --go-grpc_out=./protos \
    --go-grpc_opt=paths=source_relative \
    {}' \;"