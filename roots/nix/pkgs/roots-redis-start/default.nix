{ pkgs }:
pkgs.writeShellApplication {
  name = "roots-redis-start";
  runtimeInputs = [
    pkgs.redis
    pkgs.coreutils
    pkgs.git
  ];
  text = builtins.readFile ./main.sh;
}
