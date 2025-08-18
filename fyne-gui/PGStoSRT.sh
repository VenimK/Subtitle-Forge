#!/bin/bash
# PGStoSRT.sh - Script to install PGS to SRT conversion tool
# This script installs the PGS to SRT conversion tool from GitHub

set -e

echo "=== PGS to SRT Installation Script ==="
echo "This script will install the PGS to SRT conversion tool to your home directory."

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is not installed or not in PATH"
        echo "$2"
        return 1
    fi
    return 0
}

# Check for required dependencies
echo "Checking for required dependencies..."

# Check for git
check_command git "Please install git first (e.g., brew install git)" || exit 1

# Check for deno
check_command deno "Please install deno first (e.g., brew install deno)" || exit 1

echo "All dependencies are installed. Proceeding with installation..."

# Remove existing directory and zip if they exist
if [ -d "$HOME/pgs-to-srt" ]; then
  echo "Removing existing pgs-to-srt directory..."
  rm -rf "$HOME/pgs-to-srt"
fi

if [ -f "$HOME/pgs-to-srt.zip" ]; then
  echo "Removing existing pgs-to-srt.zip file..."
  rm -f "$HOME/pgs-to-srt.zip"
fi

# Step 1: Clone the repository
echo "Cloning PGS-to-SRT repository..."
git clone https://github.com/wydengyre/pgs-to-srt.git "$HOME/pgs-to-srt"

# Step 2: Download the ZIP file
echo "Downloading PGS-to-SRT ZIP release..."
ZIP_URL="https://github.com/wydengyre/pgs-to-srt/releases/download/release-5/pgs-to-srt.zip"

# Use curl with fallback to wget
if command -v curl &> /dev/null; then
  curl -L "$ZIP_URL" -o "$HOME/pgs-to-srt.zip"
else
  wget -O "$HOME/pgs-to-srt.zip" "$ZIP_URL"
fi

# Step 3: Extract the ZIP file to a temporary directory first
echo "Extracting files..."
TMP_EXTRACT_DIR="$HOME/pgs-to-srt-tmp"
mkdir -p "$TMP_EXTRACT_DIR"
unzip -o "$HOME/pgs-to-srt.zip" -d "$TMP_EXTRACT_DIR"

# Step 4: Find the pgs-to-srt.js file
echo "Locating pgs-to-srt.js file..."
JS_FILE=$(find "$TMP_EXTRACT_DIR" -name "pgs-to-srt.js" | head -n 1)

if [ -z "$JS_FILE" ]; then
  echo "Installation failed: pgs-to-srt.js not found in extracted files"
  rm -rf "$TMP_EXTRACT_DIR"
  rm -f "$HOME/pgs-to-srt.zip"
  exit 1
fi

echo "Found pgs-to-srt.js at: $JS_FILE"

# Copy all extracted files to the target location
echo "Copying all extracted files to $HOME/pgs-to-srt..."

# Find the parent directory containing all extracted files
JS_DIR=$(dirname "$JS_FILE")

# Copy all files and directories from the extraction directory
cp -R "$JS_DIR"/* "$HOME/pgs-to-srt/"

# Step 5: Install using deno
echo "Installing PGS-to-SRT using deno..."
cd "$HOME/pgs-to-srt"
deno install --global -f --allow-read "pgs-to-srt.js"

# Verify the tessdata_fast directory exists
if [ ! -d "$HOME/pgs-to-srt/tessdata_fast" ]; then
  echo "Warning: tessdata_fast directory not found. Creating it..."
  mkdir -p "$HOME/pgs-to-srt/tessdata_fast"
  
  # Check if we need to download language files
  if [ ! -f "$HOME/pgs-to-srt/tessdata_fast/eng.traineddata" ]; then
    echo "Downloading English language data..."
    TESSDATA_URL="https://github.com/tesseract-ocr/tessdata_fast/raw/main/eng.traineddata"
    if command -v curl &> /dev/null; then
      curl -L "$TESSDATA_URL" -o "$HOME/pgs-to-srt/tessdata_fast/eng.traineddata"
    else
      wget -O "$HOME/pgs-to-srt/tessdata_fast/eng.traineddata" "$TESSDATA_URL"
    fi
  fi
fi

# Clean up
echo "Cleaning up temporary files..."
rm -rf "$TMP_EXTRACT_DIR"
rm -f "$HOME/pgs-to-srt.zip"

# Set permissions
chmod +x "$HOME/pgs-to-srt/pgs-to-srt.js"

echo "=== Installation Complete ==="
echo "PGS-to-SRT has been installed to: $HOME/pgs-to-srt/pgs-to-srt.js"
echo "Language files are located in: $HOME/pgs-to-srt/tessdata_fast/"
echo "You can run it using: deno run --allow-read --allow-write $HOME/pgs-to-srt/pgs-to-srt.js"
echo ""
echo "To convert a PGS subtitle file:"
echo "deno run --allow-read --allow-write $HOME/pgs-to-srt/pgs-to-srt.js [tessdata_path] [input_file] > [output_file]"
exit 0
