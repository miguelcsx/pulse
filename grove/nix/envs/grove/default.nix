{ pkgs }:
let
  dependencies = [
    pkgs.nodejs_20
    pkgs.git
  ];
in {
  inherit dependencies;
}
