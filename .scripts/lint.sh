#!/usr/bin/env bash

cd "$(git rev-parse --show-toplevel)"

# Go
revive -config revive.toml -set_exit_status ./...

# Nix
nix fmt -- flake.nix --check
nix flake check --all-systems