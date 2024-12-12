#!/bin/bash
GO_VERSION="1.23.1-cgo"
APP_DIR="$PWD/app"
MAIN_FILE_PATH="/app/cmd"
BINARY_NAME="simplql"

rm -rf binary
cd $APP_DIR

APP_DIR="$PWD"

CURRENT_VERSION=`jq -r .hash $APP_DIR/pkg/configurationAndInitialization/version/version.json`
DATE=`date +%Y-%m-%d`
echo "Building $BINARY_NAME ($CURRENT_VERSION)"

if [[ $* == *--dev* ]]; then
  DEV_BUILD="true"
else
  DEV_BUILD="false"
fi
if [[ $* == *--arch* ]]; then
   if [[ $* == *linux* ]]; then
      USE_OS="linux"
   elif [[ $* == *mac* ]]; then
     USE_OS="darwin"
   elif [[ $* == *windows* ]]; then
      USE_OS="windows"
   fi
   docker run --rm -v "$APP_DIR:/app" -e CGO_LDFLAGS="-s -w" vo1d/gocompiler:$GO_VERSION $BINARY_NAME $CURRENT_VERSION $USE_OS $DEV_BUILD $MAIN_FILE_PATH && \
   mv binary/binary/ ../binary
   rm -rf binary
   echo "Generated $BINARY_NAME ($USE_OS/$CURRENT_VERSION) Binary"
   cd -
   exit 0
   if [[ $* == *all* ]]; then
      USE_OS="linux"
      docker run  --rm -v "$APP_DIR:/app" -e CGO_LDFLAGS="-s -w" vo1d/gocompiler:$GO_VERSION $BINARY_NAME $CURRENT_VERSION $USE_OS $DEV_BUILD $MAIN_FILE_PATH
      echo "Generated Linux Binary 🐧"
      USE_OS="darwin"
      docker run --rm -v "$APP_DIR:/app" -e CGO_LDFLAGS="-s -w" vo1d/gocompiler:$GO_VERSION $BINARY_NAME $CURRENT_VERSION $USE_OS $DEV_BUILD $MAIN_FILE_PATH && \
      echo "Generated Mac OSX Binary 🍎"
      USE_OS="windows"
      docker run --rm -v "$APP_DIR:/app" -e CGO_LDFLAGS="-s -w" vo1d/gocompiler:$GO_VERSION $BINARY_NAME $CURRENT_VERSION $USE_OS $DEV_BUILD $MAIN_FILE_PATH && \
      echo "Generated Windows Binary 🪟"
      cd -
      exit 0
   fi
else
   echo "You should run ./scripts/generate-binary.sh --arch <linux/mac/windows/all>"
   echo "You can also run '--dev' to create a 'Devlopment Build'"
   cd -
   exit 1
fi
