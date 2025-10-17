# Subtitle Forge v2.0.0
> Powerful subtitle extraction, conversion, and **AI translation** tool for **macOS**, **Linux**, and **Windows** 🔹 Extract, convert, translate, and manage subtitles with ease 🔹
> [![GitHub Release](https://img.shields.io/github/v/release/VenimK/Subtitle-Forge)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![GitHub Release Date](https://img.shields.io/github/release-date/VenimK/Subtitle-Forge?style=flat)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub All Releases](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/total.svg)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub Downloads Latest](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/latest/total?style=flat&label=downloads%40latest&color=orange)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![Donate](https://img.shields.io/badge/Donate-PayPal-blue.svg?logo=paypal&style=flat-square&label=Support%20☕)](https://paypal.me/VenimK)

A comprehensive tool for extracting, converting, and translating subtitles from MKV files, available in both command-line (CLI) and graphical user interface (GUI) versions.

## Overview

This project provides two applications:
1. **CLI Version** - Command-line tool for extracting subtitles from MKV files
2. **GUI Version** - Fyne-based graphical application with enhanced features including:
   - **Extract Subtitles** - Extract from MKV files with batch processing and PGS to SRT conversion
   - **Insert Subtitles** - Insert subtitles (all formats) into video files
   - **Convert Subtitles** - Universal subtitle format converter with batch processing and advanced options
   - **AI Translation** - 🆕 Translate subtitles using Google Gemini AI with intelligent batch processing
   - **Settings** - Dependency management and seasonal dark themes

## Installation

### macOS
1. Download the latest release for macOS
2. Unzip the file
3. **Important**: After unzipping, open Terminal and run the following command to remove quarantine attributes:
   ```
   xattr -cr "Subtitle Forge.app"
   ```
4. Move the app to your Applications folder

### Windows
1. Download the latest release for Windows
2. Unzip the file
3. Run the included `WindowsAIOInstaller.ps1` script as administrator to install required dependencies
4. Launch the application

### Linux
1. Download the latest release for Linux
2. Unzip the file
3. For headless servers, use the included `install_dependencies.sh` script to install dependencies:
   ```
   sudo ./install_dependencies.sh
   ```
4. Launch the application

## What's New in v2.0.0 🎉

### 🤖 AI Translation - Revolutionary New Feature!
**Transform your subtitle workflow with intelligent AI-powered translation!**

- **Google Gemini AI Integration**: Professional-quality translations using Google's advanced Gemini 2.5 Flash model
- **Intelligent Batch Processing**: Translate 100+ subtitle entries per API call for maximum efficiency
- **Multi-Language Support**: Translate between 100+ languages with automatic language detection
- **Real-Time Progress Tracking**: Watch your translations progress with detailed status updates
- **Batch File Processing**: Translate multiple subtitle files simultaneously
- **Smart Format Preservation**: Maintains original timing, line breaks, and subtitle structure perfectly
- **UTF-8 BOM Handling**: Automatic detection and removal of problematic byte order marks
- **Clean Output**: Advanced AI response cleaning ensures pure translated text without reasoning or thinking process
- **Configurable Settings**:
  - **Temperature Control**: Adjust translation creativity (0.0 - 2.0)
  - **Batch Size**: Customize processing batch size for optimal performance
  - **Model Selection**: Choose from multiple Gemini models
  - **Secondary API Key**: Quota management with automatic failover
- **Drag & Drop Support**: Simply drag subtitle files to start translating
- **Progress Logging**: Optional detailed logs of translation progress
- **Resume Capability**: Resume interrupted translations (planned feature)

### 🎯 AI Translation Features
- **Professional Quality**: Context-aware translations that understand dialogue and subtitle conventions
- **Format Support**: Works with all subtitle formats (SRT, ASS, SSA, VTT, etc.)
- **Cost Efficient**: Batch processing minimizes API calls and costs
- **Error Recovery**: Graceful handling of API errors with automatic fallback to original text
- **Real-Time Feedback**: See each batch complete with translation count and status
- **Output Control**: Choose custom output directory or save alongside source files

### 🔧 Technical Excellence
- **BOM Detection**: Automatic UTF-8 BOM removal prevents first subtitle entry loss
- **Separator-Based Batching**: Uses `---SUBTITLE_SEPARATOR---` markers for reliable parsing
- **System Instructions**: Prevents AI thinking output for clean translations
- **Response Validation**: Ensures translation count matches source entry count
- **Thread-Safe UI Updates**: Proper goroutine handling for smooth user experience

### 🚀 How to Use AI Translation
1. Open the **AI Translation** tab
2. Enter your Google Gemini API key (get one free at [Google AI Studio](https://aistudio.google.com/app/apikey))
3. Select source and target languages
4. Add subtitle files (single or batch)
5. Adjust advanced settings if desired
6. Click **Start Translation** and watch the magic happen!

**Perfect for content creators, translators, and subtitle enthusiasts who need professional-quality translations at scale!**

## What's New in v1.9.2

### 🎨 Track Sorting & Organization Enhancement
- **Comprehensive Sorting Options**: Sort tracks by default order, filename, language, codec, or track number
- **Multi-Level Sorting Logic**: Intelligent sorting with fallback criteria for consistent results
- **Enhanced Batch Processing**: Better organization when working with multiple MKV files
- **Filter + Sort Integration**: Combined filtering and sorting for powerful track management

## What's New in v1.8.5

### 🎨 Enhanced Visual Color Picker - Major UI Improvement!
- **Professional Color Selection**: Comprehensive visual color picker for ASS/SSA subtitle styling
- **Live Color Preview**: 30x25px color preview rectangle with real-time updates
- **Hex Code Display**: Monospace label showing current hex color code (#FFFFFF format)
- **Advanced Color Dialog**: Full-featured color picker with RGB sliders, alpha control, and preset colors
- **8 Preset Colors**: Quick selection buttons (Red, Green, Blue, Yellow, Magenta, Cyan, White, Black)
- **Custom Color Input**: Direct hex color code entry with validation
- **Real-time Synchronization**: All controls update simultaneously when any value changes

### 🔤 Smart Font Selection System
- **25+ Popular Fonts**: Dropdown with Arial, Helvetica, Times New Roman, Georgia, Verdana, and more
- **Cross-Platform Fonts**: Includes system fonts and modern web fonts (Open Sans, Roboto, Lato)
- **Custom Font Support**: "Custom..." option opens dialog for entering any font name
- **Dynamic Font List**: Custom fonts are added to dropdown for future use
- **Professional Typography**: Includes serif, sans-serif, monospace, and display fonts

### 🐛 Critical Color Format Fixes
- **ASS Format**: Correct RGB to BGR color conversion for proper color display
- **SSA Format**: Fixed color conversion to match ASS format (both use BGR)
- **Color Accuracy**: Red now displays as red, blue as blue (no more color swapping)
- **Format Compatibility**: Proper color handling for both ASS and SSA subtitle formats

### 🎯 User Experience Enhancements
- **Intuitive Interface**: Visual color selection replaces technical hex code typing
- **Professional Workflow**: Color picker → Preview → Apply workflow
- **Accessibility**: Both visual preview and text representation available
- **Better Organization**: Font and color options get dedicated UI space

## What's New in v1.8.0

### 🆕 Convert Subtitles Tab - Major New Feature!
- **Universal Subtitle Converter**: New dedicated tab for converting between subtitle formats
- **Supported Input Formats**: SRT, ASS, SSA, VTT, VobSub (.sub/.idx), PGS (.sup), TXT
- **Supported Output Formats**: SRT, ASS, SSA, VTT, SUB (MicroDVD), TXT
- **Smart Format Detection**: Automatic detection of subtitle format based on file extension and content analysis
- **Advanced Conversion Options**:
  - **Frame Rate Selection**: Custom frame rates for SUB format (23.976, 24, 25, 29.97, 30, 50, 59.94, 60 FPS)
  - **Time Offset Adjustment**: Shift all subtitle timing by specified seconds (+/- values supported)
  - **Text Processing**: Remove formatting tags, case conversion (UPPERCASE, lowercase, Title Case)
  - **ASS/SSA Styling**: Custom font family, size, color, margins, and style templates
- **Drag & Drop Support**: Full drag-and-drop functionality for all supported subtitle formats
- **Progress Tracking**: Real-time progress updates with detailed status information
- **Professional Output**: Format-specific optimizations for maximum compatibility

### 🔧 Technical Improvements
- **Enhanced UI Organization**: Organized conversion options into logical groups for better usability
- **Thread-Safe Operations**: Proper UI updates using fyne.Do() for goroutine safety
- **Comprehensive Error Handling**: Detailed error messages and validation for all conversion operations
- **Format Parsers**: Robust parsing engines for SRT, ASS/SSA, and VTT formats with proper timestamp handling

## What's New in v1.7.0

- **Enhanced Progress Reporting**: Added detailed time tracking with elapsed time and ETA calculations for both extraction and conversion processes
- **Improved Installation Scripts**: Fixed pip installation issues with the `--ignore-installed` flag to avoid uninstall errors with system packages
- **Progress Feedback**: Added intermediate progress updates during file processing with percentage completion
- **Utility Enhancements**: Added time formatting utilities to display human-readable duration information
- **Streamlined Dependencies**: Removed unnecessary dependency on Go installation for PGSRip

## What's New in v1.6.9b

- **Enhanced Log Output**: Redesigned the logging format for a more attractive and consistent user experience.
- **Styled Log Messages**: Implemented icons and a clear, structured format for info, success, errors, and conversion steps.
- **Improved Readability**: Log messages are now easier to read and understand at a glance.

## What's New in v1.6.9a

- **Improved Dependency Detection**: Enhanced detection logic for all external dependencies (ffmpeg, vobsub2srt, MKVMerge, MKVExtract, Deno, Tesseract, Go, PGStoSRT)
- **Packaged App Compatibility**: Fixed dependency detection in macOS app bundles by scanning multiple common installation paths
- **Debug Logging**: Added detailed debug logging to help diagnose dependency issues in packaged environments
- **Fixed Icon Format**: Corrected icon format for proper display in macOS dock/taskbar

## What's New in v1.6.5

- **Seasonal Dark Themes**: Added darker seasonal themes (Spring, Summer, Autumn, Winter) with improved contrast and readability
- **Resizable Windows**: Improved window resizing support for both the main application and theme customizer
- **Theme Persistence**: Theme preferences now persist between application restarts
- **Scrollable Content**: Added scroll containers to ensure all content is accessible regardless of window size
- **Progress Bar Improvements**: Fixed progress bar behavior to show incremental updates during subtitle extraction

## What's New in v1.6.2

- **Track Selection Controls**: Added "Select All" and "Deselect All" buttons for quick subtitle track management
- **Track Filtering**: New filter input field allowing real-time filtering of subtitle tracks by language, codec, name, or track number
- **Enhanced UI Layout**: Improved organization in the Insert Subtitles tab with better labeling and visual structure
- **Theme Support**: Application now supports multiple themes (Light, Dark, Blue, Warm, Green, and seasonal themes) with consistent readability across all UI elements
- **Improved Input Fields**: Wider input fields with helpful placeholders for better usability
- **Contextual Help Text**: Added helpful information text for output file naming options

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
