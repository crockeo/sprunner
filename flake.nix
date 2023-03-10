{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    flake-utils,
    gomod2nix,
    nixpkgs
  }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
          ];
        };
      in
        rec {
          packages = rec {
            depShell = pkgs.mkShell {
              packages = [
                pkgs.gomod2nix
              ];
            };

            server = pkgs.buildGoApplication {
              name = "sprunner";
              modules = ./gomod2nix.toml;
              src = ./.;
            };

            default = server;
          };
        }
    );
}
