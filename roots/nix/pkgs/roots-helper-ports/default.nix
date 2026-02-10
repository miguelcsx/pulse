{ pkgs }:
pkgs.writeShellApplication {
  bashOptions = [ ];
  name = "roots-helper-ports";
  runtimeInputs = [ pkgs.coreutils pkgs.lsof pkgs.netcat ];
  text = builtins.readFile ./main.sh;
}
