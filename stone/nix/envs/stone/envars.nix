{ inputs', pkgs, projectPath }:
let
  secrets_dev = projectPath "/secrets/dev.yaml";
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
        *)
          echo "[ERROR] Usage: stone-envars <dev>"
          return 1
          ;;
      esac

      # shellcheck disable=SC1091
      source roots-sops sops_export_vars "''${secrets_file}" "''${secrets[@]}"
    }

    main "$@"
  '';
}
