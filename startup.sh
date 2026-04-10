#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="${APP_DIR:-/home/samuel/.local/share/xbvr-pmv-wsl}"
BIN="${BIN:-$APP_DIR/xbvr-server}"
WEB_PORT="${WEB_PORT:-9999}"
WS_PORT="${WS_PORT:-19998}"

NO_BUILD=0
BUILD_UI=1
UI_VARIANT="new"
FOREGROUND=0

usage() {
  cat <<EOF
Usage: ./startup.sh [options]

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

if [[ "$BUILD_UI" -eq 1 ]]; then
  if [[ "$UI_VARIANT" == "old" ]]; then
    if [[ ! -x "$SCRIPT_DIR/node_modules/.bin/vue-cli-service" ]]; then
      echo "[startup] Installing legacy UI dependencies..."
      corepack yarn install --frozen-lockfile
    fi
    echo "[startup] Building legacy UI bundle..."
    corepack yarn build:old-ui
  else
    if [[ ! -x "$SCRIPT_DIR/ui-new/node_modules/.bin/vite" || ! -x "$SCRIPT_DIR/ui-new/node_modules/.bin/tsc" ]]; then
      echo "[startup] Installing ui-new dependencies..."
      npm ci --prefix ui-new
    fi
    echo "[startup] Building UI bundle..."
    corepack yarn build
  fi
fi

if [[ "$NO_BUILD" -eq 0 || ! -x "$BIN" ]]; then
  echo "[startup] Building server binary..."
  go build -o "$BIN" .
fi

echo "[startup] Stopping existing server for app_dir=$APP_DIR (if running)..."
pkill -f "$BIN -app_dir $APP_DIR" 2>/dev/null || true
sleep 1

CMD=("$BIN" "-app_dir" "$APP_DIR" "-web_port" "$WEB_PORT" "-ws_addr" "0.0.0.0:$WS_PORT")

if [[ "$FOREGROUND" -eq 1 ]]; then
  echo "[startup] Starting in foreground..."
  echo "[startup] URL: http://127.0.0.1:${WEB_PORT}/"
  exec "${CMD[@]}"
fi

echo "[startup] Starting in background..."
nohup "${CMD[@]}" >"$LOG_FILE" 2>&1 &
PID=$!
sleep 1

if ! kill -0 "$PID" 2>/dev/null; then
  echo "[startup] Failed to start. Last log lines:"
  tail -n 80 "$LOG_FILE" || true
  exit 1
fi

echo "[startup] Started PID=$PID"
echo "[startup] URL: http://127.0.0.1:${WEB_PORT}/"
echo "[startup] Log: $LOG_FILE"
