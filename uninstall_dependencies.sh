#!/usr/bin/env bash
set -e

echo "[INFO] Starting Subtitle Forge / GST uninstaller..."

# 1️⃣ Detect macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "[ERROR] This uninstaller is only for macOS."
    exit 1
fi
echo "[INFO] Detected macOS"

# 2️⃣ Remove Subtitle Forge virtual environment (gst-venv)
VENV_DIR="$HOME/.subtitle-forge/gst-venv"
if [[ -d "$VENV_DIR" ]]; then
    echo "[INFO] Removing virtual environment at $VENV_DIR"
    rm -rf "$VENV_DIR"
else
    echo "[INFO] Virtual environment not found at $VENV_DIR, skipping."
fi

# 3️⃣ Remove Tesseract language data
TESSDATA_DIR="$HOME/tessdata_best"
if [[ -d "$TESSDATA_DIR" ]]; then
    echo "[INFO] Removing Tesseract data directory at $TESSDATA_DIR"
    rm -rf "$TESSDATA_DIR"
else
    echo "[INFO] Tesseract data directory not found at $TESSDATA_DIR, skipping."
fi

# 4️⃣ Clean .zshrc (remove gst-venv PATH and TESSDATA_PREFIX)
ZSHRC="$HOME/.zshrc"
if [[ -f "$ZSHRC" ]]; then
    echo "[INFO] Cleaning up $ZSHRC"
    sed -i '' '/subtitle-forge\/gst-venv\/bin/d' "$ZSHRC"
    sed -i '' '/TESSDATA_PREFIX/d' "$ZSHRC"
else
    echo "[INFO] No .zshrc file found, skipping cleanup."
fi

# 5️⃣ Optionally remove Homebrew dependencies (uncomment to use)
# WARNING: This will remove dependencies that may affect other software.
echo "[INFO] Removing Homebrew dependencies installed for GST..."
BREW_DEPS=(mkvtoolnix ffmpeg deno tesseract git cmake unzip curl wget python@3.14)
for dep in "${BREW_DEPS[@]}"; do
    if brew list "$dep" &>/dev/null; then
        echo "[INFO] Uninstalling $dep..."
        brew uninstall --ignore-dependencies "$dep"
    else
        echo "[INFO] $dep not installed, skipping"
    fi
done

echo "[INFO] Uninstallation complete!"
echo "[INFO] Please restart your terminal to finalize changes."

