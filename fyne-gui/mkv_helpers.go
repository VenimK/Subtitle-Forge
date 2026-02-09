package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SupportedSubtitleExts is the canonical list of supported subtitle file extensions
var SupportedSubtitleExts = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt"}

// SupportedSubtitleExtsWithIdx includes .idx for formats that use index files
var SupportedSubtitleExtsWithIdx = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".idx", ".txt"}

// IsSubtitleFile checks if a file path has a supported subtitle extension
func IsSubtitleFile(path string) bool {
	ext := strings.ToLower(path)
	for _, supported := range SupportedSubtitleExts {
		if strings.HasSuffix(ext, supported) {
			return true
		}
	}
	return false
}

// IsSubtitleFileWithIdx checks if a file path has a supported subtitle extension (including .idx)
func IsSubtitleFileWithIdx(path string) bool {
	ext := strings.ToLower(path)
	for _, supported := range SupportedSubtitleExtsWithIdx {
		if strings.HasSuffix(ext, supported) {
			return true
		}
	}
	return false
}

// CodecToExtension returns the file extension for a given subtitle codec string.
// If convertOCR is true, PGS codecs return "srt" instead of "sup".
func CodecToExtension(codec string, convertOCR bool) string {
	codecLower := strings.ToLower(codec)
	switch {
	case strings.Contains(codecLower, "subrip") || strings.Contains(codecLower, "srt"):
		return "srt"
	case strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv"):
		if convertOCR {
			return "srt"
		}
		return "sup"
	case strings.Contains(codecLower, "vobsub"):
		return "sub"
	case strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "advanced substation"):
		return "ass"
	case strings.Contains(codecLower, "ssa"):
		return "ssa"
	default:
		return "srt"
	}
}

// CodecToExtensionForExtract returns the file extension for extraction (single-file mode).
// VobSub returns "idx" and unknown codecs use a cleaned codec name as fallback.
func CodecToExtensionForExtract(codec string) string {
	codecLower := strings.ToLower(codec)
	switch {
	case strings.Contains(codecLower, "subrip") || strings.Contains(codecLower, "srt"):
		return "srt"
	case strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv"):
		return "sup"
	case strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "advanced substation"):
		return "ass"
	case strings.Contains(codecLower, "ssa"):
		return "ssa"
	case strings.Contains(codecLower, "vobsub"):
		return "idx"
	default:
		// Use lowercase codec name as fallback but remove any slashes
		cleanCodec := strings.ReplaceAll(codec, "/", "_")
		return strings.ToLower(cleanCodec)
	}
}

// IsConvertibleCodec returns true if the codec supports OCR/conversion to SRT
func IsConvertibleCodec(codec string) bool {
	codecLower := strings.ToLower(codec)
	return codec == "hdmv_pgs_subtitle" || codec == "HDMV PGS" ||
		strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "ssa") ||
		strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "sub station") ||
		codec == "vobsub" || codec == "VobSub"
}

// IsPGSCodec returns true if the codec is a PGS subtitle
func IsPGSCodec(codec string) bool {
	codecLower := strings.ToLower(codec)
	return strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv") ||
		codec == "hdmv_pgs_subtitle" || codec == "HDMV PGS"
}

// IsASSCodec returns true if the codec is ASS/SSA
func IsASSCodec(codec string) bool {
	codecLower := strings.ToLower(codec)
	return strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "ssa") ||
		strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "sub station")
}

// IsVobSubCodec returns true if the codec is VobSub
func IsVobSubCodec(codec string) bool {
	codecLower := strings.ToLower(codec)
	return strings.Contains(codecLower, "vobsub")
}

// NewMkvextractCmd creates an exec.Cmd for mkvextract, using the stored binary path if available
func NewMkvextractCmd(args ...string) *exec.Cmd {
	if mkvextractBinaryPath != "" {
		return exec.Command(mkvextractBinaryPath, args...)
	}
	return exec.Command("mkvextract", args...)
}

// NewMkvmergeCmd creates an exec.Cmd for mkvmerge, using the stored binary path if available
func NewMkvmergeCmd(args ...string) *exec.Cmd {
	if mkvmergeBinaryPath != "" {
		return exec.Command(mkvmergeBinaryPath, args...)
	}
	return exec.Command("mkvmerge", args...)
}

// CreateTrackRow creates a UI row for a TrackItem in the track list
func CreateTrackRow(t *TrackItem) *fyne.Container {
	var trackInfoText string
	if t.FilePath != "" {
		// Include filename for batch processing
		trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filepath.Base(t.FilePath))
	} else {
		// Single file mode
		trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
	}
	trackInfo := widget.NewLabel(trackInfoText)

	if t.ConvertOCR != nil {
		// For PGS/VobSub subtitles, show OCR option and language selection
		ocrLabel := widget.NewLabel("Convert to SRT")
		if t.LangSelect != nil {
			// Add language selection dropdown for OCR-based conversion
			langLabel := widget.NewLabel("OCR Language:")
			return container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
		}
		// For ASS/SSA conversion (no OCR language needed)
		return container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
	}
	// For other subtitle formats
	return container.NewHBox(t.Check, t.Status, trackInfo)
}
