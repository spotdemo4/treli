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

        app = pkgs.buildGoModule {
          inherit pname version;
          src = gitignore.lib.gitignoreSource ./.;
          vendorHash = "sha256-pUFGkQzbv2sRZ/0rjehL2t+gwZsLWh/TAFYfakCs5g8=";
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

            # Update
            (writeShellApplication {
              name = "ts-update";

              text = ''
                git_root=$(git rev-parse --show-toplevel)

                cd "''${git_root}"
                go get -u
                go mod tidy
                if ! git diff --exit-code go.mod go.sum; then
                  git add go.mod
                  git add go.sum
                  git commit -m "build(server): updated go dependencies"
                fi

                cd "''${git_root}"
                nix-update --flake --version=skip default
                if ! git diff --exit-code flake.nix; then
                  git add flake.nix
                  git commit -m "build(nix): updated nix hashes"
                fi
              '';
            })

            # Bump version
            (writeShellApplication {
              name = "ts-bump";

              text = ''
                git_root=$(git rev-parse --show-toplevel)
                next_version=$(echo "${version}" | awk -F. -v OFS=. '{$NF += 1 ; print}')

                cd "''${git_root}"
                nix-update attribute --flake --version "''${next_version}" default
                git add flake.nix
                git commit -m "bump: v${version} -> v''${next_version}"
                git push origin main

                git tag -a "v''${next_version}" -m "bump: v${version} -> v''${next_version}"
                git push origin "v''${next_version}"
              '';
            })

            # Lint
            (writeShellApplication {
              name = "ts-lint";

              text = ''
                git_root=$(git rev-parse --show-toplevel)

                cd "''${git_root}/server"
                echo "Linting server"
                revive -config revive.toml -formatter friendly ./...
              '';
            })

            # Build
            (writeShellApplication {
              name = "ts-build";

              text = ''
                git_root=$(git rev-parse --show-toplevel)

                cd "''${git_root}"
                echo "Building ${pname}-windows-amd64-${version}.exe"
                GOOS=windows GOARCH=amd64 go build -o "../build/${pname}-windows-amd64-${version}.exe" .

                echo "Building ${pname}-linux-amd64-${version}"
                GOOS=linux GOARCH=amd64 go build -o "../build/${pname}-linux-amd64-${version}" .

                echo "Building ${pname}-linux-amd64-${version}"
                GOOS=linux GOARCH=arm64 go build -o "../build/${pname}-linux-arm64-${version}" .

                echo "Building ${pname}-linux-arm-${version}"
                GOOS=linux GOARCH=arm go build -o "../build/${pname}-linux-arm-${version}" .
              '';
            })
          ];
        };

        packages = rec {
          default = app;

          treli = app;
        };
      }
    );
}
