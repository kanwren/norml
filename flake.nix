{
  description = "A tool for normalizing YAML files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "norml";
          version = "0.0.1";
          src = ./.;
          vendorHash = "sha256-zwbOnOH71/QDjScypJiLOMZ/nOQROFqdXkcn8TxJHJg=";
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            go-tools
            goreleaser
          ];
        };
      }
    );
}
