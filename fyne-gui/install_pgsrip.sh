#!/bin/bash

echo "Installing pgsrip..."

# Try to install with homebrew first (macOS)
if command -v brew &> /dev/null; then
    echo "Attempting to install pgsrip with Homebrew..."
    brew install pgsrip
    if [ $? -eq 0 ]; then
        echo "pgsrip successfully installed with Homebrew"
        exit 0
    else
        echo "Homebrew installation failed, trying Go install method..."
    fi
fi

# Try to install with go install
if command -v go &> /dev/null; then
    echo "Installing pgsrip with Go..."
    go install github.com/wader/pgsrip/cmd/pgsrip@latest
    if [ $? -eq 0 ]; then
        echo "pgsrip successfully installed with Go"
        # Make sure PATH includes Go bin directory
        export PATH="$PATH:$(go env GOPATH)/bin"
        echo "You may need to add $(go env GOPATH)/bin to your PATH"
        exit 0
    else
        echo "Go installation failed"
    fi
fi

# If we get here, both methods failed
echo "Failed to install pgsrip. Please install it manually:"
echo "Option 1: brew install pgsrip"
echo "Option 2: go install github.com/wader/pgsrip/cmd/pgsrip@latest"
exit 1
