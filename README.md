# Subtitle Forge v2.0.18
> Powerful subtitle extraction, conversion, and **AI translation** tool for **macOS**, **Linux**, and **Windows** 🔹 Extract, convert, translate, and manage subtitles with ease 🔹
> [![GitHub Release](https://img.shields.io/github/v/release/VenimK/Subtitle-Forge)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![GitHub Release Date](https://img.shields.io/github/release-date/VenimK/Subtitle-Forge?style=flat)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub All Releases](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/total.svg)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub Downloads Latest](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/latest/total?style=flat&label=downloads%40latest&color=orange)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![PayPal](https://img.shields.io/badge/Donate-PayPal-blue.svg)](https://paypal.me/VenimK)
> [![Discord](https://img.shields.io/badge/Discord-Join%20Server-5865F2?logo=discord&logoColor=white)](https://discord.gg/Sm3KEmUk)

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
3. Run the included `install_dependencies.ps1` script as administrator to install required dependencies
4. Launch the application

### Linux
1. Download the latest release for Linux (choose from two options):
   - **Standard Package**: `subtitle-forge-linux.tar.gz` - Includes installation scripts
   - **Self-Contained Bundle**: `subtitle-forge-linux-bundle.tar.gz` - Includes bundled dependencies
2. Extract the archive: `tar -xzf subtitle-forge-linux*.tar.gz`
3. **For Standard Package**: Install dependencies using the included script:
   ```bash
   sudo ./install_dependencies.sh
   ```
   **For Self-Contained Bundle**: Use the launcher script:
   ```bash
   ./subtitle-forge.sh
   ```
4. Make the binary executable: `chmod +x subtitle-forge-linux` (if needed)
5. Launch the application: `./subtitle-forge-linux`

#### What's Included in Linux Packages
- **Standard Package**: Binary + installation scripts for all dependencies
- **Self-Contained Bundle**: Binary + bundled FFmpeg, Deno, Tesseract, and MKVToolNix (when available)
- Installation scripts for missing dependencies: `install_vobsub2srt.sh`, `install_pgsrip.sh`, `install_tessdata.sh`
- Comprehensive installation guide (`INSTALL.md`)

## What's New in v2.0.17 ✨

### 🚀 Gemini 3.0 Support & GST v3.0.0!

**v2.0.17** adds support for the latest Gemini 3.0 models and upgrades GST:

- **🆕 Gemini 3.0 Models**: Support for `gemini-3-flash-preview` (Latest 3.0)
- **⚡ GST v3.0.0**: Upgraded gemini-srt-translator to latest version
- **🧠 Smart Thinking**: Auto-applies correct thinking settings per model:
  - Gemini 2.5: Uses `thinking_budget` (default: 2048)
  - Gemini 3.0: Uses `thinking_level` (default: low)
- **🔧 Advanced Controls**: Thinking Budget and Thinking Level in Advanced Settings
- **✨ Translation Spacing Fix**: Post-processing fixes spacing after punctuation

## What's New in v2.0.15 ✨

### 🔄 ASS to SRT Conversion Fix!

**v2.0.15** fixes the ASS to SRT conversion in the Convert subtitles tab:

- **🎯 Fixed Convert Tab**: ASS formatting tags now properly convert to SRT HTML tags
- **📝 Tag Support**: Converts `{\i1}` → `<i>`, `{\b1}` → `<b>`, `{\u1}` → `<u>`
- **🔤 Line Breaks**: Converts `\N` and `\n` to proper newlines
- **🧹 Clean Output**: Removes other ASS tags (position, alignment, color)
- **✅ Consistency**: Convert tab now works identically to Extract tab

## What's New in v2.0.13 ✨

### 📋 Dedicated Logs Tab - Better Debugging & Support!

**v2.0.13** introduces a comprehensive logging system to help diagnose issues:

- **📁 Persistent Log Files**: Automatically saved to `~/.subtitle-forge/logs/` with timestamped filenames
- **📋 New Logs Tab**: Real-time log viewer between AI Translate and Settings tabs
- **🔍 Full Command Logging**: All external tool executions (mkvextract, ffmpeg, pgsrip, etc.) logged with output
- **🛠️ Easy Debugging**: Copy Log Path, Open Log Folder, Copy All Logs buttons
- **📤 Share with Support**: Users can now easily share log files when reporting issues

Logged operations include:
- **Extract Subtitles**: Batch extraction, PGS/ASS/SSA/VobSub extraction, generic tracks
- **Insert Subtitles**: mkvmerge commands and results
- **Convert Subtitles**: Format conversion, PGS OCR, batch processing
- **AI Translate**: GST translation commands, progress, completion
- All errors with full context

## What's New in v2.0.12 ✨

### 🐍 GST Integration - Professional Terminal-Based Translation!

**v2.0.12** introduces full integration with `gst` (gemini-srt-translator), the powerful Python-based translation tool:

v2.0.12 Release Summary:

✅ vobsub2srt - GUI installation with graphical sudo prompt
✅ pgsrip - Complete installation and detection with Python fallback
✅ UX - Cleaner Extract tab with visible results
✅ Scripts - All installation scripts bundled in macOS app
✅ Build - macOS 15 Intel runner (deprecated macos-13 removed)

- **🖥️ Native Terminal Integration**: Opens macOS Terminal.app with beautiful, ANSI-colored interface
- **✨ Live Progress Display**: Watch real-time translation progress with animated spinners and progress bars
- **📊 Rich Status Information**: Displays subtitle count, duration timer, and completion statistics
- **🎯 Complete gst Support**: Full access to all gst features:
  - **Thinking Models**: Enable/disable AI thinking with configurable token budgets (0-24576)
  - **Resume Mode**: Auto-resume interrupted translations from where they left off
  - **Progress Logging**: Save detailed progress logs for debugging and monitoring
  - **Thoughts Logging**: Capture AI reasoning process for quality analysis
  - **Secondary API Keys**: Quota management with dual API key support
- **🎨 Beautiful Terminal UI**: Professional formatting with:
  - Box-drawing characters for structured display
  - File information header with input/output details
  - Real-time timer showing elapsed time
  - Interactive post-completion menu (open file, view logs)
- **🔒 Self-Cleaning Scripts**: Temporary files automatically cleaned up after execution

### 🎛️ Dual Translation Providers

Now with **two fully-functional AI translation providers**:

1. **Google Gemini AI** (Native Go Implementation)
   - Integrated batch translation in GUI
   - Smart auto-scroll results window
   - Temperature control and model selection
   - System language detection for default target language
   
2. **gst (Python)** (Terminal-Based)
   - Professional terminal interface with ANSI colors
   - All advanced gst features available
   - Perfect for users who prefer command-line experience

### 🔧 Code Quality Improvements

- **Cleaned Provider References**: Removed placeholders for unimplemented providers (OpenAI, DeepL, Azure)
- **Fixed Terminal Script Race Condition**: Temp scripts now self-delete after execution
- **Improved Error Handling**: Better error messages with detailed diagnostics
- **Accurate Documentation**: Comments and UI reflect actual implementation

**Perfect for professionals who demand both GUI convenience and terminal power!**

---

## What's New in v2.0.12 ✨

### 🚀 Professional-Grade AI Translation Improvements

**v2.0.12** brings enterprise-level translation quality matching the best Python translators:

- **🔓 Safety Filters Disabled**: All content categories set to `BLOCK_NONE` - no more translation refusals on mature content
- **📋 Structured JSON Schema**: Enforced request/response schema ensures consistent, reliable output
- **📝 Detailed Instruction Prompts**: Comprehensive prompts with field definitions, formatting rules, and context awareness
- **🎯 Temperature Control Enhanced**: New text entry field alongside slider for precise control (e.g., 0.2, 0.15)
- **⚡ Optimized Batch Size**: Default changed to 100 entries for better speed/quality balance
- **🎨 Better Defaults**: Temperature set to 0.3 for consistent, high-quality translations
- **📚 Gender-Aware Translation Ready**: Prompt templates prepared for future audio context support

### 🎯 Translation Quality Improvements
- **More Consistent Output**: Structured JSON prevents AI rambling and ensures exact format
- **Better Context Understanding**: Detailed instructions guide AI for subtitle-specific translations
- **No Content Blocking**: Translates movies/shows with any content rating
- **Easier Configuration**: Type exact temperature values instead of struggling with slider

**Perfect for professional translators who demand the highest quality! Matches or exceeds Python-based translators.**

---

## What's New in v2.0.1 ✨

### 🔥 Production Enhancements for AI Translation

**v2.0.1** brings essential UX improvements and production-ready refinements:

- **✨ Multiple File Drag-and-Drop**: Drag multiple subtitle files at once into AI Translate tab
- **💾 Persistent API Key Storage**: "Remember API Keys" checkbox saves keys securely
- **🪟 Fixed Window Resizing**: Window resizes freely in both width and height
- **⛔ Working Stop Button**: Functional translation cancellation with graceful shutdown
- **🧹 Production-Ready Code**: Removed debug statements for professional output
- **🔌 Provider Clarity**: Clear labeling of supported vs coming soon providers
- **💬 Improved Error Messages**: User-friendly validation with helpful links

---

## What's New in v2.0.0 🎉

### 🤖 AI Translation - Revolutionary New Feature!
**Transform your subtitle workflow with intelligent AI-powered translation using Google Gemini!**

- **Google Gemini AI Integration**: Professional-quality translations using Google's advanced Gemini 2.5 Flash model
- **Intelligent Batch Processing**: Translate 100+ subtitle entries per API call for maximum efficiency
- **Multi-Language Support**: Translate between 100+ languages with automatic language detection
- **Smart Language Selection**: Automatically detects system language and sets as default target language
- **Real-Time Progress Tracking**: Watch your translations progress with detailed batch-by-batch status updates
- **Batch File Processing**: Translate multiple subtitle files simultaneously with drag & drop support
- **Smart Format Preservation**: Maintains original timing, line breaks, and subtitle structure perfectly
- **UTF-8 BOM Handling**: Automatic detection and removal of problematic byte order marks (prevents first entry loss)
- **Clean Output**: Advanced AI response cleaning ensures pure translated text without reasoning artifacts

### 🎯 Key Translation Features
- **Professional Quality**: Context-aware translations that understand dialogue and subtitle conventions
- **Format Support**: Works seamlessly with all subtitle formats (SRT, ASS, SSA, VTT, etc.)
- **Cost Efficient**: Intelligent batching minimizes API calls and translation costs
- **Error Recovery**: Graceful handling of API errors with automatic fallback to original text
- **Configurable Settings**: Temperature control, batch size adjustment, model selection, and secondary API key support
- **Progress Logging**: Optional detailed logs of translation progress and AI responses

### 🚀 How to Use
1. Open the **AI Translation** tab in Subtitle Forge
2. Get your free API key from [Google AI Studio](https://aistudio.google.com/app/apikey)
3. Select source and target languages from 100+ supported languages
4. Add subtitle files (single file or batch processing)
5. Adjust advanced settings if desired (temperature, batch size, etc.)
6. Click **Start Translation** and watch the AI work its magic!

**Perfect for content creators, translators, and subtitle enthusiasts who need professional-quality translations at scale!**

---

## What's New in v1.9.2

### 🎯 Advanced Track Sorting - Professional Track Management!
- **Comprehensive Sorting Options**: Sort tracks by Filename, Language, Codec, or Track Number
- **Multi-Level Sorting Logic**: Smart sorting with fallback criteria for consistent results
- **Combined Filter + Sort**: Powerful track organization with filtering and sorting working together
- **Batch Processing Optimized**: Perfect for managing large numbers of tracks from multiple MKV files
- **Professional Workflow**: Organize tracks like professional video editing software

### 🔧 Smart Sorting Features
- **By Filename**: Groups tracks by source file, then by track number
- **By Language**: Groups by language (English, Spanish, etc.), then filename, then track number
- **By Codec**: Groups by subtitle format (SRT, ASS, PGS, etc.), then filename, then track number
- **By Track Number**: Sorts by track ID, then filename for duplicate track numbers
- **Default Order**: Original loading sequence from MKV files

### 🚀 Enhanced User Experience
- **Real-Time Sorting**: Changes apply immediately when selection changes
- **Organized Workflow**: Easy to find specific tracks in large batch operations
- **Flexible Organization**: Multiple sorting criteria for different use cases
- **Consistent Display**: Maintains filename display logic from v1.9.1 filtering enhancement
- **Memory Efficient**: Optimized sorting without modifying original track data

## What's New in v1.9.1

### 🔍 Enhanced Track Filtering - Batch Processing Improvement!
- **Filename Display in Filters**: Track filtering now shows filenames in batch mode for easy file identification
- **Consistent Track Information**: Filtered tracks display same detailed info as unfiltered tracks
- **Enhanced Filter Capabilities**: Can now filter by filename in addition to language, codec, name, and track number
- **Smart Display Logic**: Shows `[filename.mkv]` in batch mode, clean display in single file mode
- **Improved User Experience**: No more confusion about which file each track belongs to when filtering

### 🎯 Technical Enhancements
- **Conditional Filename Display**: Automatically detects batch vs single mode and adjusts display accordingly
- **Comprehensive Search**: Filter searches across all track properties including filename
- **Case-Insensitive Filtering**: All filtering works regardless of text case
- **Updated UI Text**: Filter placeholder reflects new filename filtering capability
- **Consistent Formatting**: Same track information format whether filtering or not

## What's New in v1.9.0

### 🚀 Batch Processing Revolution - Major Productivity Enhancement!
- **Batch Subtitle Extraction**: Process multiple MKV files simultaneously with intelligent batch processing
- **Batch Subtitle Conversion**: Convert multiple subtitle files at once with consistent settings
- **Smart File Management**: Add/remove files individually with visual file list display
- **Progress Tracking**: Real-time batch progress with "Processing file X/Y" status updates
- **Unified Workflow**: Same professional interface for both single and batch operations

### 📁 Enhanced File Selection & Management
- **Multiple File Selection**: "Select Multiple Files (Batch)" button for folder-based file selection
- **Drag & Drop Batch Support**: Drop multiple files to automatically enable batch mode
- **Visual File Lists**: Scrollable file lists showing format detection and individual remove buttons
- **Smart Mode Switching**: Seamlessly switch between single file and batch processing modes
- **Clear All Function**: One-click to clear all selected files and return to single file mode

### 🎬 Universal Format Support in Insert Subtitles
- **Multi-Format Insert**: Insert Subtitles tab now supports ALL subtitle formats (not just SRT)
- **Supported Formats**: SRT, ASS, SSA, VTT, SUB, SUP (PGS), TXT - complete format compatibility
- **Smart Format Detection**: Automatic format validation with helpful error messages
- **Enhanced UI**: Updated labels, buttons, and drop areas to reflect multi-format support
- **mkvmerge Integration**: Native format support through mkvmerge for quality preservation

### 🔄 Batch Conversion Features
- **Intelligent Processing**: Batch conversion with per-file format detection
- **Progress Visualization**: Progress bar showing current file and overall completion
- **Success Tracking**: "Batch conversion completed: X/Y files successfully converted" reporting
- **Error Resilience**: Continues processing even if individual files fail
- **Consistent Styling**: Apply same conversion settings to entire batch

### 🎯 User Experience Enhancements
- **Professional Workflow**: Batch processing rivals professional video editing software
- **Time Efficiency**: Process dozens of files with single click instead of one-by-one
- **Visual Feedback**: Clear file lists, progress indicators, and completion statistics
- **Flexible Operations**: Mix and match single file and batch operations as needed
- **Format Consistency**: Same comprehensive format support across all three tabs

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

## What's New in v1.7.3a

- **Complete Linux Packaging Overhaul**: Fixed all Linux packaging issues from v1.7.2
- **Self-Contained Linux Bundle**: New option to bundle dependencies directly with the application
- **Resolved Missing Installation Scripts**: All installation scripts now properly included in packages
- **Enhanced Build System**: Added `--bundle-deps` flag for creating self-contained packages
- **Two Linux Package Types**: Standard (with install scripts) and Bundle (with dependencies)
- **Better Installation Experience**: Comprehensive `INSTALL.md` guide included in all packages
- **Fixed Dependency Installation**: No more "file not found" errors when installing dependencies

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
- User-friendly graphical interface with four main tabs:
  - **Extract Subtitles**: Extract and convert subtitle tracks from MKV files with batch processing support
  - **Insert Subtitles**: Add external subtitle files (all formats) into MKV files
  - **Convert Subtitles**: Universal subtitle format converter with advanced options and batch processing
  - **Settings**: Dependency management and theme customization
- **Batch Processing**: Process multiple files simultaneously in Extract and Convert tabs
- **Universal Format Support**: All tabs support SRT, ASS, SSA, VTT, SUB, SUP (PGS), TXT formats
- Full drag and drop support in all tabs for easy file selection (single and multiple files)
- **Subtitle Format Conversion**:
  - Convert PGS/SUP subtitles to SRT format using OCR
  - Convert VobSub (.idx/.sub) subtitles to SRT format using OCR
  - Convert ASS/SSA subtitles to SRT format
  - Universal conversion between SRT, ASS, SSA, VTT, SUB (MicroDVD), and TXT formats
  - Advanced conversion options: frame rate selection, time offset, text processing, styling
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
