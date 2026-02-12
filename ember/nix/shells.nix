{ inputs', lib', pkgs, self' }: {
  default = pkgs.mkShell {
    packages = pkgs.lib.flatten [
      lib'.envs.ember.dependencies

      (pkgs.lib.attrValues inputs'.roots.packages)
      (pkgs.lib.attrValues self'.packages)
    ];
    shellHook = ''
      echo ""
      echo "  🔥 ember — Pulse React PWA"
      echo ""
      echo "  Available commands:"
      echo "    ember-dev     Run Vite dev server on :5173"
      echo "    ember-build   Production build"
      echo "    ember-lint    Run ESLint"
      echo ""
      echo "  Infrastructure (from roots):"
      echo "    roots-services start   Start PostgreSQL + Redis"
      echo "    roots-services stop    Stop all services"
      echo "    roots-services status  Show service status"
      echo ""

      export VITE_DEV_BACKEND_ORIGIN="http://localhost:8080"
    '';
  };
}
