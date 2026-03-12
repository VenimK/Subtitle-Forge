# Subtitle Forge GUI V2.4.1
> Desktop subtitle extraction, conversion, translation, and transcription built with **Fyne**.
> [![GitHub Release](https://img.shields.io/github/v/release/VenimK/Subtitle-Forge)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![GitHub Release Date](https://img.shields.io/github/release-date/VenimK/Subtitle-Forge?style=flat)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub Downloads Latest](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/latest/total?style=flat&label=downloads%40latest&color=orange)](https://github.com/VenimK/Subtitle-Forge/releases/latest)

This directory contains the Fyne-based GUI application for Subtitle Forge. It is the desktop app most users should run when they want a visual workflow for extracting, converting, translating, and transcribing subtitles.

## What the GUI app does

- **Extract subtitles** from MKV files
- **Insert subtitles** into video containers
- **Convert subtitle formats** with optional OCR workflows
- **Translate subtitles** with GST/Gemini or LibreTranslate
- **Transcribe audio/video** with Whisper
- **Manage dependencies** from the Settings tab
- **Review logs** for troubleshooting

## Key GUI features

- **Extract Subtitles**
  - Load subtitle tracks from MKV files
  - Batch extraction support
  - OCR conversion when required tools are installed

- **Insert Subtitles**
  - Add subtitle files back into MKV containers

- **Convert Subtitles**
  - Convert between common subtitle formats
  - Apply timing and text-processing options

- **AI Translation**
  - Uses `gst` workflows for Gemini-backed translation

- **Whisper Transcription**
  - Supports local macOS Whisper setups and remote servers

- **LibreTranslate**
  - Supports local or remote translation servers

## Quick install notes

### macOS

1. Download the latest macOS release.
2. Unzip the app.
3. Remove quarantine attributes if needed:

```bash
xattr -cr "Subtitle Forge.app"
```

4. Launch the app.

### Windows

1. Download the latest Windows release.
2. Extract the archive.
3. Run `WindowsAIOInstaller.ps1` or `install_dependencies.ps1` if your package includes it.
4. Launch the app.

### Linux

1. Download the latest Linux release.
2. Extract the archive.
3. If needed, install dependencies from the included package scripts.
4. Launch the app.

For more complete end-user setup guidance, see the root [`README.md`](../README.md).

## Optional setup helpers

If you want easier setup for advanced workflows, use the included helper scripts from the repository root.

### General macOS dependency setup

```bash
bash ../install_dependencies.sh
```

This can install common dependencies and set up GST plus tessdata.

### Whisper on macOS

```bash
bash ../Whisper-mac-coreml.sh install
bash ../Whisper-mac-coreml.sh download-model base.en
```

Default app paths:

- `~/.whispercpp-coreml/whisper.cpp/build/bin/whisper-cli`
- `~/.whispercpp-coreml/whisper.cpp/models/ggml-base.en.bin`

### LibreTranslate on macOS

```bash
bash ../Libre-mac.sh install
bash ../Libre-mac.sh start
```

Default local install path:

- `~/.libretranslate`

## Build from source

### Prerequisites

- Go 1.18 or later
- Fyne build dependencies for your platform

Fyne setup guide:

- [Fyne Getting Started](https://developer.fyne.io/started/)

Typical platform examples:

- **macOS**
  - `brew install gcc`

- **Linux**
  - `apt install gcc libgl1-mesa-dev xorg-dev`

- **Windows**
  - install MinGW or MSYS2

### Build steps

```bash
chmod +x build.sh
./build.sh
```

Typical outputs:

- macOS: `./build/subtitle-forge-mac`
- Linux: `./build/subtitle-forge-linux`

## What’s new in V2.4.1

- Improved update detection for version comparisons
- Better installer/app path synchronization
- More accurate LibreTranslate readiness checks
- Better Whisper default path detection and persistence
- Cleaner Linux packaging behavior

For full project documentation and release history, see:

- [Main README](../README.md)
- [GitHub Releases](https://github.com/VenimK/Subtitle-Forge/releases)

## License

[MIT License](../LICENSE.md)
