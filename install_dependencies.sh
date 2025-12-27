#!/usr/bin/env bash
set -euo pipefail

# Dry-run support: set DRY_RUN=1 to print commands without executing them
DRY_RUN=${DRY_RUN:-0}
if [ "$DRY_RUN" = "1" ]; then
  shopt -s expand_aliases
  alias sudo='echo sudo'
  alias brew='echo brew'
  alias apt='echo apt'
  alias dnf='echo dnf'
  alias pacman='echo pacman'
  alias zypper='echo zypper'
  alias curl='echo curl'
  alias wget='echo wget'
  alias unzip='echo unzip'
  alias cmake='echo cmake'
  alias make='echo make'
  alias python3='echo python3'
  alias python='echo python'
  alias pip='echo pip'
  alias mkdir='echo mkdir'
  alias rm='echo rm'
  alias ln='echo ln'
  alias export='echo export'
  alias sh='echo sh'
  alias bash='echo bash'
fi

# Helper to echo and (optionally) execute complex commands (including pipes)
run_cmd() { echo "[CMD] $*"; if [ "$DRY_RUN" != "1" ]; then eval "$@"; fi }

# Subtitle Forge unified dependency installer (macOS + Linux)
# - Installs: mkvtoolnix (mkvmerge/mkvextract/mkvinfo), ffmpeg, deno, tesseract, python3+pip, git, curl/wget, unzip, cmake, make
# - OCR path: pgsrip via pip (chosen by user)
# - Tessdata: sets up ~/tessdata_best with common languages
# - Report: writes an installation report with versions/paths

REPORT_FILE="$HOME/subtitle_forge_install_report.txt"
TESSDATA_DIR="$HOME/tessdata_best"
GST_VENV_DIR="$HOME/.subtitle-forge/gst-venv"
COMMON_LANGS=(eng fra deu spa ita nld)

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
err()  { echo "[ERROR] $*" >&2; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    return 1
  fi
}

is_macos() { [[ "$(uname -s)" == "Darwin" ]]; }

# Detect Linux package manager
linux_pkg_mgr() {
  if command -v apt >/dev/null 2>&1; then echo apt; return; fi
  if command -v dnf >/dev/null 2>&1; then echo dnf; return; fi
  if command -v pacman >/dev/null 2>&1; then echo pacman; return; fi
  if command -v zypper >/dev/null 2>&1; then echo zypper; return; fi
  echo none
}

install_macos() {
  if ! require_cmd brew; then
    warn "Homebrew not found. Installing Homebrew..."
    run_cmd "/bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    # Ensure brew in PATH for Apple Silicon/Intel
    if [ -d "/opt/homebrew/bin" ]; then export PATH="/opt/homebrew/bin:$PATH"; fi
    if [ -d "/usr/local/bin" ]; then export PATH="/usr/local/bin:$PATH"; fi
  fi

  log "Installing packages via brew..."
  brew update || true
  brew install mkvtoolnix ffmpeg deno tesseract git cmake unzip || true
  # curl/wget may already exist
  brew install curl wget || true
  # Python
  brew install python@3 || true
}

install_linux() {
  local mgr; mgr=$(linux_pkg_mgr)
  case "$mgr" in
    apt)
      log "Using apt"
      sudo apt update
      sudo apt install -y mkvtoolnix ffmpeg tesseract-ocr git curl wget unzip cmake make python3 python3-pip
      # Deno via official script (fastest, cross-distro)
      if ! require_cmd deno; then run_cmd "curl -fsSL https://deno.land/install.sh | sh"; fi
      ;;
    dnf)
      log "Using dnf"
      sudo dnf install -y mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python3 python3-pip
      if ! require_cmd deno; then run_cmd "curl -fsSL https://deno.land/install.sh | sh"; fi
      ;;
    pacman)
      log "Using pacman"
      sudo pacman -Sy --noconfirm
      sudo pacman -S --noconfirm mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python python-pip
      if ! require_cmd deno; then run_cmd "curl -fsSL https://deno.land/install.sh | sh"; fi
      ;;
    zypper)
      log "Using zypper"
      sudo zypper refresh
      sudo zypper install -y mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python3 python3-pip
      if ! require_cmd deno; then run_cmd "curl -fsSL https://deno.land/install.sh | sh"; fi
      ;;
    *)
      err "Unsupported Linux package manager. Please install dependencies manually."
      exit 1
      ;;
  esac

  # Add deno to PATH if installed to ~/.deno/bin
  if [ -d "$HOME/.deno/bin" ]; then export PATH="$HOME/.deno/bin:$PATH"; fi
}

install_pgsrip() {
  log "Installing pgsrip (Python package for PGS OCR)..."
  local py=python3
  if ! require_cmd python3 && require_cmd python; then py=python; fi
  if ! require_cmd "$py"; then
    err "Python not found after installation step."
    return 1
  fi
  run_cmd "$py -m pip install --user --upgrade pip || true"
  run_cmd "$py -m pip install --user --break-system-packages --ignore-installed pgsrip || $py -m pip install --user pgsrip"
}

install_gst() {
  log "Installing GST (gemini-srt-translator) into: $GST_VENV_DIR"

  local py=python3
  if ! require_cmd python3 && require_cmd python; then py=python; fi
  if ! require_cmd "$py"; then
    err "Python not found after installation step."
    return 1
  fi

  log "GST debug: base python command: $py"
  (command -v "$py") 2>/dev/null || true
  ("$py" --version) 2>/dev/null || true
  ("$py" -m pip -V) 2>/dev/null || true

  run_cmd "mkdir -p \"$(dirname \"$GST_VENV_DIR\")\""

  # Create venv if missing (or if python is missing inside it)
  if [ ! -x "$GST_VENV_DIR/bin/python3" ] && [ ! -x "$GST_VENV_DIR/bin/python" ]; then
    run_cmd "$py -m venv \"$GST_VENV_DIR\""
  fi

  local vpy="$GST_VENV_DIR/bin/python3"
  if [ ! -x "$vpy" ]; then vpy="$GST_VENV_DIR/bin/python"; fi
  if [ ! -x "$vpy" ]; then
    err "Failed to create GST virtual environment (python not found in venv)."
    return 1
  fi

  log "GST debug: venv python: $vpy"
  ("$vpy" --version) 2>/dev/null || true
  ("$vpy" -m pip -V) 2>/dev/null || true

  # Upgrade pip inside venv and install/upgrade GST
  run_cmd "$vpy -m pip install --upgrade pip"
  run_cmd "$vpy -m pip install --upgrade gemini-srt-translator"

  log "GST debug: pip show gemini-srt-translator (venv)"
  ("$vpy" -m pip show gemini-srt-translator) 2>/dev/null || true
  log "GST debug: listing venv bin directory (top entries):"
  (ls -la "$GST_VENV_DIR/bin" | head -n 60) 2>/dev/null || true

  local gst_bin=""
  if [ -x "$GST_VENV_DIR/bin/gst" ]; then
    gst_bin="$GST_VENV_DIR/bin/gst"
  elif [ -f "$GST_VENV_DIR/bin/gst" ]; then
    # Sometimes the file exists but isn't marked executable
    run_cmd "chmod +x \"$GST_VENV_DIR/bin/gst\" || true"
    if [ -x "$GST_VENV_DIR/bin/gst" ]; then
      gst_bin="$GST_VENV_DIR/bin/gst"
    fi
  fi

  if [ -z "$gst_bin" ]; then
    # Try to resolve via PATH when venv bin is prefixed
    local resolved
    resolved=$(PATH="$GST_VENV_DIR/bin:$PATH" command -v gst 2>/dev/null || true)
    log "GST debug: command -v gst with venv PATH => ${resolved:-<empty>}"
    if [ -n "$resolved" ] && [ -x "$resolved" ]; then
      gst_bin="$resolved"
    fi
  fi

  if [ -n "$gst_bin" ]; then
    log "GST installed at: $gst_bin"
    ("$gst_bin" --version | head -n1) 2>/dev/null || true
    log "GST usage: run '$gst_bin translate ...' (works even if 'gst' is not on your shell PATH)"
    log "GST usage: to use 'gst' directly, either run 'source \"$GST_VENV_DIR/bin/activate\"' or add '$GST_VENV_DIR/bin' to your PATH"
  else
    warn "GST install completed but gst binary not found in venv bin: $GST_VENV_DIR/bin"
    warn "Debug: base python: $py"
    ("$py" --version) 2>/dev/null || true
    ("$py" -m pip -V) 2>/dev/null || true

    warn "Debug: venv python: $vpy"
    ("$vpy" --version) 2>/dev/null || true
    ("$vpy" -m pip -V) 2>/dev/null || true

    warn "Debug: pip show gemini-srt-translator (venv):"
    ("$vpy" -m pip show gemini-srt-translator) 2>/dev/null || true

    warn "Debug: pip list (filtered for gst/gemini) (venv):"
    ("$vpy" -m pip list | grep -iE "gemini|gst" || true) 2>/dev/null || true

    warn "Debug: venv bin directory listing (top entries):"
    (ls -la "$GST_VENV_DIR/bin" | head -n 120) 2>/dev/null || true

    warn "Debug: resolving gst via PATH with venv bin prefixed:"
    (PATH="$GST_VENV_DIR/bin:$PATH" command -v gst && PATH="$GST_VENV_DIR/bin:$PATH" gst --version | head -n 1) 2>/dev/null || true
    return 1
  fi
}

setup_tessdata() {
  log "Setting up tessdata at $TESSDATA_DIR"
  mkdir -p "$TESSDATA_DIR"
  # Download common language traineddata if missing
  for lang in "${COMMON_LANGS[@]}"; do
    if [ ! -f "$TESSDATA_DIR/$lang.traineddata" ]; then
      log "Downloading $lang.traineddata"
      curl -fsSL "https://github.com/tesseract-ocr/tessdata_best/raw/main/$lang.traineddata" -o "$TESSDATA_DIR/$lang.traineddata" || \
      wget -qO "$TESSDATA_DIR/$lang.traineddata" "https://github.com/tesseract-ocr/tessdata_best/raw/main/$lang.traineddata"
    fi
  done
}

write_report() {
  log "Writing installation report to $REPORT_FILE"
  {
    echo "=== Subtitle Forge Dependency Report ==="
    date
    echo "OS: $(uname -a)"
    echo "PATH: $PATH"
    echo "TESSDATA_PREFIX: $TESSDATA_DIR"
    echo "GST_VENV_DIR: $GST_VENV_DIR"
    echo
    echo "-- Versions --"
    (ffmpeg -version | head -n1) 2>/dev/null || true
    (mkvmerge --version | head -n1) 2>/dev/null || true
    (mkvextract --version | head -n1) 2>/dev/null || true
    (mkvinfo --version | head -n1) 2>/dev/null || true
    (deno --version) 2>/dev/null || true
    (tesseract --version | head -n1) 2>/dev/null || true
    (python3 --version || python --version) 2>/dev/null || true
    (pgsrip --version || echo "pgsrip: try 'python3 -m pgsrip --version'") 2>/dev/null || true
    ([ -x "$GST_VENV_DIR/bin/gst" ] && "$GST_VENV_DIR/bin/gst" --version | head -n1) 2>/dev/null || true
    echo
    echo "-- Locations --"
    command -v ffmpeg || true
    command -v mkvmerge || true
    command -v mkvextract || true
    command -v mkvinfo || true
    command -v deno || true
    command -v tesseract || true
    command -v python3 || command -v python || true
    command -v pgsrip || true
    [ -x "$GST_VENV_DIR/bin/gst" ] && echo "$GST_VENV_DIR/bin/gst" || true
  } > "$REPORT_FILE" || warn "Could not write report"
}

main() {
  if is_macos; then
    log "Detected macOS"
    install_macos
  else
    log "Detected Linux"
    install_linux
  fi

  setup_tessdata
  install_pgsrip || warn "pgsrip installation reported an issue"


  install_gst || warn "GST installation reported an issue"

  # Export for current shell; app also sets this internally
  export TESSDATA_PREFIX="$TESSDATA_DIR"

  write_report
  log "Done. Please review $REPORT_FILE if anything failed."
}

main "$@"
