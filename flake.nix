{
  description = "Sign-in to AWS IAM Identity Center and credentials for the roles it grants";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      # shared by `nix fmt` and the treefmt check
      formatTools =
        pkgs: with pkgs; [
          treefmt
          nixfmt
          go
        ];
    in
    {
      packages = forAllSystems (pkgs: {
        default = pkgs.callPackage ./package.nix { };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.gopls
            pkgs.fzf
          ]
          ++ formatTools pkgs;
        };
      });

      formatter = forAllSystems (
        pkgs:
        pkgs.writeShellApplication {
          name = "treefmt";
          runtimeInputs = formatTools pkgs;
          text = ''exec treefmt "$@"'';
        }
      );

      checks = forAllSystems (pkgs: {
        # runs the tests
        package = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
        # copy out of the read-only store so treefmt can rewrite in place,
        # then fail if anything changed
        treefmt = pkgs.runCommand "check-treefmt" { nativeBuildInputs = formatTools pkgs; } ''
          cp -r --no-preserve=mode,ownership ${self} src
          cd src
          HOME=$TMPDIR treefmt --no-cache --fail-on-change --tree-root .
          touch $out
        '';
      });
    };
}
