{
  description = "A CLI application that intertwines linting and building";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, gitignore }:
    let
      pname = "treli";
      version = "0.0.5";

      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        inherit system;
        pkgs = import nixpkgs {
          inherit system;
        };
      });
    in
    {
      devShells = forSystem ({ pkgs, ... }: {
        default = pkgs.mkShell {
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
              name = "${pname}-update";

              text = ''
                git_root=$(git rev-parse --show-toplevel)
                cd "''${git_root}"

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
              '';
            })

            # Bump version
            (writeShellApplication {
              name = "${pname}-bump";

              text = ''
                git_root=$(git rev-parse --show-toplevel)
                cd "''${git_root}"

                next_version=$(echo "${version}" | awk -F. -v OFS=. '{$NF += 1 ; print}')

                nix-update --flake --version "''${next_version}" default
                git add flake.nix
                git commit -m "bump: v${version} -> v''${next_version}"
                git push origin main

                git tag -a "v''${next_version}" -m "bump: v${version} -> v''${next_version}"
                git push origin "v''${next_version}"
              '';
            })

            # Lint
            (writeShellApplication {
              name = "${pname}-lint";

              text = ''
                git_root=$(git rev-parse --show-toplevel)
                cd "''${git_root}"

                # Go
                revive -config revive.toml -set_exit_status ./...

                # Nix
                nix flake check --all-systems
              '';
            })

            # Build
            (writeShellApplication {
              name = "${pname}-build";

              text = ''
                git_root=$(git rev-parse --show-toplevel)
                cd "''${git_root}"

                echo "Building ${pname}-windows-amd64-${version}.exe"
                GOOS=windows GOARCH=amd64 go build -o "./build/${pname}-windows-amd64-${version}.exe" .

                echo "Building ${pname}-linux-amd64-${version}"
                GOOS=linux GOARCH=amd64 go build -o "./build/${pname}-linux-amd64-${version}" .

                echo "Building ${pname}-linux-amd64-${version}"
                GOOS=linux GOARCH=arm64 go build -o "./build/${pname}-linux-arm64-${version}" .

                echo "Building ${pname}-linux-arm-${version}"
                GOOS=linux GOARCH=arm go build -o "./build/${pname}-linux-arm-${version}" .
              '';
            })
          ];
        };
      });

      packages = forSystem ({ pkgs, ... }: rec {
        default = treli;
        treli = pkgs.buildGoModule {
          inherit pname version;
          src = gitignore.lib.gitignoreSource ./.;
          vendorHash = "sha256-QdW0DlOscw49bKD/ZaI1jSAkjjOHiB3WMedpi+Ni3iM=";
        };
      });
    };
}
