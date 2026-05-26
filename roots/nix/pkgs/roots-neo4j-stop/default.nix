{ pkgs }:
pkgs.writeShellApplication {
  name = "roots-neo4j-stop";
  runtimeInputs = [
    pkgs.neo4j
  ];
  text = builtins.readFile ./main.sh;
}
