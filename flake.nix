{
  description = "A CLI application that intertwines linting and building";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gitignore }:
    flake-utils.lib.eachDefaultSystem (system:

      let
        pname = "treli";
        version = "0.0.1";

        pkgs = import nixpkgs { 
          inherit system;
          config.allowUnfree = true;
        };

        protoc-gen-connect-openapi = pkgs.buildGoModule {
          name = "protoc-gen-connect-openapi";
          src = pkgs.fetchFromGitHub {
            owner = "sudorandom";
            repo = "protoc-gen-connect-openapi";
            rev = "v0.16.1";
            sha256 = "sha256-3XBQCc9H9N/AZm/8J5bJRgBhVtoZKFvbdTB+glHxYdA=";
          };
          vendorHash = "sha256-CIiG/XhV8xxjYY0sZcSvIFcJ1Wh8LyDDwqem2cSSwBA=";
          nativeCheckInputs = with pkgs; [ less ];
        };

        bobgen = pkgs.buildGoModule {
          name = "bobgen";
          src = pkgs.fetchFromGitHub {
            owner = "stephenafamo";
            repo = "bob";
            rev = "v0.31.0";
            sha256 = "sha256-APAckQ+EDAu459NTPXUISLIrcAcX3aQ5B/jrMUEW0EY=";
          };
          vendorHash = "sha256-3blGiSxlKpWH8k0acAXXks8nCdnoWmXLmzPStJmmGcM=";
          subPackages = [
            "gen/bobgen-sqlite"
            "gen/bobgen-psql"
          ];
        };

        treli = pkgs.buildGoModule {
          inherit pname version;
          src = gitignore.lib.gitignoreSource ./.;
          vendorHash = "sha256-sANPwYLGwMcWyMR7Veho81aAMfIQpVzZS5Q9eveR8o8=";
          env.CGO_ENABLED = 0;
        };

      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            git
            nix-update

            # Go backend
            go
            gotools
            gopls
            revive
            bobgen
            
            # Protobuf middleware
            buf
            protoc-gen-go
            protoc-gen-connect-go
            protoc-gen-es
            protoc-gen-connect-openapi
            inotify-tools

            # Svelte frontend
            nodejs_22
          ];
        };

        packages = rec {
          default = treli;

          treli = treli;
        };
      }
    );
}
