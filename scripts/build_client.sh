#!/usr/bin/env bash

SCRIPT_REL_DIR="$(dirname $0)"
PROJ_ROOT="$(realpath "$SCRIPT_REL_DIR/..")"
BUILD_DIR=$(realpath "$SCRIPT_REL_DIR"/../build)
BIN_NAME="4bit-client"

# Create the build directory and build the binary in there.
echo "Creating build directory."
mkdir -p "$BUILD_DIR"

# Build the client cli binary.
cd "$PROJ_ROOT" || exit 1
go mod tidy
go build -o "$BUILD_DIR/$BIN_NAME" "$PROJ_ROOT/client/cmd"
