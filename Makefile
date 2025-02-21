# Makefile for building and pushing Docker images
SHELL := /bin/bash

# Default values
IMAGE_REGISTRY ?= hobbyfarm
IMAGE_TAG ?= latest
PLATFORMS ?= linux/amd64
DOCKER_COMMAND ?= $(notdir $(shell command -v docker || command -v podman))
DOCKER_BUILD_COMMAND ?= buildx build
SERVICE_BASE_PATH ?= $(shell realpath ./v3/services)
SERVICE_DOCKERFILE ?= $(shell realpath ./v3/Dockerfile)
GARGANTUA_DOCKERFILE ?= $(shell realpath ./Dockerfile)
SERVICES ?=

ifeq ($(DOCKER_COMMAND),podman)
    DOCKER_BUILD_COMMAND := build
    PLATFORM_ARG :=
else
    PLATFORM_ARG := --platform $(PLATFORMS)
endif

# Validate Docker/Podman installation
ifeq ($(DOCKER_COMMAND),)
    $(error Neither "docker" nor "podman" exists on the system)
endif

# If no services are specified, get all service directories
ifeq ($(SERVICES),)
    SERVICES := gargantua $(shell find $(SERVICE_BASE_PATH) -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
endif

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make docker-setup"
	@echo "  make docker-build [SERVICES=service1 service2] [IMAGE_TAG=tag] [IMAGE_REGISTRY=registry] [PLATFORMS=os/arch,os/arch] [PUSH=true]"
	@echo "  make docker-push [SERVICES=service1 service2] [IMAGE_TAG=tag] [IMAGE_REGISTRY=registry] [PLATFORMS=os/arch,os/arch]"
	@echo "  make generate-client"
	@echo "  make generate-protos"
	@echo "  make list-services"
	@echo "  make help"
	@echo ""
	@echo "Targets:"
	@echo "  docker-setup       Sets up and configures Docker Buildx for the first time"
	@echo "  docker-build       Build the docker images with multi-platform support"
	@echo "  docker-push        Build and push the docker images with multi-platform support"
	@echo "  generate-client    Generate kubernetes glue code for go"
	@echo "  generate-protos    Generate go code from proto files"
	@echo "  list-services      Prints the available services for building"
	@echo "  help               Prints help"
	@echo ""
	@echo "Options:"
	@echo "  SERVICES           List of services to build (default: all detected)"
	@echo "  IMAGE_TAG          Tag for the built images (default: latest)"
	@echo "  IMAGE_REGISTRY     Registry for images (default: hobbyfarm)"
	@echo "  PLATFORMS          Comma-separated list of target platforms (default: linux/amd64)"
	@echo "  PUSH               If set to true, push images to the registry (default: false)"

.PHONY: generate-client
generate-client:
	./generate-client.sh

.PHONY: generate-protos
generate-protos:
	./generate-protos.sh

.PHONY: docker-setup
docker-setup:
	@if [ "$(DOCKER_COMMAND)" = "docker" ]; then \
		docker buildx create --use --name hobbyfarm || true; \
		docker buildx inspect --bootstrap; \
	fi

.PHONY: docker-build
docker-build:
	@for service in $(SERVICES); do \
		if [ "$$service" = "gargantua" ]; then \
			echo "Building gargantua..."; \
			$(DOCKER_COMMAND) $(DOCKER_BUILD_COMMAND) $(if $(PUSH),--push,--load) $(PLATFORM_ARG) --file $(GARGANTUA_DOCKERFILE) --tag $(IMAGE_REGISTRY)/$$service:$(IMAGE_TAG) .; \
		else \
			echo "Building $$service..."; \
			image_tag=$(IMAGE_REGISTRY)/$${service/%svc/-service}:$(IMAGE_TAG); \
			$(DOCKER_COMMAND) $(DOCKER_BUILD_COMMAND) $(if $(PUSH),--push,--load) $(PLATFORM_ARG) --build-arg SERVICE_NAME=$$service --file $(SERVICE_DOCKERFILE) --tag $$image_tag .; \
		fi; \
	done

.PHONY: docker-push
docker-push:
	@make docker-build PUSH=true

.PHONY: list-services
list-services:
	@for service in $(SERVICES); do \
		echo "$$service"; \
	done
