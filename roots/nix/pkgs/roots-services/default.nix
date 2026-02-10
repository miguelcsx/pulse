{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "roots-services";
  runtimeInputs = [
    lib'.envs.infra.pg
    pkgs.redis
    pkgs.coreutils
    pkgs.git
  ];
  text = builtins.readFile ./main.sh;
}
