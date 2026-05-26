{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "roots-neo4j-start";
  runtimeInputs = [
    pkgs.neo4j
    pkgs.coreutils
    pkgs.git
  ];
  text = builtins.readFile ./main.sh;
}
