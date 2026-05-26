{ inputs', pkgs, projectPath }:
let
  secrets_dev = projectPath "/secrets/dev.yaml";
  secrets_prod = projectPath "/secrets/prod.yaml";
in pkgs.writeShellApplication {
  bashOptions = [ ];
  name = "stone-envars";
  runtimeInputs = [
    inputs'.roots.packages.roots-sops
  ];
  text = ''
    main() {
      local secrets=(
        DATABASE_URL
        NEO4J_URI
        NEO4J_USER
        NEO4J_PASSWORD
        REDIS_URL
        JWT_SECRET
        ENV
        CORS_ORIGINS
        WS_ORIGINS
      )

      local secrets_file

      case "''${1:-}" in
        dev)
          secrets_file="${secrets_dev}"
          ;;
        prod)
          secrets_file="${secrets_prod}"
          ;;
        *)
          echo "[ERROR] Usage: stone-envars <dev|prod>"
          return 1
          ;;
      esac

      # shellcheck disable=SC1091
      source roots-sops sops_export_vars "''${secrets_file}" "''${secrets[@]}"
    }

    main "$@"
  '';
}
