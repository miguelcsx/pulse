# shellcheck shell=bash

function sops_export_vars {
  local manifest="${1}"
  local json

  if [ ! -f "${manifest}" ]; then
    echo "[ERROR] Secrets file not found: ${manifest}"
    return 1
  fi

  echo "[INFO] Decrypting ${manifest}"
  json="$(sops --decrypt --output-type json "${manifest}")"
  for var in "${@:2}"; do
    local value
    value="$(echo "${json}" | jq -erc ".${var}")"
    if [ $? -eq 0 ]; then
      export "${var}=${value}"
      echo "[INFO] Exported: ${var}"
    else
      echo "[WARNING] Key not found in secrets: ${var}"
    fi
  done
}

"${@}"
