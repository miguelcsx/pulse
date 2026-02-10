# shellcheck shell=bash

function kill_port {
  local pids
  local port="${1}"

  pids="$(mktemp)"

  if ! lsof -t "-i:${port}" > "${pids}" 2>/dev/null; then
    echo "[INFO] No process was found listening on port ${port}"
    return 0
  fi

  while read -r pid; do
    if kill -9 "${pid}" 2>/dev/null; then
      if timeout 5 tail --pid="${pid}" -f /dev/null 2>/dev/null; then
        echo "[INFO] Process listening on port ${port} with PID ${pid} was successfully killed"
      else
        echo "[WARNING] Timeout while attempting to kill process with PID ${pid} listening on port ${port}"
      fi
    else
      echo "[ERROR] Unable to kill process with PID ${pid} listening on port ${port}"
      return 1
    fi
  done < "${pids}"
}

function wait_port {
  local elapsed=1
  local max_timeout=$(("${1}"))
  local host="${2%:*}"
  local port="${2#*:}"

  while true; do
    if timeout 1s nc -z "${host}" "${port}" 2>/dev/null; then
      echo "[INFO] ${host}:${port} is now open"
      return 0
    elif [ "${elapsed}" -gt "${max_timeout}" ]; then
      echo "[ERROR] Timeout while waiting for ${host}:${port} to open"
      return 1
    else
      echo "[INFO] Waiting 1 second for ${host}:${port} to open: ${elapsed} seconds in total"
      sleep 1
      ((elapsed++))
    fi
  done
}

"${@}"
