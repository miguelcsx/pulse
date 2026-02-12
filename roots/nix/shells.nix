{ lib', pkgs, self' }: {
  default = pkgs.mkShell {
    packages = pkgs.lib.flatten [
      lib'.envs.infra.dependencies
      (pkgs.lib.attrValues self'.packages)
    ];
    shellHook = ''
      echo ""
      echo "  🌱 roots — Pulse shared infrastructure"
      echo ""
      echo "  Available commands:"
      echo "    roots-db-start      Start PostgreSQL (pgvector)"
      echo "    roots-db-stop       Stop PostgreSQL"
      echo "    roots-db-create     Create pulse_dev database"
      echo "    roots-redis-start   Start Redis"
      echo "    roots-redis-stop    Stop Redis"
      echo "    roots-services      Start/stop all services"
      echo ""
    '';
  };
}
