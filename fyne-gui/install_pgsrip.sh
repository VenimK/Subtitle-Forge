#!/bin/bash

echo "Installing pgsrip..."

# Find Python installation
PYTHON_CMD=""

# Check for Python in common locations
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
elif command -v python3 &> /dev/null; then
    # Check if this is the Apple system Python (doesn't support pip packages)
    FOUND_PYTHON=$(command -v python3)
    if [[ "$FOUND_PYTHON" != *"/Library/Developer/CommandLineTools"* ]]; then
        PYTHON_CMD="python3"
        echo "Found Python 3 in PATH"
    else
        echo "Skipping Apple system Python (doesn't support pip packages)"
    fi
elif command -v python &> /dev/null; then
    FOUND_PYTHON=$(command -v python)
    if [[ "$FOUND_PYTHON" != *"/Library/Developer/CommandLineTools"* ]]; then
        PYTHON_CMD="python"
        echo "Found Python in PATH"
    else
        echo "Skipping Apple system Python (doesn't support pip packages)"
    fi
fi

# Install with pip if Python is found
if [ -n "$PYTHON_CMD" ]; then
    echo "Attempting to install pgsrip with $PYTHON_CMD..."
    $PYTHON_CMD -m pip install --break-system-packages --ignore-installed pgsrip
    if [ $? -eq 0 ]; then
        echo "pgsrip successfully installed with $PYTHON_CMD"
        exit 0
    else
        echo "pip installation failed with $PYTHON_CMD"
    fi
else
    echo "❌ No suitable Python installation found"
    echo ""
    echo "Usage:"
    echo "/Library/Developer/CommandLineTools/usr/bin/python3 -m pip install [options]"
    echo "  <requirement specifier> [package-index-options] ..."
    echo "/Library/Developer/CommandLineTools/usr/bin/python3 -m pip install [options] -r"
    echo "  <requirements file> [package-index-options] ..."
    echo "/Library/Developer/CommandLineTools/usr/bin/python3 -m pip install [options]"
    echo "  [-e] <vcs project url> ..."
    echo "/Library/Developer/CommandLineT..."
    echo "  (output truncated)"
    echo ""
    echo "Suggestions:"
    echo "- Make sure Homebrew is properly installed"
    echo "- Try running 'brew doctor' to diagnose Homebrew issues"
    echo "- Try installing manually: brew install pgsrip"
    exit 1
fi


# If we get here, all methods failed
echo "❌ Failed to install pgsrip"
echo ""
echo "Suggestions:"
echo "- Make sure Homebrew is properly installed"
echo "- Try running 'brew doctor' to diagnose Homebrew issues"
echo "- Try installing manually: brew install pgsrip"
exit 1
