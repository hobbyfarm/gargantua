
# Image URL to use all building/pushing image targets
IMG ?= hobbyfarm/gargantua:dev
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/gargantua main.go
	go build -o bin/gargantua/services/authservice services/authservice/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go
	go run ./services/authservice/main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate:
	./generate-client.sh

# Build the docker image
docker-build: fmt vet
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

protos:
	protoc -I ./protos protos/authn/authn.proto --go_out ./protos --go_opt paths=source_relative --go-grpc_out=require_unimplemented_servers=false:./protos protos/authn/authn.proto --go-grpc_opt paths=source_relative
	protoc -I ./protos protos/k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto --go_out ./protos --go_opt paths=source_relative --go-grpc_out=require_unimplemented_servers=false:./protos protos/k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto --go-grpc_opt paths=source_relative
