#!/usr/bin/env bash
set -e

echo "[INFO] Starting Subtitle Forge / GST installer..."

# 1️⃣ Detect macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "[ERROR] This installer is only for macOS."
    exit 1
fi
echo "[INFO] Detected macOS"

# 2️⃣ Homebrew dependencies
BREW_DEPS=(mkvtoolnix ffmpeg deno tesseract git cmake unzip curl wget python@3.14)
echo "[INFO] Checking Homebrew dependencies..."
for dep in "${BREW_DEPS[@]}"; do
    if ! brew list "$dep" &>/dev/null; then
        echo "[INFO] Installing $dep..."
        brew install "$dep"
    else
        echo "[INFO] $dep already installed"
    fi
done

# 3️⃣ Set up directories
BASE_DIR="$HOME/.subtitle-forge"
VENV_DIR="$BASE_DIR/gst-venv"
TESSDATA_DIR="$HOME/tessdata_best"

mkdir -p "$BASE_DIR"
mkdir -p "$TESSDATA_DIR"

echo "[INFO] Tesseract data directory: $TESSDATA_DIR"

# 4️⃣ Download common Tesseract languages
LANGUAGES=(eng fra spa deu nld ita)
echo "[INFO] Downloading Tesseract languages: ${LANGUAGES[*]}"
for lang in "${LANGUAGES[@]}"; do
    if [ ! -f "$TESSDATA_DIR/${lang}.traineddata" ]; then
        curl -L -o "$TESSDATA_DIR/${lang}.traineddata" \
            "https://github.com/tesseract-ocr/tessdata_best/raw/main/${lang}.traineddata"
        echo "[INFO] Downloaded $lang"
    else
        echo "[INFO] $lang already exists"
    fi
done

# 5️⃣ Python virtual environment (Python 3.11+)
PYTHON_CMD=$(brew --prefix python@3.14)/bin/python3

if [[ ! -x "$PYTHON_CMD" ]]; then
    echo "[ERROR] Python 3.14 not found. Make sure Homebrew installed it."
    exit 1
fi

echo "[INFO] Creating virtual environment at $VENV_DIR"
"$PYTHON_CMD" -m venv "$VENV_DIR"

# 6️⃣ Upgrade pip inside venv
"$VENV_DIR/bin/python" -m pip install --upgrade pip setuptools wheel

# 7️⃣ Install GST and dependencies
"$VENV_DIR/bin/pip" install --upgrade \
    git+https://github.com/MaKTaiL/gemini-srt-translator.git \
    pgsrip

echo "[INFO] Installed gemini-srt-translator and dependencies"

# 8️⃣ Add venv bin to PATH and set TESSDATA_PREFIX
ZSHRC="$HOME/.zshrc"
if [ ! -f "$ZSHRC" ]; then
    touch "$ZSHRC"
fi

if ! grep -q 'subtitle-forge/gst-venv/bin' "$ZSHRC"; then
    echo "[INFO] Adding gst-venv to PATH in $ZSHRC"
    {
        echo ''
        echo '# Subtitle Forge GST venv'
        echo "export PATH=\"$VENV_DIR/bin:\$PATH\""
        echo "export TESSDATA_PREFIX=\"$TESSDATA_DIR\""
    } >> "$ZSHRC"
    echo "[INFO] You may need to restart your terminal or run 'source ~/.zshrc'"
else
    echo "[INFO] gst-venv already in PATH in $ZSHRC"
fi

echo "[INFO] Installation complete!"
echo "Run 'gst --help' to test."

