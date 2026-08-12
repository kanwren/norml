{
  description = "A tool for normalizing YAML files";

  inputs = {
    systems.url = "github:nix-systems/default";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, ... }@inputs:
    let
      systems = import inputs.systems;
      pkgsEval = inputs.nixpkgs.lib.genAttrs systems (
        system:
        import inputs.nixpkgs {
          inherit system;
          overlays = [ self.overlays.default ];
        }
      );
      eachSystem = f: builtins.mapAttrs f pkgsEval;
    in
    {
      overlays.default = final: prev: {
        norml = final.buildGoModule {
          pname = "norml";
          version =
            let
              m = builtins.match "[ \t\r\n]*(.*[^ \t\r\n])[ \t\r\n]*" (builtins.readFile ./version);
            in
            if (m == null) then "" else builtins.head m;
          src = ./.;
          vendorHash = "sha256-JffM9G4m5tIPERZLKMKAHWssTtYqAafU3olcW+DNPss=";
          meta = {
            homepage = "https://github.com/kanwren/norml";
            mainProgram = "norml";
            description = "A tool for normalizing YAML files";
            license = prev.lib.licenses.mit;
          };
        };
      };

      packages = eachSystem (
        system: pkgs: {
          default = self.packages.${system}.norml;
          inherit (pkgs) norml;
        }
      );

      apps = eachSystem (
        system: pkgs: {
          default =
            let
              pkg = self.packages.${system}.default;
            in
            {
              type = "app";
              program = pkgs.lib.getExe pkg;
              inherit (pkg) meta;
            };
        }
      );

      devShells = eachSystem (
        _: pkgs: {
          default = pkgs.mkShell {
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
    };
}
