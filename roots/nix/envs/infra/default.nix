{ pkgs }:
let
  pg = pkgs.postgresql_16.withPackages (ps: [ ps.pgvector ]);
  dependencies = [
    pg
    pkgs.redis
    pkgs.neo4j
  ];
in {
  inherit dependencies pg;
}
