#!/bin/bash
set -eo pipefail

SCRIPT_NAME=$(basename "$0")
SCRIPT_PATH=$(dirname "$(realpath "$0")")

IMAGE_REGISTRY="hobbyfarm"
IMAGE_TAG="latest"
PLATFORM="linux/amd64"
DOCKER_PUSH_COMMAND=""
SERVICE_BASE_PATH="$SCRIPT_PATH/v3/services"
SERVICE_DOCKERFILE="$SCRIPT_PATH/v3/Dockerfile"
GARGANTUA_DOCKERFILE="$SCRIPT_PATH/Dockerfile"

function print_help() {
  echo "Usage:"
  echo "  $SCRIPT_NAME [options] [service_name...]"
  echo ""
  echo "Examples:"
  echo "$0                            - builds all images"
  echo "$0 costsvc gargantua          - builds costsvc and gargantua image (names of services match services in $SERVICE_BASE_PATH)"
  echo "$0 -p linux/arm64,linux/arm64 - builds all images with for linux/arm64"
  echo "$0 -r my-hobbyfarm            - builds all images with registry my-hobbyfarm in image name e.g. my-hobbyfarm/cost-service:latest"
  echo "$0 --push                     - pushes the built images to the docker registry"
  echo "$0 -t v0.0.1                  - builds all images with image tag v0.0.1 e.g. hobbyfarm/cost-service:v0.0.1"
  echo ""
  echo "Options:"
  echo "  -p, --platform OS/ARCH[/VARIANT]        define the OS/ARCH[/VARIANT] of the image (default is $PLATFORM)"
  echo "  -r, --image-registry IMAGE_REGISTRY     define the image registry (default is $IMAGE_REGISTRY)"
  echo "  -t, --image-tag IMAGE_TAG               define the image tag (default is $IMAGE_TAG)"
  echo "  --push                                  enable push of built images (disabled per default)"
  echo "  -h, --help                              print help"
}

SERVICES=()

while [[ $# -gt 0 ]]; do
  case $1 in
    -r|--image-registry)
      IMAGE_REGISTRY="$2"
      shift # past argument
      shift # past value
      ;;
    -p|--platform)
      PLATFORM="$2"
      shift # past argument
      shift # past value
      ;;
    -t|--image-tag)
      IMAGE_TAG="$2"
      shift # past argument
      shift # past value
      ;;
    --push)
      DOCKER_PUSH_COMMAND="--push"
      shift # past argument
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    -*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      SERVICES+=("$1") # save positional arg
      shift # past argument
      ;;
  esac
done

set -- "${SERVICES[@]}" # restore positional parameters

DOCKER_COMMAND="docker"
DOCKER_BUILD_COMMAND="buildx build"

# check if podman should be used
if ! command -v "$DOCKER_COMMAND" > /dev/null 2>&1
then
    DOCKER_COMMAND="podman"
    DOCKER_BUILD_COMMAND="build --platform"
fi
# check if podman exists
if ! command -v "$DOCKER_COMMAND" > /dev/null 2>&1
then
    >&2 echo "neither \"docker\" nor \"podman\" exists on the system"
    exit 1
fi

cd "$SCRIPT_PATH"

# no services specified, populate SERVICES array
if (( ${#SERVICES[@]} == 0 )); then
  SERVICES=("gargantua")
  while IFS='' read -r service_name; do SERVICES+=("$service_name"); done < <(find "$SERVICE_BASE_PATH"/* -maxdepth 0 -type d -exec basename {} \;)
fi

# check if all services exist
for service_name in "${SERVICES[@]}"; do
  if [ "$service_name" != "gargantua" ]; then
    service_path="$SERVICE_BASE_PATH/$service_name"
    if [ ! -d "$service_path" ]; then
      echo "service \"$service_name\" does not exist"
      exit 1
    fi
  fi
done

# build specified services
for service_name in "${SERVICES[@]}"; do
  echo "building \"$service_name\""
  echo ""

  if [ "$service_name" = "gargantua" ]; then
    $DOCKER_COMMAND $DOCKER_BUILD_COMMAND $DOCKER_PUSH_COMMAND --platform "$PLATFORM" --file "$GARGANTUA_DOCKERFILE" --tag "${IMAGE_REGISTRY}/${service_name}:${IMAGE_TAG}" .
    echo ""
  else
    image_tag="${IMAGE_REGISTRY}/${service_name/%svc/-service}:${IMAGE_TAG}"
    $DOCKER_COMMAND $DOCKER_BUILD_COMMAND $DOCKER_PUSH_COMMAND --platform "$PLATFORM" --build-arg SERVICE_NAME="$service_name" --file "$SERVICE_DOCKERFILE" --tag "$image_tag" .
    echo ""
  fi
done