#!/bin/bash
# bundle_linux_deps.sh
# Script to create a self-contained Linux package with bundled dependencies

set -e

echo "Creating self-contained Linux package with bundled dependencies..."

# Create directory structure
BUNDLE_DIR="build/linux-bundle"
mkdir -p "$BUNDLE_DIR/bin"
mkdir -p "$BUNDLE_DIR/lib"
mkdir -p "$BUNDLE_DIR/share"

# Copy the main binary
if [ -f "build/subtitle-forge-linux" ]; then
    cp build/subtitle-forge-linux "$BUNDLE_DIR/subtitle-forge"
    chmod +x "$BUNDLE_DIR/subtitle-forge"
else
    echo "Error: subtitle-forge-linux binary not found. Build it first with: ./build.sh --all"
    exit 1
fi

# Function to download and extract static binaries
download_static_binary() {
    local name=$1
    local url=$2
    local extract_path=$3
    local target_name=$4
    
    echo "Downloading $name..."
    
    if command -v wget &> /dev/null; then
        wget -q "$url" -O "/tmp/$name.tar.gz"
    elif command -v curl &> /dev/null; then
        curl -sL "$url" -o "/tmp/$name.tar.gz"
    else
        echo "Error: Neither wget nor curl found. Cannot download dependencies."
        return 1
    fi
    
    # Extract and copy binary
    cd /tmp
    tar -xzf "$name.tar.gz"
    if [ -f "$extract_path" ]; then
        cp "$extract_path" "$BUNDLE_DIR/bin/$target_name"
        chmod +x "$BUNDLE_DIR/bin/$target_name"
        echo "✅ $name bundled successfully"
    else
        echo "⚠️  Could not find $name binary at $extract_path"
    fi
    
    # Cleanup
    rm -f "/tmp/$name.tar.gz"
    rm -rf "/tmp/${extract_path%/*}"
    cd - > /dev/null
}

# Bundle FFmpeg (static build)
echo "Bundling FFmpeg..."
if command -v ffmpeg &> /dev/null; then
    cp "$(which ffmpeg)" "$BUNDLE_DIR/bin/"
    echo "✅ FFmpeg bundled from system"
else
    echo "⚠️  FFmpeg not found on system, skipping bundle"
fi

# Bundle Deno
echo "Bundling Deno..."
DENO_VERSION="v1.37.2"
case "$(uname -m)" in
    x86_64)
        DENO_ARCH="x86_64-unknown-linux-gnu"
        ;;
    aarch64|arm64)
        DENO_ARCH="aarch64-unknown-linux-gnu"
        ;;
    *)
        echo "⚠️  Unsupported architecture for Deno bundle"
        DENO_ARCH=""
        ;;
esac

if [ -n "$DENO_ARCH" ]; then
    DENO_URL="https://github.com/denoland/deno/releases/download/$DENO_VERSION/deno-$DENO_ARCH.zip"
    echo "Downloading Deno $DENO_VERSION for $DENO_ARCH..."
    
    if command -v wget &> /dev/null; then
        wget -q "$DENO_URL" -O "/tmp/deno.zip"
    elif command -v curl &> /dev/null; then
        curl -sL "$DENO_URL" -o "/tmp/deno.zip"
    fi
    
    if [ -f "/tmp/deno.zip" ]; then
        cd /tmp
        unzip -q deno.zip
        if [ -f "deno" ]; then
            cp deno "$BUNDLE_DIR/bin/"
            chmod +x "$BUNDLE_DIR/bin/deno"
            echo "✅ Deno bundled successfully"
        fi
        rm -f deno.zip deno
        cd - > /dev/null
    fi
fi

# Bundle Tesseract (if available)
echo "Bundling Tesseract..."
if command -v tesseract &> /dev/null; then
    cp "$(which tesseract)" "$BUNDLE_DIR/bin/"
    echo "✅ Tesseract bundled from system"
    
    # Copy tessdata if available
    TESSDATA_DIRS=(
        "/usr/share/tesseract-ocr/tessdata"
        "/usr/share/tessdata"
        "/opt/homebrew/share/tessdata"
    )
    
    for tessdata_dir in "${TESSDATA_DIRS[@]}"; do
        if [ -d "$tessdata_dir" ]; then
            mkdir -p "$BUNDLE_DIR/share/tessdata"
            cp -r "$tessdata_dir"/* "$BUNDLE_DIR/share/tessdata/" 2>/dev/null || true
            echo "✅ Tessdata copied from $tessdata_dir"
            break
        fi
    done
else
    echo "⚠️  Tesseract not found on system, skipping bundle"
fi

# Try to bundle MKVToolNix
echo "Bundling MKVToolNix..."
if command -v mkvmerge &> /dev/null && command -v mkvextract &> /dev/null; then
    cp "$(which mkvmerge)" "$BUNDLE_DIR/bin/"
    cp "$(which mkvextract)" "$BUNDLE_DIR/bin/"
    echo "✅ MKVToolNix bundled from system"
else
    echo "⚠️  MKVToolNix not found on system, skipping bundle"
fi

# Copy installation scripts
echo "Copying installation scripts..."
cp LinuxHeadless.sh "$BUNDLE_DIR/install_dependencies.sh"
cp install_vobsub2srt.sh "$BUNDLE_DIR/" 2>/dev/null || echo "⚠️  install_vobsub2srt.sh not found"
cp install_pgsrip.sh "$BUNDLE_DIR/" 2>/dev/null || echo "⚠️  install_pgsrip.sh not found"
cp install_tessdata.sh "$BUNDLE_DIR/" 2>/dev/null || echo "⚠️  install_tessdata.sh not found"

# Copy PGS-to-SRT script
if [ -f "PGStoSRT.sh" ]; then
    cp PGStoSRT.sh "$BUNDLE_DIR/"
    echo "✅ PGStoSRT.sh copied"
fi

# Copy README and LICENSE
if [ -f "../README.md" ]; then
    cp ../README.md "$BUNDLE_DIR/"
elif [ -f "README.md" ]; then
    cp README.md "$BUNDLE_DIR/"
fi

if [ -f "../LICENSE.md" ]; then
    cp ../LICENSE.md "$BUNDLE_DIR/"
elif [ -f "../LICENSE" ]; then
    cp ../LICENSE "$BUNDLE_DIR/"
fi

# Create launcher script that sets up PATH
cat > "$BUNDLE_DIR/subtitle-forge.sh" << 'EOF'
#!/bin/bash
# Subtitle Forge Launcher Script

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Add bundled binaries to PATH
export PATH="$SCRIPT_DIR/bin:$PATH"

# Set tessdata path if bundled
if [ -d "$SCRIPT_DIR/share/tessdata" ]; then
    export TESSDATA_PREFIX="$SCRIPT_DIR/share"
fi

# Run the application
exec "$SCRIPT_DIR/subtitle-forge" "$@"
EOF

chmod +x "$BUNDLE_DIR/subtitle-forge.sh"

# Create installation guide
cat > "$BUNDLE_DIR/INSTALL.md" << 'EOF'
# Subtitle Forge Self-Contained Linux Package

This package includes bundled dependencies to minimize external requirements.

## Quick Start
1. Extract this archive: `tar -xzf subtitle-forge-linux-bundle.tar.gz`
2. Run the launcher: `./subtitle-forge.sh`

## What's Included
- Subtitle Forge binary
- Bundled dependencies (when available):
  - FFmpeg
  - Deno runtime
  - Tesseract OCR
  - MKVToolNix (mkvmerge, mkvextract)
- Installation scripts for missing dependencies
- Tesseract language data (English)

## If Dependencies Are Missing
If some dependencies couldn't be bundled, install them using:
```bash
sudo ./install_dependencies.sh
```

## Manual Installation
For missing dependencies, install manually:
- `sudo apt install mkvtoolnix ffmpeg tesseract-ocr` (Debian/Ubuntu)
- `sudo dnf install mkvtoolnix ffmpeg tesseract` (Fedora)
- `sudo pacman -S mkvtoolnix ffmpeg tesseract` (Arch)

## Running
Use the launcher script for best compatibility:
```bash
./subtitle-forge.sh
```

Or run the binary directly:
```bash
./subtitle-forge
```
EOF

# Create the bundle archive
echo "Creating bundle archive..."
tar -czf "build/subtitle-forge-linux-bundle.tar.gz" -C build linux-bundle

echo "✅ Self-contained Linux bundle created: build/subtitle-forge-linux-bundle.tar.gz"
echo ""
echo "Bundle contents:"
ls -la "$BUNDLE_DIR/"
echo ""
echo "Bundled binaries:"
ls -la "$BUNDLE_DIR/bin/" 2>/dev/null || echo "No binaries bundled"
