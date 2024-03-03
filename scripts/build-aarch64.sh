#!/usr/bin/env bash

# Thin wrapper which builds a server binary for aarch64.
SCRIPT_REL_DIR=$(dirname "$0")
SCRIPT_DIR=$(realpath "$SCRIPT_REL_DIR")

# Architecture envs.
export GOARCH=arm64
export GOOS=linux
$SCRIPT_DIR/build.sh
