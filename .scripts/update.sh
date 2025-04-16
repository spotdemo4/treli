#!/usr/bin/env bash

cd "$(git rev-parse --show-toplevel)"

# Go
go get -u
go mod tidy
if ! git diff --exit-code go.mod go.sum; then
    git add go.mod
    git add go.sum
    git commit -m "build(go): updated go dependencies"
fi

# Nix
nix-update --flake --version=skip default
if ! git diff --exit-code flake.nix; then
    git add flake.nix
    git commit -m "build(nix): updated nix hashes"
fi