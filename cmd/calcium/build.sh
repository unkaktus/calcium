#!/bin/bash
set -e

PROGRAM=calcium
VERSION=$(git describe --exact-match --tags)
echo $VERSION
LDFLAGS="-X main.version=$VERSION"

env GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o $PROGRAM-linux-amd64
env GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o $PROGRAM-linux-arm64
env GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o $PROGRAM-darwin-amd64
env GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o $PROGRAM-darwin-arm64
