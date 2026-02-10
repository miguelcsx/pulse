{ pkgs }:
pkgs.writeShellApplication {
  bashOptions = [ ];
  name = "roots-sops";
  runtimeInputs = [ pkgs.gnugrep pkgs.jq pkgs.sops pkgs.age ];
  text = builtins.readFile ./main.sh;
}
