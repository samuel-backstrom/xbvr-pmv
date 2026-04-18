#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="${APP_NAME:-xbvr-pmv}"

default_app_dir() {
  if [[ -n "${APP_DIR:-}" ]]; then
    printf '%s\n' "$APP_DIR"
    return
  fi

  case "$(uname -s)" in
    Darwin)
      printf '%s\n' "$HOME/Library/Application Support/$APP_NAME"
      ;;
    Linux)
      if [[ -n "${WSL_DISTRO_NAME:-}" || -n "${WSL_INTEROP:-}" ]]; then
        printf '%s\n' "$HOME/.local/share/${APP_NAME}-wsl"
      else
        printf '%s\n' "${XDG_DATA_HOME:-$HOME/.local/share}/$APP_NAME"
      fi
      ;;
    *)
      printf '%s\n' "${XDG_DATA_HOME:-$HOME/.local/share}/$APP_NAME"
      ;;
  esac
}

APP_DIR="${APP_DIR:-$(default_app_dir)}"
BIN="${BIN:-$APP_DIR/xbvr-server}"
PID_FILE="${PID_FILE:-$APP_DIR/xbvr-server.pid}"
WEB_PORT="${WEB_PORT:-9999}"
WS_PORT="${WS_PORT:-19998}"
GO_BUILD_TAGS="${GO_BUILD_TAGS:-json1}"

NO_BUILD=0
BUILD_UI=1
UI_VARIANT="new"
FOREGROUND=0

usage() {
  cat <<EOF
Usage: ./startup [options]

Options:
  --no-build     Skip go build
  --no-build-ui  Skip UI bundle rebuild before go build
  --build-ui     Force UI bundle rebuild before go build (default)
  --old-ui       Build and launch the legacy UI on port 19999 instead
  --foreground   Run in foreground (no nohup/background)
  -h, --help     Show this help

Env overrides:
  APP_DIR, BIN, WEB_PORT, WS_PORT, LOG_FILE, UI_VARIANT
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-build)
      NO_BUILD=1
      ;;
    --no-build-ui)
      BUILD_UI=0
      ;;
    --build-ui)
      BUILD_UI=1
      ;;
    --old-ui)
      UI_VARIANT="old"
      WEB_PORT=19999
      ;;
    --foreground)
      FOREGROUND=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

if [[ "$UI_VARIANT" == "old" ]]; then
  WEB_PORT=19999
fi

LOG_FILE="${LOG_FILE:-/tmp/xbvr-${WEB_PORT}.log}"

mkdir -p "$APP_DIR"
cd "$SCRIPT_DIR"

run_yarn() {
  if command -v corepack >/dev/null 2>&1; then
    corepack yarn "$@"
    return
  fi

  if command -v yarn >/dev/null 2>&1; then
    yarn "$@"
    return
  fi

  if command -v npx >/dev/null 2>&1; then
    npx --yes yarn@1.22.22 "$@"
    return
  fi

  echo "[startup] Error: neither corepack nor yarn is available on PATH" >&2
  exit 1
}

stop_existing_server() {
  if [[ -f "$PID_FILE" ]]; then
    local pid
    pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "[startup] Stopping existing server PID=$pid from $PID_FILE..."
      kill "$pid" 2>/dev/null || true
      sleep 1
    fi
    rm -f "$PID_FILE"
  fi

  local needle="$BIN -app_dir $APP_DIR -web_port $WEB_PORT -ws_addr 0.0.0.0:$WS_PORT"
  while IFS= read -r pid command; do
    if [[ "$command" == *"$needle"* ]]; then
      echo "[startup] Stopping existing server PID=$pid (matched command line)..."
      kill "$pid" 2>/dev/null || true
    fi
  done < <(ps -ax -o pid= -o command=)
  sleep 1
}

if [[ "$BUILD_UI" -eq 1 ]]; then
  if [[ "$UI_VARIANT" == "old" ]]; then
    if [[ ! -x "$SCRIPT_DIR/node_modules/.bin/vue-cli-service" ]]; then
      echo "[startup] Installing legacy UI dependencies..."
      run_yarn install --frozen-lockfile
    fi
    echo "[startup] Building legacy UI bundle..."
    run_yarn build:old-ui
  else
    if [[ ! -x "$SCRIPT_DIR/ui-new/node_modules/.bin/vite" || ! -x "$SCRIPT_DIR/ui-new/node_modules/.bin/tsc" ]]; then
      echo "[startup] Installing ui-new dependencies..."
      npm ci --prefix ui-new
    fi
    echo "[startup] Building UI bundle..."
    run_yarn build
  fi
fi

if [[ "$NO_BUILD" -eq 0 || ! -x "$BIN" ]]; then
  echo "[startup] Building server binary..."
  if [[ -n "$GO_BUILD_TAGS" ]]; then
    CGO_ENABLED=1 go build -tags "$GO_BUILD_TAGS" -o "$BIN" .
  else
    CGO_ENABLED=1 go build -o "$BIN" .
  fi
fi

echo "[startup] Checking for an existing server for app_dir=$APP_DIR..."
stop_existing_server

CMD=("$BIN" "-app_dir" "$APP_DIR" "-web_port" "$WEB_PORT" "-ws_addr" "0.0.0.0:$WS_PORT")

if [[ "$FOREGROUND" -eq 1 ]]; then
  echo "[startup] Starting in foreground..."
  echo "[startup] URL: http://127.0.0.1:${WEB_PORT}/"
  exec "${CMD[@]}"
fi

echo "[startup] Starting in background..."
nohup "${CMD[@]}" >"$LOG_FILE" 2>&1 &
PID=$!
printf '%s\n' "$PID" > "$PID_FILE"
sleep 1

if ! kill -0 "$PID" 2>/dev/null; then
  echo "[startup] Failed to start. Last log lines:"
  tail -n 80 "$LOG_FILE" || true
  rm -f "$PID_FILE"
  exit 1
fi

echo "[startup] Started PID=$PID"
echo "[startup] URL: http://127.0.0.1:${WEB_PORT}/"
echo "[startup] Log: $LOG_FILE"
