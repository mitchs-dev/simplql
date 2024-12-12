#/bin/bash
TAG_LATEST="true"
DEV_BUILD="true"

BUILD_OS="linux"
BUILD_ARCH="amd64"
IMAGE_NAME="simplql"
IMAGE_AUTHOR="Mitchell Stanton <simplql@mitchs.dev>"
IMAGE_DESCRIPTION="A lightweight SQLite server designed to provide a simple and efficient solution for applications that require a basic database functionality."
DOCKERFILLE_PATH="ci/images/Dockerfile"
BUILD_CONTEXT="."

BUILD_VERSION=`jq -r .symantic app/pkg/configurationAndInitialization/version/version.json`
BUILD_SHA=`jq -r .hash app/pkg/configurationAndInitialization/version/version.json`
BUILD_DATE=`date -u +%Y-%m-%d`

if [[ "$DEV_BUILD" == "true" ]]; then
    DEV_BUILD_FLAG="-DEVBUILD"
    else
    DEV_BUILD_FLAG=""
fi

if [[ "$TAG_LATEST" == "true" ]]; then
  echo "Building Docker Image $IMAGE_NAME:{$BUILD_VERSION,$BUILD_SHA,latest}"
  docker build -t "$IMAGE_NAME:$BUILD_VERSION" \
    -t "$IMAGE_NAME:$BUILD_SHA" \
    -t "$IMAGE_NAME:latest" \
    -f "$DOCKERFILLE_PATH" \
    --build-arg IMAGE_NAME="$IMAGE_NAME" \
    --build-arg IMAGE_AUTHOR="$IMAGE_AUTHOR" \
    --build-arg IMAGE_DESCRIPTION="$IMAGE_DESCRIPTION" \
    --build-arg BUILD_DATE="$BUILD_DATE" \
    --build-arg BUILD_SHA="$BUILD_SHA" \
    --build-arg BUILD_VERSION="$BUILD_VERSION" \
    --build-arg BUILD_OS="$BUILD_OS" \
    --build-arg BUILD_ARCH="$BUILD_ARCH" \
    --build-arg DEV_BUILD="$DEV_BUILD" \
    --build-arg DEV_BUILD_FLAG="$DEV_BUILD_FLAG" \
    "$BUILD_CONTEXT"
else
  echo "Building Docker Image $IMAGE_NAME:{$BUILD_VERSION,$BUILD_SHA}"
  docker build -t "$IMAGE_NAME:$BUILD_VERSION" \
    -t "$IMAGE_NAME:$BUILD_SHA" \
    -f "$DOCKERFILLE_PATH" \
    --build-arg IMAGE_NAME="$IMAGE_NAME" \
    --build-arg IMAGE_AUTHOR="$IMAGE_AUTHOR" \
    --build-arg IMAGE_DESCRIPTION="$IMAGE_DESCRIPTION" \
    --build-arg BUILD_DATE="$BUILD_DATE" \
    --build-arg BUILD_SHA="$BUILD_SHA" \
    --build-arg BUILD_VERSION="$BUILD_VERSION" \
    --build-arg BUILD_OS="$BUILD_OS" \
    --build-arg BUILD_ARCH="$BUILD_ARCH" \
    --build-arg DEV_BUILD="$DEV_BUILD" \
    --build-arg DEV_BUILD_FLAG="$DEV_BUILD_FLAG" \
    "$BUILD_CONTEXT"
fi

