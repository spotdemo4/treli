{
  description = "Development environment multiplexer";

  nixConfig = {
    extra-substituters = [
      "https://treli.cachix.org"
    ];
    extra-trusted-public-keys = [
      "treli.cachix.org-1:1v82azXKBNYjWTWm2B80GUPYpCnb1uZrhw5O3wbP+e0="
    ];
  };

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = {nixpkgs, ...}: let
    pname = "treli";
    version = "0.0.9";

    build-systems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];
    host-systems = [
      {
        GOOS = "linux";
        GOARCH = "amd64";
      }
      {
        GOOS = "linux";
        GOARCH = "arm64";
      }
      {
        GOOS = "linux";
        GOARCH = "arm";
      }
      {
        GOOS = "windows";
        GOARCH = "amd64";
      }
      {
        GOOS = "darwin";
        GOARCH = "amd64";
      }
      {
        GOOS = "darwin";
        GOARCH = "arm64";
      }
    ];
    forSystem = f:
      nixpkgs.lib.genAttrs build-systems (
        system:
          f {
            inherit system;
            pkgs = import nixpkgs {
              inherit system;
            };
          }
      );
  in {
    devShells = forSystem (
      {pkgs, ...}: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            git
            nix-update

            # Go
            go
            gotools
            gopls
            revive
          ];
        };
      }
    );

    checks = forSystem ({pkgs, ...}: {
      nix = with pkgs;
        runCommandLocal "check-nix" {
          nativeBuildInputs = with pkgs; [
            alejandra
          ];
        } ''
          cd ${./.}
          alejandra -c .
          touch $out
        '';

      server = with pkgs;
        runCommandLocal "check-server" {
          nativeBuildInputs = with pkgs; [
            revive
          ];
        } ''
          cd ${./.}
          revive -config revive.toml -set_exit_status ./...
          touch $out
        '';
    });

    apps = forSystem ({pkgs, ...}: {
      update = {
        type = "app";
        program = pkgs.lib.getExe (pkgs.writeShellApplication {
          name = "update";
          runtimeInputs = with pkgs; [
            git
            nix
            go
            nix-update
          ];
          text = builtins.readFile ./.scripts/update.sh;
        });
      };

      bump = {
        type = "app";
        program = pkgs.lib.getExe (pkgs.writeShellApplication {
          name = "bump";
          runtimeInputs = with pkgs; [
            git
            nix-update
          ];
          text = builtins.readFile ./.scripts/bump.sh;
        });
      };
    });

    formatter = forSystem ({pkgs, ...}: pkgs.alejandra);

    packages = forSystem (
      {pkgs, ...}: let
        treli = pkgs.buildGoModule {
          inherit pname version;
          src = ./.;
          vendorHash = "sha256-+f70tVKvvsXVDpCVEmGIOcpP6tIeSjGL14/wd8P+/GA=";
          env.CGO_ENABLED = 0;
        };
      in
        {
          default = treli;
        }
        // builtins.listToAttrs (builtins.map (x: {
            name = "${pname}-${x.GOOS}-${x.GOARCH}";
            value = treli.overrideAttrs {
              nativeBuildInputs =
                treli.nativeBuildInputs
                ++ [
                  pkgs.rename
                ];
              env.CGO_ENABLED = 0;
              env.GOOS = x.GOOS;
              env.GOARCH = x.GOARCH;

              installPhase = ''
                runHook preInstall

                mkdir -p $out/bin
                find $GOPATH/bin -type f -exec mv -t $out/bin {} +
                rename 's/(.+\/)(.+?)(\.[^.]*$|$)/$1${pname}-${x.GOOS}-${x.GOARCH}-${version}$3/' $out/bin/*

                runHook postInstall
              '';
            };
          })
          host-systems)
    );
  };
}
