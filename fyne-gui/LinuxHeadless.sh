#!/bin/bash
# install_dependencies.sh
# Script to install all dependencies for Subtitle Forge on Linux
# Designed for headless servers

# Output function with timestamp
log_message() {
    local level="$1"
    local message="$2"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo "[$timestamp] [$level] $message"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_message "ERROR" "This script requires root privileges. Please run with sudo."
    exit 1
fi

log_message "INFO" "Starting dependency installation for Subtitle Forge..."

# Detect package manager
if command -v apt-get &> /dev/null; then
    PKG_MANAGER="apt"
    log_message "INFO" "Detected Debian/Ubuntu system using apt"
elif command -v dnf &> /dev/null; then
    PKG_MANAGER="dnf"
    log_message "INFO" "Detected Fedora/RHEL system using dnf"
elif command -v yum &> /dev/null; then
    PKG_MANAGER="yum"
    log_message "INFO" "Detected CentOS/RHEL system using yum"
elif command -v pacman &> /dev/null; then
    PKG_MANAGER="pacman"
    log_message "INFO" "Detected Arch Linux system using pacman"
else
    log_message "ERROR" "Unsupported package manager. Please install dependencies manually."
    exit 1
fi

# Update package lists
log_message "INFO" "Updating package lists..."
case $PKG_MANAGER in
    apt)
        apt-get update -y
        ;;
    dnf)
        dnf check-update
        ;;
    yum)
        yum check-update
        ;;
    pacman)
        pacman -Sy
        ;;
esac

# Install dependencies based on package manager
install_dependencies() {
    case $PKG_MANAGER in
        apt)
            # For Debian/Ubuntu
            log_message "INFO" "Installing MKVToolNix..."
            apt-get install -y mkvtoolnix
            
            log_message "INFO" "Installing FFmpeg..."
            apt-get install -y ffmpeg
            
            log_message "INFO" "Installing Tesseract OCR..."
            apt-get install -y tesseract-ocr
            
            log_message "INFO" "Installing Go..."
            apt-get install -y golang
            
            log_message "INFO" "Installing Deno dependencies..."
            apt-get install -y curl unzip
            ;;
            
        dnf|yum)
            # For Fedora/RHEL/CentOS
            log_message "INFO" "Installing MKVToolNix..."
            $PKG_MANAGER install -y mkvtoolnix
            
            log_message "INFO" "Installing FFmpeg..."
            $PKG_MANAGER install -y ffmpeg
            
            log_message "INFO" "Installing Tesseract OCR..."
            $PKG_MANAGER install -y tesseract
            
            log_message "INFO" "Installing Go..."
            $PKG_MANAGER install -y golang
            
            log_message "INFO" "Installing Deno dependencies..."
            $PKG_MANAGER install -y curl unzip
            ;;
            
        pacman)
            # For Arch Linux
            log_message "INFO" "Installing MKVToolNix..."
            pacman -S --noconfirm mkvtoolnix
            
            log_message "INFO" "Installing FFmpeg..."
            pacman -S --noconfirm ffmpeg
            
            log_message "INFO" "Installing Tesseract OCR..."
            pacman -S --noconfirm tesseract
            
            log_message "INFO" "Installing Go..."
            pacman -S --noconfirm go
            
            log_message "INFO" "Installing Deno dependencies..."
            pacman -S --noconfirm curl unzip
            ;;
    esac
}

# Install Deno (not commonly available in package managers)
install_deno() {
    log_message "INFO" "Installing Deno runtime..."
    if command -v deno &> /dev/null; then
        log_message "INFO" "Deno is already installed."
    else
        curl -fsSL https://deno.land/install.sh | sh
        
        # Add to PATH for current session
        export DENO_INSTALL="$HOME/.deno"
        export PATH="$DENO_INSTALL/bin:$PATH"
        
        # Add to .bashrc for persistence
        if ! grep -q "DENO_INSTALL" "$HOME/.bashrc"; then
            echo 'export DENO_INSTALL="$HOME/.deno"' >> "$HOME/.bashrc"
            echo 'export PATH="$DENO_INSTALL/bin:$PATH"' >> "$HOME/.bashrc"
        fi
        
        log_message "INFO" "Deno installed successfully."
    fi
}

# Install VobSub2SRT (optional)
install_vobsub2srt() {
    log_message "INFO" "Installing VobSub2SRT dependencies..."
    
    case $PKG_MANAGER in
        apt)
            apt-get install -y cmake build-essential libpng-dev libtesseract-dev
            ;;
        dnf|yum)
            $PKG_MANAGER install -y cmake gcc-c++ libpng-devel tesseract-devel
            ;;
        pacman)
            pacman -S --noconfirm cmake base-devel libpng tesseract
            ;;
    esac
    
    log_message "INFO" "Building VobSub2SRT from source..."
    
    # Create temp directory
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Clone and build
    git clone https://github.com/ruediger/VobSub2SRT.git
    cd VobSub2SRT
    ./configure
    make
    make install
    
    # Clean up
    cd /
    rm -rf "$TEMP_DIR"
    
    log_message "INFO" "VobSub2SRT installed successfully."
}

# Run installations
install_dependencies
install_deno
install_vobsub2srt

log_message "INFO" "All dependencies have been installed successfully!"
log_message "INFO" "You can now run Subtitle Forge on this server."
