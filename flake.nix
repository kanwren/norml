{
  description = "A tool for normalizing YAML files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        trimSpace =
          x:
          let
            m = builtins.match "[ \t\r\n]*(.*[^ \t\r\n])[ \t\r\n]*" x;
          in
          if (m == null) then "" else builtins.head m;
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "norml";
          version = trimSpace (builtins.readFile ./version);
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
