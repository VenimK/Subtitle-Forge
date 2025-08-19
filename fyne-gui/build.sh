#!/bin/bash
# Build script for Subtitle Forge application

# Parse command line arguments
BUILD_ALL=false
VERSION="v1.6.9b"

for arg in "$@"; do
    case $arg in
        --all)
            BUILD_ALL=true
            ;;
        --version=*)
            VERSION="${arg#*=}"
            ;;
    esac
done

echo "Building version: $VERSION"

# Function to check if a command exists
check_dependency() {
    local cmd=$1
    local name=$2
    local install_cmd=$3
    
    echo "Checking for $name..."
    if ! command -v $cmd &> /dev/null; then
        echo "❌ $name not found. It's required to build Subtitle Forge."
        echo "   Install with: $install_cmd"
        return 1
    else
        echo "✅ $name found: $(command -v $cmd)"
        return 0
    fi
}

# Check for required dependencies
echo "Checking dependencies..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Attempting to install with Homebrew..."
    
    # Check if Homebrew is installed
    if ! command -v brew &> /dev/null; then
        echo "❌ Homebrew not found. Please install Homebrew first:"
        echo "   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        exit 1
    fi
    
    # Install Go using Homebrew
    echo "Installing Go with Homebrew..."
    brew install go
    
    # Verify installation
    if ! command -v go &> /dev/null; then
        echo "❌ Failed to install Go. Please install it manually:"
        echo "   https://golang.org/dl/"
        exit 1
    else
        echo "✅ Go installed successfully: $(command -v go)"
    fi
else
    echo "✅ Go found: $(command -v go)"
fi

# Check for gcc (required by Fyne)
if ! command -v gcc &> /dev/null; then
    echo "❌ GCC not found. Attempting to install with Homebrew..."
    
    # Check if Homebrew is installed (should already be checked above, but just in case)
    if ! command -v brew &> /dev/null; then
        echo "❌ Homebrew not found. Please install Homebrew first."
        echo "   Warning: GCC not found. You may encounter issues with Fyne GUI compilation."
    else
        # Install GCC using Homebrew
        echo "Installing GCC with Homebrew..."
        brew install gcc
        
        # Verify installation
        if ! command -v gcc &> /dev/null; then
            echo "❌ Failed to install GCC. You may encounter issues with Fyne GUI compilation."
        else
            echo "✅ GCC installed successfully: $(command -v gcc)"
        fi
    fi
else
    echo "✅ GCC found: $(command -v gcc)"
fi

# Remove existing build directory if it exists
if [ -d "build" ]; then
    echo "Removing existing build directory..."
    rm -rf build
fi

# Create fresh build directory      
echo "Creating build directory..."
mkdir -p build

# Build for macOS
echo "Building for macOS..."
go build -ldflags="-X 'main.AppVersion=$VERSION'" -o build/subtitle-forge-mac
if [ $? -eq 0 ]; then
    echo "✅ macOS build successful"
else
    echo "❌ macOS build failed"
    exit 1
fi

# Windows build removed as requested
echo "Windows build disabled"

# Only build for Linux if --all flag is provided
if [ "$BUILD_ALL" = true ]; then
    # Build for Linux (requires CGO)
    echo "Building for Linux..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -ldflags="-X 'main.AppVersion=$VERSION'" -o build/subtitle-forge-linux
    if [ $? -eq 0 ]; then
        echo "✅ Linux build successful"
    else
        echo "❌ Linux build failed (you may need to install musl-cross for cross-compilation)"
        echo "   Install with: brew install FiloSottile/musl-cross/musl-cross"
    fi
else
    echo "Skipping Linux build (use --all flag to build for all platforms)"
fi

echo "Creating distribution packages..."

# Check if README.md exists in the project root
README_PATH="../README.md"
if [ ! -f "$README_PATH" ]; then
    README_PATH="README.md"
    if [ ! -f "$README_PATH" ]; then
        echo "Warning: README.md not found"
        README_PATH=""
    fi
fi

# Create macOS package
if [ -f "build/subtitle-forge-mac" ]; then
    echo "Creating macOS package..."
    mkdir -p build/macos
    cp build/subtitle-forge-mac build/macos/
    
    # Copy scripts if they exist
    if [ -d "../scripts" ]; then
        cp -r ../scripts build/macos/
    elif [ -d "scripts" ]; then
        cp -r scripts build/macos/
    else
        echo "Warning: scripts directory not found"
    fi
    
    # Copy README and LICENSE if they exist
    if [ -n "$README_PATH" ]; then
        cp "$README_PATH" build/macos/
    fi
    
    if [ -f "../LICENSE" ]; then
        cp ../LICENSE build/macos/
    elif [ -f "LICENSE" ]; then
        cp LICENSE build/macos/
    else
        echo "Warning: LICENSE file not found"
    fi
    
    tar -czf build/subtitle-forge-macos.tar.gz -C build macos
    echo "✅ macOS package created"
fi

# Windows package creation removed as requested
echo "Windows packaging disabled"

# Create Linux package only if Linux build exists
if [ -f "build/subtitle-forge-linux" ]; then
    echo "Creating Linux package..."
    mkdir -p build/linux
    cp build/subtitle-forge-linux build/linux/
    
    # Copy scripts if they exist
    if [ -d "../scripts" ]; then
        cp -r ../scripts build/linux/
    elif [ -d "scripts" ]; then
        cp -r scripts build/linux/
    else
        echo "Warning: scripts directory not found"
    fi
    
    # Copy README and LICENSE if they exist
    if [ -n "$README_PATH" ]; then
        cp "$README_PATH" build/linux/
    fi
    
    if [ -f "../LICENSE" ]; then
        cp ../LICENSE build/linux/
    elif [ -f "LICENSE" ]; then
        cp LICENSE build/linux/
    else
        echo "Warning: LICENSE file not found"
    fi
    
    tar -czf build/subtitle-forge-linux.tar.gz -C build linux
    echo "✅ Linux package created"
fi

echo "Build process completed. Check the 'build' directory for binaries and packages."
