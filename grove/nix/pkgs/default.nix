{ lib', pkgs }: {
  grove-dev = pkgs.callPackage ./grove-dev { inherit lib'; };
  grove-build = pkgs.callPackage ./grove-build { inherit lib'; };
  grove-lint = pkgs.callPackage ./grove-lint { inherit lib'; };
}
