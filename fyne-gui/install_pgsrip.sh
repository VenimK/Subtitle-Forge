#!/bin/bash
set -e

echo "Installing pgsrip..."

BASE_DIR="$HOME/.subtitle-forge"
VENV_DIR="$BASE_DIR/pgsrip-venv"
PIP_BIN="$VENV_DIR/bin/pip"
PGSRIP_BIN="$VENV_DIR/bin/pgsrip"

PYTHON_CMD=""

if [ -f "/opt/homebrew/bin/python3" ]; then
    PYTHON_CMD="/opt/homebrew/bin/python3"
    echo "Found Python 3 at /opt/homebrew/bin/python3"
elif [ -f "/opt/homebrew/bin/python" ]; then
    PYTHON_CMD="/opt/homebrew/bin/python"
    echo "Found Python at /opt/homebrew/bin/python"
elif [ -f "/usr/local/bin/python3" ]; then
    PYTHON_CMD="/usr/local/bin/python3"
    echo "Found Python 3 at /usr/local/bin/python3"
elif [ -f "/usr/local/bin/python" ]; then
    PYTHON_CMD="/usr/local/bin/python"
    echo "Found Python at /usr/local/bin/python"
elif command -v python3 >/dev/null 2>&1; then
    FOUND_PYTHON=$(command -v python3)
    if [[ "$FOUND_PYTHON" != *"/Library/Developer/CommandLineTools"* ]]; then
        PYTHON_CMD="$FOUND_PYTHON"
        echo "Found Python 3 in PATH"
    else
        echo "Skipping Apple system Python (doesn't support pip packages)"
    fi
elif command -v python >/dev/null 2>&1; then
    FOUND_PYTHON=$(command -v python)
    if [[ "$FOUND_PYTHON" != *"/Library/Developer/CommandLineTools"* ]]; then
        PYTHON_CMD="$FOUND_PYTHON"
        echo "Found Python in PATH"
    else
        echo "Skipping Apple system Python (doesn't support pip packages)"
    fi
fi

if [ -z "$PYTHON_CMD" ]; then
    echo "❌ No suitable Python installation found"
    echo ""
    echo "Suggestions:"
    echo "- Install Python 3 first"
    echo "- Try installing manually in a virtual environment"
    exit 1
fi

mkdir -p "$BASE_DIR"

echo "Creating dedicated pgsrip virtual environment at $VENV_DIR"
"$PYTHON_CMD" -m venv "$VENV_DIR"

"$VENV_DIR/bin/python" -m pip install --upgrade pip setuptools wheel
"$PIP_BIN" install --upgrade pgsrip

if [ ! -x "$PGSRIP_BIN" ]; then
    echo "❌ pgsrip install completed but executable was not found at $PGSRIP_BIN"
    exit 1
fi

echo "pgsrip successfully installed at $PGSRIP_BIN"
echo "You can test it with: $PGSRIP_BIN --help"
