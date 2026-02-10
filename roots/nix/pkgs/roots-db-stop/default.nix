{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "roots-db-stop";
  runtimeInputs = [
    lib'.envs.infra.pg
    pkgs.coreutils
  ];
  text = builtins.readFile ./main.sh;
}
