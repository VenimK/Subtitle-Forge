# Subtitle Forge v1.6.9 Release Notes

## Dependency Detection and Packaging Improvements

This release focuses on improving the robustness of dependency detection and packaging for macOS:

### Dependency Detection Improvements
- Enhanced dependency detection for all external tools (mkvmerge, mkvextract, ffmpeg, vobsub2srt, etc.)
- Added support for detecting tools in non-standard installation paths
- Improved reliability when running in packaged/bundled environments
- Fixed runtime errors related to mkvmerge and mkvextract path handling
- Added debug logging for dependency detection to aid troubleshooting

### Packaging Improvements
- Added proper application icon for macOS
- Fixed icon format issues for proper display in dock/taskbar
- Improved packaging process using Fyne CLI

### Other Changes
- Updated all seasonal themes (Spring, Summer, Autumn, Winter) to dark color palettes
- Improved input background handling in CustomTheme for dark themes
- Updated documentation to reflect new features and improvements

## Installation

Download the appropriate package for your platform and follow the installation instructions in the README.md file.

## Known Issues

None at this time.

## Feedback

Please report any issues or suggestions on the GitHub issue tracker.
