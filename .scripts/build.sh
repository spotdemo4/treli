#!/usr/bin/env bash

cd "$(git rev-parse --show-toplevel)"

echo "Building ${pname}-windows-amd64-${version}.exe"
GOOS=windows GOARCH=amd64 go build -o "./build/${pname}-windows-amd64-${version}.exe" .

echo "Building ${pname}-linux-amd64-${version}"
GOOS=linux GOARCH=amd64 go build -o "./build/${pname}-linux-amd64-${version}" .

echo "Building ${pname}-linux-amd64-${version}"
GOOS=linux GOARCH=arm64 go build -o "./build/${pname}-linux-arm64-${version}" .

echo "Building ${pname}-linux-arm-${version}"
GOOS=linux GOARCH=arm go build -o "./build/${pname}-linux-arm-${version}" .