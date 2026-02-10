{ inputs', lib', pkgs, self' }: {
  default = pkgs.mkShell {
    packages = pkgs.lib.flatten [
      lib'.envs.stone.dependencies
      lib'.envs.stone.envars

      (pkgs.lib.attrValues inputs'.roots.packages)
      (pkgs.lib.attrValues self'.packages)
    ];
    env = {
      GOFLAGS = "-trimpath";
    };
    shellHook = ''
      echo ""
      echo "  🪨 stone — Pulse Go API backend"
      echo ""
      echo "  Available commands:"
      echo "    stone-dev       Run dev server on :8080"
      echo "    stone-build     Build static binaries"
      echo "    stone-test      Run tests"
      echo "    stone-lint      Run go vet + govulncheck"
      echo "    stone-migrate   Run database migrations"
      echo "    stone-seed      Seed demo data"
      echo ""
      echo "  Infrastructure (from roots):"
      echo "    roots-services start   Start PostgreSQL + Redis"
      echo "    roots-services stop    Stop all services"
      echo "    roots-services status  Show service status"
      echo ""

      source stone-envars dev
    '';
  };
}
