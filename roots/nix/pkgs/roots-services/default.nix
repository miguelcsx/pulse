{ lib', pkgs, roots-db-start, roots-db-stop, roots-db-create, roots-redis-start, roots-redis-stop, roots-neo4j-start, roots-neo4j-stop }:
pkgs.writeShellApplication {
  name = "roots-services";
  runtimeInputs = [
    lib'.envs.infra.pg
    pkgs.redis
    pkgs.neo4j
    pkgs.coreutils
    pkgs.git
    roots-db-start
    roots-db-stop
    roots-db-create
    roots-redis-start
    roots-redis-stop
    roots-neo4j-start
    roots-neo4j-stop
  ];
  text = builtins.readFile ./main.sh;
}
