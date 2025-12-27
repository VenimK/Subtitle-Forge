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
  alias rm='echo rm'
  alias python3='echo python3'
  alias python='echo python'
fi

run_cmd() { echo "[CMD] $*"; if [ "$DRY_RUN" != "1" ]; then eval "$@"; fi }

REPORT_FILE="$HOME/subtitle_forge_install_report.txt"
TESSDATA_DIR="$HOME/tessdata_best"
GST_VENV_DIR="$HOME/.subtitle-forge/gst-venv"

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
err()  { echo "[ERROR] $*" >&2; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1
}

is_macos() { [[ "$(uname -s)" == "Darwin" ]]; }

linux_pkg_mgr() {
  if command -v apt >/dev/null 2>&1; then echo apt; return; fi
  if command -v dnf >/dev/null 2>&1; then echo dnf; return; fi
  if command -v pacman >/dev/null 2>&1; then echo pacman; return; fi
  if command -v zypper >/dev/null 2>&1; then echo zypper; return; fi
  echo none
}

usage() {
  cat <<'EOF'
Subtitle Forge dependency uninstaller (macOS + Linux)

This script can remove the dependencies installed by install_dependencies.sh.

SAFE DEFAULT:
- Removes only Subtitle Forge user-scoped items:
  - ~/.subtitle-forge/gst-venv
  - ~/tessdata_best
  - ~/subtitle_forge_install_report.txt

OPTIONAL (more destructive):
  --remove-pgsrip      Uninstall pgsrip from your Python user site-packages
  --remove-brew        Uninstall Homebrew packages (macOS)
  --remove-system      Uninstall system packages (Linux package manager)

Other:
  --yes                Skip interactive confirmation

Environment:
  DRY_RUN=1            Print commands without executing
EOF
}

confirm() {
  local msg="$1"
  if [ "${ASSUME_YES:-0}" = "1" ]; then
    return 0
  fi
  echo
  echo "$msg"
  read -r -p "Type 'YES' to continue: " answer
  [[ "$answer" == "YES" ]]
}

remove_user_items() {
  log "Removing Subtitle Forge user-scoped items (safe)"

  if [ -d "$GST_VENV_DIR" ]; then
    run_cmd "rm -rf \"$GST_VENV_DIR\""
  else
    log "GST venv not found: $GST_VENV_DIR"
  fi

  if [ -d "$TESSDATA_DIR" ]; then
    run_cmd "rm -rf \"$TESSDATA_DIR\""
  else
    log "Tessdata dir not found: $TESSDATA_DIR"
  fi

  if [ -f "$REPORT_FILE" ]; then
    run_cmd "rm -f \"$REPORT_FILE\""
  else
    log "Install report not found: $REPORT_FILE"
  fi
}

remove_pgsrip() {
  log "Uninstalling pgsrip (Python user install)"

  local py=python3
  if ! require_cmd python3 && require_cmd python; then py=python; fi
  if ! require_cmd "$py"; then
    warn "Python not found; cannot uninstall pgsrip"
    return 0
  fi

  run_cmd "$py -m pip uninstall -y pgsrip || true"
}

remove_brew_pkgs() {
  if ! require_cmd brew; then
    warn "Homebrew not found; skipping brew removals"
    return 0
  fi

  log "Uninstalling Homebrew packages installed by Subtitle Forge"
  run_cmd "brew uninstall --ignore-dependencies mkvtoolnix ffmpeg deno tesseract git cmake unzip curl wget python@3 || true"
  run_cmd "brew autoremove || true"
}

remove_linux_pkgs() {
  local mgr; mgr=$(linux_pkg_mgr)
  case "$mgr" in
    apt)
      run_cmd "sudo apt remove -y mkvtoolnix ffmpeg tesseract-ocr git curl wget unzip cmake make python3 python3-pip || true"
      run_cmd "sudo apt autoremove -y || true"
      ;;
    dnf)
      run_cmd "sudo dnf remove -y mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python3 python3-pip || true"
      ;;
    pacman)
      run_cmd "sudo pacman -Rns --noconfirm mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python python-pip || true"
      ;;
    zypper)
      run_cmd "sudo zypper remove -y mkvtoolnix ffmpeg tesseract git curl wget unzip cmake make python3 python3-pip || true"
      ;;
    *)
      warn "Unsupported Linux package manager; skipping system package removal"
      ;;
  esac

  # Deno installed by official script lives in ~/.deno by default
  if [ -d "$HOME/.deno" ]; then
    run_cmd "rm -rf \"$HOME/.deno\""
  fi
}

main() {
  local remove_pgsrip_flag=0
  local remove_brew_flag=0
  local remove_system_flag=0

  while [ $# -gt 0 ]; do
    case "$1" in
      --remove-pgsrip)
        remove_pgsrip_flag=1
        ;;
      --remove-brew)
        remove_brew_flag=1
        ;;
      --remove-system)
        remove_system_flag=1
        ;;
      --yes)
        ASSUME_YES=1
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        err "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
    shift
  done

  if ! confirm "This will uninstall Subtitle Forge dependencies. Safe items will be removed by default."; then
    log "Cancelled"
    exit 0
  fi

  remove_user_items

  if [ "$remove_pgsrip_flag" = "1" ]; then
    if confirm "Uninstall pgsrip from Python user site-packages?"; then
      remove_pgsrip
    else
      log "Skipped pgsrip uninstall"
    fi
  fi

  if is_macos; then
    if [ "$remove_brew_flag" = "1" ]; then
      if confirm "Uninstall Homebrew packages (may affect other apps)?"; then
        remove_brew_pkgs
      else
        log "Skipped Homebrew uninstall"
      fi
    fi
  else
    if [ "$remove_system_flag" = "1" ]; then
      if confirm "Uninstall Linux system packages (may affect other apps)?"; then
        remove_linux_pkgs
      else
        log "Skipped system package uninstall"
      fi
    fi
  fi

  log "Done."
}

main "$@"
