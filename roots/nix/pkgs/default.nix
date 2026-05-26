{ lib', pkgs }:
let
  roots-db-start = pkgs.callPackage ./roots-db-start { inherit lib'; };
  roots-db-stop = pkgs.callPackage ./roots-db-stop { inherit lib'; };
  roots-db-create = pkgs.callPackage ./roots-db-create { inherit lib'; };
  roots-redis-start = pkgs.callPackage ./roots-redis-start { };
  roots-redis-stop = pkgs.callPackage ./roots-redis-stop { };
  roots-neo4j-start = pkgs.callPackage ./roots-neo4j-start { inherit lib'; };
  roots-neo4j-stop = pkgs.callPackage ./roots-neo4j-stop { };
in {
  inherit roots-db-start roots-db-stop roots-db-create roots-redis-start roots-redis-stop roots-neo4j-start roots-neo4j-stop;
  roots-services = pkgs.callPackage ./roots-services {
    inherit lib' roots-db-start roots-db-stop roots-db-create roots-redis-start roots-redis-stop roots-neo4j-start roots-neo4j-stop;
  };
  roots-sops = pkgs.callPackage ./roots-sops { };
}
