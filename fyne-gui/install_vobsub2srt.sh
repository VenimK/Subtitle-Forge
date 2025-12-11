#!/bin/bash

# Script to install VobSub2SRT from leonard-slass fork
echo "Installing VobSub2SRT from leonard-slass fork..."

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
check_command git "Please install git: brew install git" || exit 1

# Check for cmake
check_command cmake "Please install cmake: brew install cmake" || exit 1

# Check for make
check_command make "Please install Xcode command line tools: xcode-select --install" || exit 1

# Check for tesseract (required for OCR)
check_command tesseract "Please install tesseract: brew install tesseract" || exit 1

echo "All dependencies are installed. Proceeding with installation..."

# Create a temporary directory for the build
TEMP_DIR=$(mktemp -d)
echo "Using temporary directory: $TEMP_DIR"

# Navigate to the temporary directory
cd "$TEMP_DIR" || exit 1

# Clone the repository
echo "Cloning the repository..."
git clone https://github.com/leonard-slass/VobSub2SRT.git
if [ $? -ne 0 ]; then
    echo "Failed to clone repository."
    echo "Please check your internet connection and try again."
    exit 1
fi

# Navigate to the cloned repository
cd VobSub2SRT || exit 1

# Fix CMakeLists.txt to update cmake_minimum_required version
echo "Patching CMakeLists.txt..."
if [ -f "CMakeLists.txt" ]; then
    # Replace any old cmake_minimum_required version with 3.5
    sed -i.bak 's/cmake_minimum_required.*$/cmake_minimum_required(VERSION 3.5)/g' CMakeLists.txt
    # Move cmake_minimum_required to the very first line if it's not already
    if head -n 1 CMakeLists.txt | grep -q "cmake_minimum_required"; then
        echo "✅ CMakeLists.txt already has cmake_minimum_required at the top"
    else
        # Extract cmake_minimum_required line and remove it from its current position
        CMAKE_LINE=$(grep "cmake_minimum_required" CMakeLists.txt | head -n 1)
        grep -v "cmake_minimum_required" CMakeLists.txt > CMakeLists.txt.tmp
        # Add it at the top
        echo "$CMAKE_LINE" > CMakeLists.txt.new
        cat CMakeLists.txt.tmp >> CMakeLists.txt.new
        mv CMakeLists.txt.new CMakeLists.txt
        rm -f CMakeLists.txt.tmp
        echo "✅ Moved cmake_minimum_required to top of CMakeLists.txt"
    fi
    rm -f CMakeLists.txt.bak
fi

# Create build directory
echo "Creating build directory..."
mkdir -p build
cd build || exit 1

# Run cmake
echo "Running cmake..."
cmake ..
if [ $? -ne 0 ]; then
    echo "Failed to configure with cmake."
    echo "This might be due to missing dependencies or configuration issues."
    echo "Try installing additional dependencies: brew install leptonica libtiff"
    exit 1
fi

# Build and install
echo "Building and installing..."
make
if [ $? -ne 0 ]; then
    echo "Failed to build."
    echo "This might be due to compilation errors or missing dependencies."
    exit 1
fi

# Install (may require sudo)
echo "Installing (may require sudo)..."

# Check if we're running in a GUI context (no TTY)
if [ ! -t 0 ]; then
    # Running from GUI - use osascript for graphical sudo prompt
    echo "Running from GUI context - using graphical password prompt..."
    osascript -e 'do shell script "cd '"$(pwd)"' && make install" with administrator privileges'
    if [ $? -ne 0 ]; then
        echo "Failed to install via osascript."
        exit 1
    fi
else
    # Running from terminal - use regular sudo
    sudo make install
    if [ $? -ne 0 ]; then
        echo "Failed to install. Try running this script with sudo."
        echo "Command: sudo ./install_vobsub2srt.sh"
        exit 1
    fi
fi

# Verify installation
echo "Verifying installation..."
if command -v vobsub2srt &> /dev/null; then
    echo "VobSub2SRT installed successfully!"
    vobsub2srt --version
else
    echo "VobSub2SRT installation could not be verified."
    echo "The binary might not be in your PATH. Try running: sudo ln -s /usr/local/bin/vobsub2srt /usr/bin/vobsub2srt"
    exit 1
fi

# Clean up
echo "Cleaning up temporary files..."
cd
rm -rf "$TEMP_DIR"

echo "Installation complete!"
exit 0
