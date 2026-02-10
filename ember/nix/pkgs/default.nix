{ lib', pkgs }: {
  ember-dev = pkgs.callPackage ./ember-dev { inherit lib'; };
  ember-build = pkgs.callPackage ./ember-build { inherit lib'; };
  ember-lint = pkgs.callPackage ./ember-lint { inherit lib'; };
}
