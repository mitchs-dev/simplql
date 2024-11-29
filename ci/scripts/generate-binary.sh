#!/bin/bash
APP_DIR="app/"
MAIN_FILE_PATH="cmd/main.go"
BINARY_DIR="../binary"
BINARY_NAME="simplql"

CURRENT_VERSION=`jq -r .hash app/pkg/configurationAndInitalization/version/version.json`
DATE=`date +%Y-%m-%d`
echo "Building $BINARY_NAME ($CURRENT_VERSION)"
echo "Changing Directory to $APP_DIR"
cd $APP_DIR
mkdir -p $BINARY_DIR/sha256
if [[ $* == *--arch* ]]; then
   if [[ $* == *linux* ]]; then
      if [[ $* == *--dev* ]]; then
         env GOOS=linux GOARCH=amd64 go build -v -o $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION-$DATE-DEVBUILD $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION-$DATE-DEVBUILD  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-linux-amd64-$CURRENT_VERSION-$DATE-DEVBUILD.sha256
      else
         env GOOS=linux GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-linux-amd64-$CURRENT_VERSION.sha256
      fi
      echo "Generated Linux Binary 🐧"
      cd -
      exit 0
   elif [[ $* == *mac* ]]; then
      if [[ $* == *--dev* ]]; then
         env GOOS=darwin GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION-$DATE-DEVBUILD $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION-$DATE-DEVBUILD  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION-$DATE-DEVBUILD.sha256
         env GOOS=darwin GOARCH=arm64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION-$DATE-DEVBUILD $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION-$DATE-DEVBUILD  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION-$DATE-DEVBUILD.sha256

      else
         env GOOS=darwin GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION.sha256
         env GOOS=darwin GOARCH=arm64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION.sha256
      fi
      echo "Generated Mac OSX Binary 🍎"
      cd -
      exit 0
   elif [[ $* == *windows* ]]; then
      if [[ $* == *--dev*  ]]; then
         env GOOS=windows GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION-$DATE-DEVBUILD.exe $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION-$DATE-DEVBUILD.exe  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-$CURRENT_VERSION-$DATE-DEVBUILD-windows.sha256
      else
         env GOOS=windows GOARCH=amd64  go build -o $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION.exe $MAIN_FILE_PATH
         sha256sum $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION.exe  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-$CURRENT_VERSION-windows.sha256
      fi
      echo "Generated Windows Binary 🪟"
      cd -
      exit 0
   elif [[ $* == *all* ]]; then
      if [[ $* == *--dev* ]]; then
         echo "'--dev' is currently unsupported with '--all' - Please use architecture specific flag (--arch <linux/mac/windows>) instead"
         cd -
         exit 0
      fi
      env GOOS=linux GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION $MAIN_FILE_PATH
      sha256sum $BINARY_DIR/$BINARY_NAME-linux-amd64-$CURRENT_VERSION  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-linux-amd64-$CURRENT_VERSION.sha256
      echo "Generated Linux Binary 🐧"
      cd -
      env GOOS=darwin GOARCH=amd64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION $MAIN_FILE_PATH
      sha256sum $BINARY_DIR/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-amd64-$CURRENT_VERSION.sha256
      env GOOS=darwin GOARCH=arm64 go build -o $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION $MAIN_FILE_PATH
      sha256sum $BINARY_DIR/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-darwin-arm64-$CURRENT_VERSION.sha256
      echo "Generated Mac OSX Binary 🍎"
      cd -
      env GOOS=windows GOARCH=amd64  go build -o $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION.exe $MAIN_FILE_PATH
      sha256sum $BINARY_DIR/$BINARY_NAME-$CURRENT_VERSION.exe  | cut -d ' ' -f 1 > $BINARY_DIR/sha256/$BINARY_NAME-$CURRENT_VERSION-windows.sha256
      echo "Generated Windows Binary 🪟"
      cd -
      exit 0
   fi
else
   echo "You should run ./scripts/generate-binary.sh --arch <linux/mac/windows/all>"
   echo "You can also run '--dev' to create a 'Devlopment Build'"
   cd -
   exit 0
fi
