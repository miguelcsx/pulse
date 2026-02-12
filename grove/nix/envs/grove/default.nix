{ pkgs }:
let
  dependencies = [
    pkgs.nodejs_22
    pkgs.git
    pkgs.cacert
  ];
in {
  inherit dependencies;
}
