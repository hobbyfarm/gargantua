
# Image URL to use all building/pushing image targets
IMG ?= hobbyfarm/gargantua:dev
SVC ?= usersvc
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

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go

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

build-svc:
	docker build -f ./services/${SVC}/Dockerfile -t ${IMG} bin

push-svc: build-svc
	docker push ${IMG}

build-gargantua:
	docker build -t ${IMG} .

push-gargantua: build-gargantua
	docker push ${IMG}