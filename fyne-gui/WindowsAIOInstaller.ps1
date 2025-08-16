# install_dependencies.ps1
# PowerShell script to install all dependencies for Subtitle Forge on Windows
# This script will install Chocolatey if not already installed, then use it to install required tools

# Output function with timestamp
function Write-Log {
    param (
        [string]$Message,
        [string]$Level = "INFO"
    )
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message"
}

# Check if running as administrator
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Log "This script requires administrator privileges. Please run as administrator." "ERROR"
    Write-Log "Right-click on the script and select 'Run as administrator'." "INFO"
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Log "Starting dependency installation for Subtitle Forge..."

# Check if Chocolatey is installed
$chocoInstalled = $false
if (Get-Command choco -ErrorAction SilentlyContinue) {
    Write-Log "Chocolatey is already installed."
    $chocoInstalled = $true
} else {
    Write-Log "Chocolatey is not installed. Installing now..."
    try {
        # Install Chocolatey
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
        
        # Check if installation was successful
        if (Get-Command choco -ErrorAction SilentlyContinue) {
            Write-Log "Chocolatey installed successfully."
            $chocoInstalled = $true
        } else {
            Write-Log "Failed to install Chocolatey." "ERROR"
            Read-Host "Press Enter to exit"
            exit 1
        }
    } catch {
        Write-Log "Error installing Chocolatey: $_" "ERROR"
        Write-Log "Please install Chocolatey manually from https://chocolatey.org/install" "INFO"
        Read-Host "Press Enter to exit"
        exit 1
    }
}

# Install dependencies using Chocolatey
if ($chocoInstalled) {
    $dependencies = @(
        @{Name = "mkvtoolnix"; Description = "MKVToolNix (provides mkvmerge and mkvextract)"},
        @{Name = "ffmpeg"; Description = "FFmpeg multimedia framework"},
        @{Name = "deno"; Description = "Deno runtime"},
        @{Name = "tesseract"; Description = "Tesseract OCR engine"},
        @{Name = "golang"; Description = "Go programming language"}
    )
    
    foreach ($dep in $dependencies) {
        Write-Log "Installing $($dep.Description)..."
        try {
            choco install $dep.Name -y
            Write-Log "$($dep.Description) installed successfully."
        } catch {
            Write-Log "Error installing $($dep.Name): $_" "ERROR"
        }
    }
    
    # Refresh environment variables to make the newly installed tools available
    Write-Log "Refreshing environment variables..."
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    Write-Log "All dependencies have been installed successfully!"
    Write-Log "You may need to restart your computer for some changes to take effect."
} else {
    Write-Log "Chocolatey installation failed. Cannot install dependencies." "ERROR"
}

Read-Host "Press Enter to exit"
