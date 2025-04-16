#!/usr/bin/env bash

url=$(git config --get remote.origin.url)
name=$(basename -s .git "${url}")
tmpv=$(git describe --tags --abbrev=0)
version=${tmpv#v}

cd "$(git rev-parse --show-toplevel)"

echo "Building ${name}-windows-amd64-${version}.exe"
GOOS=windows GOARCH=amd64 go build -o "./build/${name}-windows-amd64-${version}.exe" .

echo "Building ${name}-linux-amd64-${version}"
GOOS=linux GOARCH=amd64 go build -o "./build/${name}-linux-amd64-${version}" .

echo "Building ${name}-linux-amd64-${version}"
GOOS=linux GOARCH=arm64 go build -o "./build/${name}-linux-arm64-${version}" .

echo "Building ${name}-linux-arm-${version}"
GOOS=linux GOARCH=arm go build -o "./build/${name}-linux-arm-${version}" .