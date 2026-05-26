{ lib', pkgs, roots-db-start, roots-db-stop, roots-db-create, roots-redis-start, roots-redis-stop }:
pkgs.writeShellApplication {
  name = "roots-services";
  runtimeInputs = [
    lib'.envs.infra.pg
    pkgs.redis
    pkgs.coreutils
    pkgs.git
    roots-db-start
    roots-db-stop
    roots-db-create
    roots-redis-start
    roots-redis-stop
  ];
  text = builtins.readFile ./main.sh;
}
