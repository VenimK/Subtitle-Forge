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
    PYTHON_CMD="python3"
    echo "Found Python 3 in PATH"
elif command -v python &> /dev/null; then
    PYTHON_CMD="python"
    echo "Found Python in PATH"
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
    echo "No Python installation found"
fi


# If we get here, all methods failed
echo "Failed to install pgsrip. Please install it manually using:"
echo "$PYTHON_CMD -m pip install --break-system-packages --ignore-installed pgsrip"
exit 1
