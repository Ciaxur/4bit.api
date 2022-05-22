#!/usr/bin/env bash

# This script serves to build the api server binary into the build directory.
# Lets get all of the variables sorted out.
SCRIPT_REL_DIR=$(dirname $0)
SCRIPT_DIR=$(realpath $SCRIPT_REL_DIR)
PROJ_ROOT=$(realpath $SCRIPT_REL_DIR/..)
BUILD_DIR=$(realpath $SCRIPT_DIR/../build)

# Extract the binary version based on the checked out branch name.
BIN_VERSION=$(git rev-parse HEAD | head -c 8)
BIN_NAME="4bit-api-$BIN_VERSION"

# Create the build directory and build the binary in there.
echo "Creating build directory."
mkdir -p $BUILD_DIR

echo "Tidy up modules and building the binary to build/$BIN_NAME."
cd $PROJ_ROOT
go mod tidy
go build -o $BUILD_DIR/$BIN_NAME
