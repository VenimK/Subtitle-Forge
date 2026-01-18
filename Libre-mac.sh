#!/usr/bin/env bash
set -euo pipefail

PREFIX_DIR="${PREFIX_DIR:-$HOME/.libretranslate}"
REPO_DIR="${REPO_DIR:-$PREFIX_DIR/LibreTranslate}"
VENV_DIR="${VENV_DIR:-$PREFIX_DIR/venv}"
DB_DIR="${DB_DIR:-$PREFIX_DIR/db}"
PID_FILE="${PID_FILE:-$PREFIX_DIR/libretranslate.pid}"
API_KEY_FILE="${API_KEY_FILE:-$PREFIX_DIR/api_key.txt}"

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-5000}"
THREADS="${THREADS:-8}"

API_KEYS="${API_KEYS:-true}"
REQ_LIMIT="${REQ_LIMIT:-60}"

usage() {
  cat <<'EOF'
Usage:
  ./Libre-mac.sh install
  ./Libre-mac.sh list-models
  ./Libre-mac.sh list-installed
  ./Libre-mac.sh install-model <from> <to>
  ./Libre-mac.sh start
  ./Libre-mac.sh start-bg
  ./Libre-mac.sh stop
  ./Libre-mac.sh status

Environment variables:
  PREFIX_DIR   (default: ~/.libretranslate)
  HOST         (default: 127.0.0.1)
  PORT         (default: 5000)
  THREADS      (default: 8)
  API_KEYS     (default: true)
  REQ_LIMIT    (default: 60)
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

ensure_brew_deps() {
  need_cmd brew
  brew install -q python git cmake pkg-config sentencepiece || true
}

ensure_dirs() {
  mkdir -p "$PREFIX_DIR" "$DB_DIR"
}

install_libretranslate() {
  ensure_dirs

  need_cmd git
  need_cmd python3

  if command -v brew >/dev/null 2>&1; then
    ensure_brew_deps
  else
    echo "Homebrew not found. Install it first (recommended for macOS):" >&2
    echo "  https://brew.sh" >&2
    exit 1
  fi

  if [ ! -d "$REPO_DIR/.git" ]; then
    git clone https://github.com/LibreTranslate/LibreTranslate.git "$REPO_DIR"
  else
    git -C "$REPO_DIR" pull --ff-only
  fi

  python3 -m venv "$VENV_DIR"
  "$VENV_DIR/bin/pip" install --upgrade pip setuptools wheel

  "$VENV_DIR/bin/pip" install -e "$REPO_DIR"
  "$VENV_DIR/bin/pip" install argostranslate argos-translate-files translatehtml

  cat >"$PREFIX_DIR/install_argos_models.py" <<'PY'
from argostranslate import package, translate

package.update_package_index()
available = package.get_available_packages()

pairs = [
    ("en", "es"), ("es", "en"),
    ("en", "fr"), ("fr", "en"),
    ("en", "de"), ("de", "en"),
    ("en", "it"), ("it", "en"),
    ("en", "pt"), ("pt", "en"),
    ("en", "nl"), ("nl", "en"),
]

for from_code, to_code in pairs:
    pkg = next((p for p in available if p.from_code == from_code and p.to_code == to_code), None)
    if not pkg:
        print(f"[!] Model {from_code}->{to_code} not found")
        continue
    print(f"Installing {pkg.from_name} -> {pkg.to_name}...")
    try:
        download_path = pkg.download()
        package.install_from_path(download_path)
        print("  Installed successfully")
    except Exception as e:
        print(f"  [!] Failed: {e}")

print("Installed languages:", [str(lang) for lang in translate.get_installed_languages()])
PY

  "$VENV_DIR/bin/python" "$PREFIX_DIR/install_argos_models.py" || echo "[!] Some Argos models failed"

  echo "Install complete"
  echo "LibreTranslate binary: $VENV_DIR/bin/libretranslate"
}

need_venv() {
  if [ ! -x "$VENV_DIR/bin/python" ]; then
    echo "LibreTranslate not installed. Run: ./Libre-mac.sh install" >&2
    exit 1
  fi
}

list_models() {
  need_venv
  "$VENV_DIR/bin/python" - <<'PY'
from argostranslate import package

package.update_package_index()
available = package.get_available_packages()
for p in available:
    try:
        print(f"{p.from_code}->{p.to_code}\t{p.from_name} -> {p.to_name}")
    except Exception:
        print(f"{p.from_code}->{p.to_code}")
PY
}

list_installed() {
  need_venv
  "$VENV_DIR/bin/python" - <<'PY'
from argostranslate import translate

langs = translate.get_installed_languages()
if not langs:
    print("No languages installed")
else:
    for l in langs:
        try:
            print(f"{l.code}\t{l.name}")
        except Exception:
            print(str(l))
PY
}

install_model() {
  local from_code="${1:-}"
  local to_code="${2:-}"
  if [ -z "$from_code" ] || [ -z "$to_code" ]; then
    echo "Usage: ./Libre-mac.sh install-model <from> <to>" >&2
    exit 1
  fi
  need_venv
  "$VENV_DIR/bin/python" - <<PY
from argostranslate import package

from_code = ${from_code@Q}
to_code = ${to_code@Q}

package.update_package_index()
available = package.get_available_packages()

pkg = next((p for p in available if p.from_code == from_code and p.to_code == to_code), None)
if not pkg:
    raise SystemExit(f"Model {from_code}->{to_code} not found. Run: ./Libre-mac.sh list-models")

print(f"Installing {pkg.from_name} -> {pkg.to_name} ({from_code}->{to_code})")
download_path = pkg.download()
package.install_from_path(download_path)
print("Installed successfully")
PY
}

ensure_api_key() {
  if [ "$API_KEYS" != "true" ]; then
    return 0
  fi

  ensure_dirs
  need_cmd openssl

  if [ ! -f "$API_KEY_FILE" ]; then
    openssl rand -hex 16 >"$API_KEY_FILE"
    chmod 600 "$API_KEY_FILE"
  fi

  local key
  key="$(cat "$API_KEY_FILE")"

  # Initialize API keys database if it doesn't exist
  local db_file="$DB_DIR/api_keys.db"
  if [ -x "$VENV_DIR/bin/ltmanage" ]; then
    if [ ! -f "$db_file" ]; then
      echo "Initializing API keys database..."
      (
        cd "$PREFIX_DIR" && \
        LT_API_KEYS_DB_PATH="$db_file" "$VENV_DIR/bin/ltmanage" keys add 999999 --key "$key"
      ) 2>&1 || echo "Warning: Failed to initialize API key database. LibreTranslate may prompt for setup."
    else
      # Database exists, try to add/update key (ignore errors if key already exists)
      (
        cd "$PREFIX_DIR" && \
        LT_API_KEYS_DB_PATH="$db_file" "$VENV_DIR/bin/ltmanage" keys add 999999 --key "$key"
      ) 2>/dev/null || true
    fi
  else
    echo "Warning: ltmanage not found, cannot initialize API keys" >&2
  fi
}

start_libretranslate() {
  ensure_dirs
  need_cmd python3

  if [ ! -x "$VENV_DIR/bin/libretranslate" ]; then
    echo "LibreTranslate not installed. Run: ./Libre-mac.sh install" >&2
    exit 1
  fi

  ensure_api_key

  export LT_API_KEYS="$API_KEYS"
  export LT_API_KEYS_DB_PATH="$DB_DIR/api_keys.db"

  exec "$VENV_DIR/bin/libretranslate" \
    --host "$HOST" \
    --port "$PORT" \
    --threads "$THREADS" \
    --req-limit "$REQ_LIMIT" \
    --req-limit-storage "memory://" \
    $( [ "$API_KEYS" = "true" ] && echo "--api-keys" )
}

start_bg() {
  ensure_dirs

  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" >/dev/null 2>&1; then
    echo "Already running (pid $(cat "$PID_FILE"))"
    exit 0
  fi

  ensure_api_key

  export LT_API_KEYS="$API_KEYS"
  export LT_API_KEYS_DB_PATH="$DB_DIR/api_keys.db"

  nohup "$VENV_DIR/bin/libretranslate" \
    --host "$HOST" \
    --port "$PORT" \
    --threads "$THREADS" \
    --req-limit "$REQ_LIMIT" \
    --req-limit-storage "memory://" \
    $( [ "$API_KEYS" = "true" ] && echo "--api-keys" ) \
    >"$PREFIX_DIR/libretranslate.log" 2>&1 &

  echo $! >"$PID_FILE"
  echo "Started (pid $!)"
}

stop_bg() {
  if [ ! -f "$PID_FILE" ]; then
    echo "Not running (no pid file)"
    exit 0
  fi

  local pid
  pid="$(cat "$PID_FILE")"

  if kill -0 "$pid" >/dev/null 2>&1; then
    kill "$pid" || true
    sleep 1
  fi

  rm -f "$PID_FILE"
  echo "Stopped"
}

status() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" >/dev/null 2>&1; then
    echo "Running (pid $(cat "$PID_FILE"))"
    echo "URL: http://$HOST:$PORT"
    if [ -f "$API_KEY_FILE" ]; then
      echo "API key: $(cat "$API_KEY_FILE")"
    fi
  else
    echo "Not running"
  fi
}

cmd="${1:-}"
case "$cmd" in
  install) install_libretranslate ;;
  list-models) list_models ;;
  list-installed) list_installed ;;
  install-model) shift; install_model "${1:-}" "${2:-}" ;;
  start) start_libretranslate ;;
  start-bg) start_bg ;;
  stop) stop_bg ;;
  status) status ;;
  -h|--help|help|"") usage ;;
  *)
    echo "Unknown command: $cmd" >&2
    usage
    exit 1
    ;;
esac
