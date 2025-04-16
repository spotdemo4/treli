{
  description = "Development environment multiplexer";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = {nixpkgs, ...}: let
    pname = "treli";
    version = "''''0.0.8";

    supportedSystems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];
    forSystem = f:
      nixpkgs.lib.genAttrs supportedSystems (
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
          packages = with pkgs;
            [
              git
              nix-update

              # Go backend
              go
              gotools
              gopls
              revive
            ]
            # Use .scripts
            ++ map (
              x: (
                pkgs.writeShellApplication {
                  name = "${pname}-${(lib.nameFromURL (baseNameOf x) ".")}";
                  text = builtins.readFile x;
                }
              )
            ) (pkgs.lib.filesystem.listFilesRecursive ./.scripts);
        };
      }
    );

    formatter = forSystem ({pkgs, ...}: pkgs.alejandra);

    packages = forSystem (
      {pkgs, ...}: rec {
        default = treli;
        treli = pkgs.buildGoModule {
          inherit pname version;
          src = ./.;
          vendorHash = "sha256-+f70tVKvvsXVDpCVEmGIOcpP6tIeSjGL14/wd8P+/GA=";
        };
      }
    );
  };
}
