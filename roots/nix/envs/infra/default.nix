{ pkgs }:
let
  pg = pkgs.postgresql_16.withPackages (ps: [ ps.pgvector ]);
  dependencies = [
    pg
    pkgs.redis
  ];
in {
  inherit dependencies pg;
}
