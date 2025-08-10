# Subtitle Forge v1.6

A tool for extracting and converting subtitles from MKV files, available in both command-line (CLI) and graphical user interface (GUI) versions.

## Overview

This project provides two applications:
1. **CLI Version** - Command-line tool for extracting subtitles from MKV files
2. **GUI Version** - Fyne-based graphical application with enhanced features including PGS to SRT conversion

## What's New in v1.6

- **Enhanced Drag and Drop**: Improved drag and drop functionality in both Extract and Insert Subtitles tabs
- **Consistent User Experience**: File dropping now works reliably across all application tabs
- **Visual Feedback**: Better visual indicators when files are dropped
- **Subtitle Replacement Option**: New option to remove all existing subtitle tracks when inserting a new subtitle

## What's New in v1.5

- **Dependency Auto-Install**: Application now detects missing dependencies and offers to install them automatically
- **Improved Error Handling**: Better feedback when dependencies are missing or installation fails
- **Streamlined Setup**: One-click installation of required tools like ffmpeg, mkvtoolnix, and vobsub2srt

## What's New in v1.4.1

- **Window Size Persistence**: Application now remembers and restores your preferred window size
- **Keyboard Shortcuts**: Added convenient shortcuts for common actions:
  - **Ctrl+O**: Open MKV file
  - **Ctrl+D**: Change output directory
  - **Ctrl+L**: Load tracks
  - **Ctrl+E**: Start extraction

## What's New in v1.4

- **OCR Language Selection**: Manual language selection for PGS and VobSub subtitle conversion
- **Improved UI Layout**: Larger window size for better visibility
- **Enhanced Track Display**: Scrollable track list that can handle any number of subtitle tracks
- **Better Usability**: Optimized track list area to show more tracks at once

## What's New in v1.3

- **VobSub to SRT Conversion**: Convert VobSub (.idx/.sub) subtitles to SRT format using OCR
- **Improved Dependency Detection**: Better detection of required tools including vobsub2srt
- **Enhanced Language Support**: Automatic mapping between 3-letter and 2-letter language codes
- **Robust Error Handling**: Improved logging and error reporting for subtitle conversion

## Features

### CLI Version
- Extract subtitles from MKV files
- Support for multiple subtitle formats including SRT, ASS, and SUP
- Automatic naming of extracted subtitle files based on track properties

### GUI Version
- User-friendly graphical interface with two main tabs:
  - **Extract Subtitles**: Extract and convert subtitle tracks from MKV files
  - **Insert Subtitles**: Add external SRT subtitle files into MKV files
- Full drag and drop support in both tabs for easy file selection
- Convert PGS/SUP subtitles to SRT format using OCR
- Convert VobSub (.idx/.sub) subtitles to SRT format using OCR
- Convert ASS/SSA subtitles to SRT format
- Enhanced progress reporting:
  - Detailed progress bar showing percentage complete
  - Real-time frame processing status
  - Elapsed time tracking
  - Estimated time remaining calculation
- Detailed logging for troubleshooting
- Cross-platform support (macOS, Windows, Linux)
- Automatic dependency checking at startup with one-click installation
- Drag-and-drop support for MKV files
- Automatic output directory setting (defaults to MKV file location)
- Support button for donations
- Proper file permissions for extracted subtitle files

## Requirements

### CLI Version
- Go 1.16 or later
- `mkvmerge` and `mkvextract` tools from the MKVToolNix package
- `gocmd` library

### GUI Version
- Go 1.18 or later
- Fyne v2.6.1 or later
- [Deno](https://deno.land/) (for running the PGS to SRT conversion script)
- [mkvmerge and mkvextract](https://mkvtoolnix.download/) (part of MKVToolNix)
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) (used by the PGS-to-SRT and VobSub-to-SRT conversion)
- [VobSub2SRT](https://github.com/ruediger/VobSub2SRT) (for VobSub to SRT conversion)
- PGS-to-SRT conversion script

   
git clone https://github.com/leonard-slass/VobSub2SRT.git
cd VobSub2SRT
mkdir build
cd build
cmake -DCMAKE_POLICY_VERSION_MINIMUM=3.5 ..
sudo make install

## Installation

### CLI Version
1. Install Go from [golang.org](https://golang.org/dl/)
2. Install MKVToolNix from [mkvtoolnix.download](https://mkvtoolnix.download/)
3. Clone the repository and navigate to the project directory:
    ```sh
    git clone https://github.com/rhaseven7h/gmmmkvsubsextract.git
    cd gmmmkvsubsextract
    ```
4. Build the CLI version:
    ```sh
    go build -o gmmmkvsubsextract
    ```

### GUI Version

#### macOS
1. Extract the `gmmmkvsubsextract-macos.tar.gz` archive
2. Install Deno: `brew install deno`
3. Install MKVToolNix: `brew install mkvtoolnix`
4. Run the application: `./gmmmkvsubsextract-mac`

#### Windows
1. Extract the `gmmmkvsubsextract-windows.zip` archive
2. Install Deno: [Deno Installation](https://deno.land/#installation)
3. Install MKVToolNix: [MKVToolNix Download](https://mkvtoolnix.download/downloads.html)
4. Add both to your PATH environment variable
5. Run the application by double-clicking `gmmmkvsubsextract.exe`

#### Linux
1. Extract the `gmmmkvsubsextract-linux.tar.gz` archive
2. Install Deno: `curl -fsSL https://deno.land/x/install/install.sh | sh`
3. Install MKVToolNix: Use your distribution's package manager (e.g., `apt install mkvtoolnix`)
4. Run the application: `./gmmmkvsubsextract-linux`

#### Building from Source
1. Clone the repository
2. Navigate to the `fyne-gui` directory
3. Install Fyne dependencies: [Fyne Getting Started](https://developer.fyne.io/started/)
4. Run the build script: `./build.sh`

## Usage

### CLI Version
To extract subtitles from an MKV file, use the `-x` or `--extract` flag followed by the path to the MKV file:

## PGS to SRT Conversion Process

The application includes a powerful feature to convert PGS/SUP subtitle files (image-based subtitles) to SRT format (text-based subtitles) using Optical Character Recognition (OCR). This process involves several steps:

### How It Works

1. **Extraction**: First, the PGS subtitles are extracted from the MKV file using `mkvextract` as .sup files

2. **OCR Processing**: The extracted .sup files are then processed using a Deno-based script that:
   - Decodes the PGS/SUP format to extract individual subtitle frames
   - Uses Tesseract OCR to convert the subtitle images to text
   - Preserves timing information from the original subtitles
   - Formats the output as a standard SRT file

3. **Real-time Feedback**: During conversion, the application provides:
   - Progress updates
   - Elapsed time tracking
   - Detailed logs of the conversion process

### Requirements for OCR

- **Deno Runtime**: Required to execute the conversion script
- **Tesseract OCR**: The underlying OCR engine used for text recognition
- **Tessdata Files**: Language training data for Tesseract (English data included by default)

### Performance Considerations

- OCR conversion is CPU-intensive and may take significant time for longer subtitle tracks
- The quality of the OCR results depends on several factors:
  - Resolution and clarity of the original PGS subtitles
  - Font style used in the original subtitles
  - Language of the subtitles (English works best with the default configuration)

### Troubleshooting OCR Conversion

- If conversion fails, check that Deno is properly installed and in your PATH
- Verify that the Tesseract language data files are available
- For poor OCR quality, you may need to adjust the conversion parameters in the script
- The application creates detailed logs that can help diagnose conversion issues

## VobSub to SRT Conversion Process

The application also supports converting VobSub subtitles (.idx/.sub files) to SRT format using OCR technology. This feature works similarly to the PGS conversion but uses the vobsub2srt tool.

### How It Works

1. **Extraction**: First, the VobSub subtitles are extracted from the MKV file using `mkvextract` as .idx and .sub files

2. **OCR Processing**: The extracted files are then processed using the vobsub2srt tool that:
   - Reads the subtitle images from the .sub file and the timing information from the .idx file
   - Uses Tesseract OCR to convert the subtitle images to text
   - Automatically handles language detection and mapping
   - Formats the output as a standard SRT file

3. **Language Support**: The conversion process requires proper language mapping:
   - MKV files typically use 3-letter language codes (e.g., 'eng', 'fre', 'ger')
   - The vobsub2srt tool uses 2-letter language codes (e.g., 'en', 'fr', 'de')
   - The application automatically maps between these formats
   - You can manually select the OCR language from a dropdown menu for better accuracy

### Requirements for VobSub Conversion

- **vobsub2srt**: The command-line tool that performs the actual conversion
   - Should be installed at `/usr/local/bin/vobsub2srt`
   - Can be built from [VobSub2SRT GitHub repository](https://github.com/ruediger/VobSub2SRT)
- **Tesseract OCR**: The underlying OCR engine used for text recognition
- **Tessdata Files**: Language training data for Tesseract

./gmmmkvsubsextract -x /path/to/yourfile.mkv

### GUI Version
1. Load an MKV file using one of these methods:
   - Click "Select MKV File" to choose your MKV file using the file dialog
   - Or simply drag and drop an MKV file onto the application window
2. The output directory is automatically set to the same location as your MKV file
   - You can change it by clicking "Change Output Directory" if needed
3. Click "Load Tracks" to see available subtitle tracks
4. Select the subtitle tracks you want to extract/convert
5. Click "Start Extract" to begin the process
6. Monitor the progress in the application window

## Building from Source

### Prerequisites

- **Go 1.18 or later** (REQUIRED): You must have Go installed to build from source
  - Install on macOS: `brew install go` or download from [golang.org](https://golang.org/dl/)
  - Install on Linux: `apt install golang` (Ubuntu/Debian) or download from [golang.org](https://golang.org/dl/)
  - Install on Windows: Download and run the installer from [golang.org](https://golang.org/dl/)
  - Verify installation: `go version` should show the installed version
- **Fyne dependencies**: Required for GUI compilation
  - Follow the setup guide at [Fyne Getting Started](https://developer.fyne.io/started/)
  - macOS: `brew install gcc`
  - Linux: `apt install gcc libgl1-mesa-dev xorg-dev`
  - Windows: Install MinGW or MSYS2

### Build Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/VenimK/Subtitle-Forge.git
   ```

2. Navigate to the `fyne-gui` directory:
   ```bash
   cd Subtitle-Forge/fyne-gui
   ```

3. Make the build script executable:
   ```bash
   chmod +x build.sh
   ```

4. Run the build script:
   ```bash
   ./build.sh
   ```

5. Run the compiled application:
   ```bash
   # On macOS
   ./build/subtitle-forge-mac
   
   # On Linux
   ./build/subtitle-forge-linux
   ```

For cross-compilation, you may need additional tools:
- For Windows builds on macOS: `brew install mingw-w64`
- For Linux builds on macOS: `brew install FiloSottile/musl-cross/musl-cross`

## Dependency Auto-Install

Subtitle Forge v1.5 introduces a new feature that automatically detects missing dependencies and offers to install them for you:

1. **Automatic Detection**: When you start the application, it checks for all required dependencies
2. **Installation Prompt**: If any dependencies are missing, you'll see a notification with an "Install" button
3. **One-Click Installation**: Click the button to automatically install the missing dependency
4. **Progress Tracking**: A progress dialog shows the installation status
5. **Completion Notification**: You'll be notified when installation is complete

### Supported Dependencies

- **ffmpeg**: For media processing and subtitle conversion
- **mkvtoolnix** (mkvmerge, mkvextract): For working with MKV files
- **vobsub2srt**: For converting VobSub subtitles to SRT format

### Requirements

- **Homebrew**: On macOS, dependencies are installed via Homebrew
- **sudo access**: Some installations may require administrator privileges
- **cmake**: Required for building vobsub2srt from source
- **tesseract**: Required for OCR functionality

## Troubleshooting

- The application automatically checks for required dependencies at startup
- Missing dependencies will be clearly indicated in the application window with an option to install them
- If automatic installation fails, detailed error messages will guide you through manual installation
- Ensure Deno, mkvmerge, and mkvextract are in your PATH
- Check the conversion logs in the output directory
- For permission issues, try running the application with administrator privileges

## Updating the Application

If you've previously cloned the repository and want to update to the latest version, follow these steps:

### Clean Update (Recommended)
1. Remove any local build artifacts before pulling:
   ```sh
   cd gmmmkvsubsextract
   rm -rf fyne-gui/build/*
   git pull
   ```

### If You Encounter Conflicts
If you see errors like "Your local changes would be overwritten by merge", you can:

1. Stash your local changes:
   ```sh
   git stash
   git pull
   ```
   
2. Or discard local changes to specific files:
   ```sh
   git checkout -- fyne-gui/build/
   git pull
   ```

3. After updating, rebuild the application:
   ```sh
   cd fyne-gui
   ./build.sh
   ```

## License

[MIT License](LICENSE)
