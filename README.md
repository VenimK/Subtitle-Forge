# Subtitle Forge V2.4.2
> Powerful subtitle extraction, conversion, translation, and transcription for **macOS**, **Windows**, and **Linux**.
> [![GitHub Release](https://img.shields.io/github/v/release/VenimK/Subtitle-Forge)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![GitHub Release Date](https://img.shields.io/github/release-date/VenimK/Subtitle-Forge?style=flat)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub All Releases](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/total.svg)](https://github.com/VenimK/Subtitle-Forge/releases)
> [![GitHub Downloads Latest](https://img.shields.io/github/downloads/VenimK/Subtitle-Forge/latest/total?style=flat&label=downloads%40latest&color=orange)](https://github.com/VenimK/Subtitle-Forge/releases/latest)
> [![PayPal](https://img.shields.io/badge/Donate-PayPal-blue.svg)](https://paypal.me/VenimK)
> [![Discord](https://img.shields.io/badge/Discord-Join%20Server-5865F2?logo=discord&logoColor=white)](https://discord.gg/qpdZc7GcJP)

Subtitle Forge is a desktop application for working with subtitles across the full workflow: **extract**, **insert**, **convert**, **OCR**, **translate**, and **transcribe**. It includes a modern GUI, optional helper installers, and support for both local and remote subtitle-processing tools.

## Why use Subtitle Forge

- **Extract subtitles** from MKV files, including batch workflows
- **Insert subtitles** back into video containers
- **Convert subtitle formats** between SRT, ASS, SSA, VTT, TXT, SUB, SUP, and more
- **OCR image-based subtitles** such as PGS and VobSub
- **Translate subtitles** with Gemini/GST or LibreTranslate
- **Transcribe audio/video** with Whisper
- **Manage dependencies** from the built-in Settings tab
- **Work across platforms** with macOS, Windows, and Linux release packages

## Main features

### GUI application

- **Extract Subtitles**
  - Load subtitle tracks from MKV files
  - Batch process multiple files
  - Convert OCR-based tracks to text formats when tools are available

- **Insert Subtitles**
  - Add external subtitles to MKV files
  - Works with multiple subtitle formats

- **Convert Subtitles**
  - Convert between common subtitle formats
  - Apply time offsets and text-processing options
  - Handle OCR-driven conversions when the required tools are installed

- **AI Translation**
  - Use Gemini-based workflows through `gst`
  - Supports advanced translation settings and batch jobs

- **Whisper Transcription**
  - Transcribe media into subtitle files
  - Supports local `whisper.cpp` on macOS and remote API workflows

- **LibreTranslate**
  - Translate subtitle files with a local or remote LibreTranslate server

- **Settings and Logs**
  - Dependency readiness summary
  - Install actions for supported tools
  - Logs for troubleshooting
  - UI language selection

### CLI project

This repository also contains the original CLI extraction tooling. Most end users should use the packaged GUI releases, but you can still build and run the CLI from source if needed.

## Quick start

If you want the shortest path to a working setup:

- **Basic extraction / insertion / conversion**
  - Install the app for your platform
  - Make sure `ffmpeg`, `mkvmerge`, and `mkvextract` are available

- **OCR workflows**
  - Install `tesseract`
  - Install `deno`
  - Install the OCR helper tools shown in **Settings**

- **AI translation**
  - Install the GST/Gemini dependencies
  - Provide your API key in the app

- **Whisper transcription**
  - On macOS, run the included Whisper setup script or use a remote server

- **LibreTranslate**
  - Set a remote LibreTranslate URL, or install the local macOS helper and start the server

If you are unsure what is missing, open **Settings** in the app. Subtitle Forge shows a setup summary and dependency status there.

## Feature requirements

Use this table to understand what each workflow needs.

| Workflow | Required | Optional / Notes |
| --- | --- | --- |
| Extract subtitles | `mkvmerge`, `mkvextract` | `ffmpeg` is commonly useful |
| Insert subtitles | `mkvmerge` | |
| Convert text subtitles | none for basic formats | `ffmpeg` may be used depending on workflow |
| OCR PGS / SUP | `deno`, `tesseract`, tessdata | PGS-to-SRT helper script |
| OCR VobSub | `tesseract`, tessdata, `vobsub2srt` | |
| AI Translate (GST) | `gst` plus provider credentials | Gemini API key required for Gemini-based use |
| Whisper local | `whisper-cli` and model file | macOS helper script available |
| Whisper remote | Remote server URL | API key may be required |
| LibreTranslate remote | Server URL | API key optional depending on server |
| LibreTranslate local | Local LibreTranslate install | macOS helper script available |

## Installation

### macOS

1. Download the latest macOS release from the [Releases page](https://github.com/VenimK/Subtitle-Forge/releases/latest).
2. Unzip the archive.
3. Remove the macOS quarantine flag:

```bash
xattr -cr "Subtitle Forge.app"
```

4. Move the app to `/Applications` or another preferred location.
5. Launch the app.

### Windows

1. Download the latest Windows release.
2. Extract the archive.
3. If included in your package, run `install_dependencies.ps1` or `WindowsAIOInstaller.ps1` as needed.
4. Launch the application.

### Linux

1. Download the latest Linux release.
2. Choose one of the package types:
   - **Standard package**: includes installation scripts
   - **Bundle package**: includes bundled dependencies where available
3. Extract the archive:

```bash
tar -xzf subtitle-forge-linux*.tar.gz
```

4. For the standard package, install dependencies if needed:

```bash
sudo ./install_dependencies.sh
```

5. Launch the application.

## Optional setup helpers

Subtitle Forge can work with only the tools needed for your chosen workflow. These helper scripts make setup easier.

### General dependency installer on macOS

Run:

```bash
bash ./install_dependencies.sh
```

This installer:

- installs common Homebrew packages such as `ffmpeg`, `mkvtoolnix`, `deno`, `tesseract`, and Python
- creates a dedicated GST virtual environment at `~/.subtitle-forge/gst-venv`
- installs or updates `gst`
- downloads tessdata to `~/tessdata_best`
- adds PATH and `TESSDATA_PREFIX` lines to `~/.zshrc`

After installation:

```bash
gst --help
gst translate --help
```

If you want to remove the user-scoped items later:

```bash
bash ./uninstall_dependencies.sh
```

### Whisper on macOS

Run:

```bash
bash ./Whisper-mac-coreml.sh install
bash ./Whisper-mac-coreml.sh download-model base.en
```

Optional:

```bash
bash ./Whisper-mac-coreml.sh generate-coreml base.en
```

The default install locations used by the app are:

- `~/.whispercpp-coreml/whisper.cpp/build/bin/whisper-cli`
- `~/.whispercpp-coreml/whisper.cpp/models/ggml-base.en.bin`

### LibreTranslate on macOS

Run:

```bash
bash ./Libre-mac.sh install
bash ./Libre-mac.sh start
```

Useful commands:

```bash
bash ./Libre-mac.sh start-bg
bash ./Libre-mac.sh status
```

The local install is placed under:

- `~/.libretranslate`

## First run guide

After launching Subtitle Forge:

1. Open **Settings**
2. Review the dependency summary
3. Install only the tools you need for your workflow
4. Configure any required API keys or server URLs
5. Return to the relevant tab and start working

## API keys and external services

Some features need external services or credentials.

- **Gemini / GST**
  - Requires a Gemini API key for Gemini-backed translation workflows
  - You can create one from [Google AI Studio](https://aistudio.google.com/app/apikey)

- **Whisper remote**
  - Requires a remote server URL
  - Some servers may also require an API key

- **LibreTranslate remote**
  - Requires a server URL
  - Some servers may require an API key

For privacy-sensitive use cases, local Whisper and local LibreTranslate setups are available on macOS.

## OCR notes

OCR-based subtitle conversion depends on the quality of the source subtitles and the installed language data.

- **PGS / SUP**
  - Uses Deno-based tooling with Tesseract

- **VobSub**
  - Uses `vobsub2srt` with Tesseract

Better results generally require:

- correct OCR language selection
- matching tessdata files
- high-quality source subtitles

## Troubleshooting

- **The app says a dependency is missing**
  - Open **Settings** and review the setup summary
  - Install the missing tool or point the app to the correct path

- **OCR is not working**
  - Make sure `tesseract` is installed
  - Make sure tessdata is present
  - For PGS workflows, make sure `deno` is installed

- **Whisper local mode does not start**
  - Verify `whisper-cli` exists and is executable
  - Verify the selected model file exists and is readable

- **LibreTranslate local mode is not ready**
  - Make sure `Libre-mac.sh install` completed successfully
  - Start the local server
  - Confirm the app is using the expected local or remote URL

- **Update notifications do not appear**
  - Install the latest release
  - Older releases may contain outdated updater logic

- **macOS says the app is damaged or blocked**
  - Run:

```bash
xattr -cr "Subtitle Forge.app"
```

## Building from source

### Prerequisites

- Go 1.18 or later
- Platform dependencies required by Fyne
- Additional platform build tools as needed

Fyne setup guide:

- [Fyne Getting Started](https://developer.fyne.io/started/)

Typical examples:

- **macOS**
  - `brew install gcc`

- **Linux**
  - `apt install gcc libgl1-mesa-dev xorg-dev`

- **Windows**
  - install MinGW or MSYS2

### Build steps

```bash
git clone https://github.com/VenimK/Subtitle-Forge.git
cd Subtitle-Forge/fyne-gui
chmod +x build.sh
./build.sh
```

Typical outputs:

- macOS: `./build/subtitle-forge-mac`
- Linux: `./build/subtitle-forge-linux`

## What’s new in V2.4.2

- Improved update detection for version comparisons
- Better installer/app path synchronization
- More accurate LibreTranslate readiness checks
- Better Whisper default path detection and persistence
- Cleaner packaging behavior for Linux releases

For the full release history, see:

- [GitHub Releases](https://github.com/VenimK/Subtitle-Forge/releases)

## Contributing

Bug reports, feature requests, and pull requests are welcome.

## License

[MIT License](LICENSE.md)
