#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="${APP_DIR:-/home/samuel/.local/share/xbvr-pmv-wsl}"
BIN="${BIN:-$APP_DIR/xbvr-server}"
WEB_PORT="${WEB_PORT:-19999}"
WS_PORT="${WS_PORT:-19998}"
LOG_FILE="${LOG_FILE:-/tmp/xbvr-${WEB_PORT}.log}"

NO_BUILD=0
BUILD_UI=0
FOREGROUND=0

usage() {
  cat <<EOF
Usage: ./startup.sh [options]

Options:
  --no-build     Skip go build
  --build-ui     Build UI bundle before go build (corepack yarn build)
  --foreground   Run in foreground (no nohup/background)
  -h, --help     Show this help

Env overrides:
  APP_DIR, BIN, WEB_PORT, WS_PORT, LOG_FILE
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-build)
      NO_BUILD=1
      ;;
    --build-ui)
      BUILD_UI=1
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

mkdir -p "$APP_DIR"
cd "$SCRIPT_DIR"

if [[ "$BUILD_UI" -eq 1 ]]; then
  echo "[startup] Building UI bundle..."
  corepack yarn build
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
