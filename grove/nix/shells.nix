{ inputs', lib', pkgs, self' }: {
  default = pkgs.mkShell {
    packages = pkgs.lib.flatten [
      lib'.envs.grove.dependencies

      (pkgs.lib.attrValues inputs'.roots.packages)
      (pkgs.lib.attrValues self'.packages)
    ];
    shellHook = ''
      echo ""
      echo "  🌿 grove — Pulse landing page"
      echo ""
      echo "  Available commands:"
      echo "    grove-dev     Run Vite dev server on :5174"
      echo "    grove-build   Production build"
      echo "    grove-lint    Run ESLint"
      echo ""
      echo "  Infrastructure (from roots):"
      echo "    roots-services start   Start PostgreSQL + Redis"
      echo "    roots-services stop    Stop all services"
      echo "    roots-services status  Show service status"
      echo ""
    '';
  };
}
