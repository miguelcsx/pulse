{ lib', pkgs }: {
  stone-dev = pkgs.callPackage ./stone-dev { inherit lib'; };
  stone-build = pkgs.callPackage ./stone-build { inherit lib'; };
  stone-test = pkgs.callPackage ./stone-test { inherit lib'; };
  stone-lint = pkgs.callPackage ./stone-lint { inherit lib'; };
  stone-migrate = pkgs.callPackage ./stone-migrate { inherit lib'; };
  stone-seed = pkgs.callPackage ./stone-seed { inherit lib'; };
}
