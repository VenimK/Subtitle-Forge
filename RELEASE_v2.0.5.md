# Release Notes - Subtitle Forge v2.0.5

## 🐛 Bug Fixes

### GST Path Persistence Fixed
**Issue:** Users reported that custom GST (Gemini Subtitle Translator) paths were not being saved between app sessions. When users changed the path from the default and restarted the app, their custom path was lost and reset to the default `findGSTPath()` value.

**Fix:** Implemented proper persistence for the GST path setting using Fyne's preferences system:
- GST path is now loaded from saved preferences on app startup
- Path is automatically saved whenever the user changes it
- Path is also saved when starting a translation (belt-and-suspenders approach)
- Follows the same pattern used for API key persistence

**Impact:** Users can now set their custom GST binary path once and it will be remembered across app sessions, improving workflow for users with custom GST installations.

---

## 📝 Technical Details

### Modified Files
- `fyne-gui/ai_translation.go`: Added GST path persistence logic
  - Load saved path on startup with fallback to `findGSTPath()`
  - Added `OnChanged` handler to save path immediately when modified
  - Added safety save when starting translation

### Code Changes
```go
// Load saved GST path from preferences, fallback to findGSTPath()
savedGSTPath := a.Preferences().StringWithFallback("ai_gst_path", "")
if savedGSTPath != "" {
    gstPathEntry.SetText(savedGSTPath)
} else {
    gstPathEntry.SetText(findGSTPath())
}

// Save GST path when it changes
gstPathEntry.OnChanged = func(newPath string) {
    a.Preferences().SetString("ai_gst_path", newPath)
}
```

---

## 🔄 Previous Features (Maintained)
- AI-powered subtitle translation with GST
- Multiple subtitle format support (SRT, ASS, VTT, etc.)
- Batch translation capability
- PGS to SRT OCR conversion
- MKV subtitle extraction and remuxing
- Custom themes and UI preferences
- Auto-update checker
- Cross-platform support (macOS, Linux, Windows)

---

## 📦 Installation

Download the appropriate package for your platform:
- **macOS**: `Subtitle-Forge-macOS.zip`
- **Linux**: `subtitle-forge-linux.tar.gz`
- **Windows**: `Subtitle-Forge-Windows.zip`

For detailed installation instructions, see the README.md file.

---

## 🙏 Acknowledgments

Thank you to the users who reported this issue and helped test the fix!

---

**Full Changelog**: https://github.com/VenimK/Subtitle-Forge/compare/v2.0.2...v2.0.5
