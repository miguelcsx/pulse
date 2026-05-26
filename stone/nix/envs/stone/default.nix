{ inputs', pkgs, projectPath }:
let
  envars = pkgs.callPackage ./envars.nix { inherit inputs' projectPath; };
  dependencies = [
    pkgs.go_1_25
    pkgs.gopls
    pkgs.gotools
    pkgs.go-tools
    pkgs.govulncheck
    pkgs.git
    pkgs.cacert
  ];
in {
  inherit dependencies envars;
}
