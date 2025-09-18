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
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-X2qVf3/9WvWkS6HjGVw4Ns4WUhjPm539ve6qr8u2Ys0=";
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            pre-commit
            go
            go-tools
            golangci-lint
            goreleaser
          ];
        };
      }
    );
}
