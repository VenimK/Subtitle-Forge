package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// maxPreviewLines is the number of subtitle lines to show in the preview.
const maxPreviewLines = 20

// isPreviewableCodec returns true if the codec produces text-based subtitles
// that can be previewed (not bitmap-based like PGS or VobSub).
func isPreviewableCodec(codec string) bool {
	cl := strings.ToLower(codec)
	switch {
	case strings.Contains(cl, "pgs"), strings.Contains(cl, "hdmv"):
		return false
	case strings.Contains(cl, "vobsub"), cl == "dvd_subtitle", cl == "dvdsub":
		return false
	default:
		return true
	}
}

// PreviewSubtitleTrack extracts a subtitle track to a temp file and returns
// the first maxPreviewLines lines of text. Works for both MKV and MP4 files.
func PreviewSubtitleTrack(videoPath string, trackIndex int, codec string) (string, error) {
	// Determine output extension
	var ext string
	if IsMP4File(videoPath) {
		ext = MP4CodecToExtension(codec)
	} else {
		ext = CodecToExtensionForExtract(codec)
	}

	// Create temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("sf_preview_%d.%s", trackIndex, ext))
	defer os.Remove(tmpFile)

	// Extract to temp file
	if IsMP4File(videoPath) {
		_, err := ExtractMP4Subtitle(videoPath, trackIndex, tmpFile)
		if err != nil {
			return "", fmt.Errorf("failed to extract subtitle for preview: %w", err)
		}
	} else {
		// MKV: use mkvextract
		cmd := NewMkvextractCmd(videoPath, "tracks", fmt.Sprintf("%d:%s", trackIndex, tmpFile))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to extract subtitle for preview: %s\n%s", err, string(output))
		}
	}

	// Read first N lines from the extracted file
	f, err := os.Open(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to read preview file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < maxPreviewLines {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return T("preview.empty"), nil
	}

	preview := strings.Join(lines, "\n")
	if len(lines) == maxPreviewLines {
		preview += "\n\n... (truncated)"
	}
	return preview, nil
}

// ShowSubtitlePreview shows a dialog with a preview of the subtitle track content.
func ShowSubtitlePreview(videoPath string, track *TrackItem, w fyne.Window) {
	if !isPreviewableCodec(track.Codec) {
		dialog.ShowInformation(T("preview.not_available"),
			T("preview.not_available_msg"),
			w)
		return
	}

	// Show loading dialog
	loadingLabel := widget.NewLabel(T("preview.extracting"))
	loadingDialog := dialog.NewCustomWithoutButtons(T("preview.loading"), loadingLabel, w)
	loadingDialog.Show()

	go func() {
		preview, err := PreviewSubtitleTrack(videoPath, track.Num, track.Codec)

		fyne.Do(func() {
			loadingDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("%s", Tf("preview.failed", err)), w)
				return
			}

			// Build the preview dialog
			title := fmt.Sprintf("Track %d: %s (%s) %s", track.Num, track.Lang, track.Codec, track.Name)
			titleLabel := widget.NewLabel(title)
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}

			previewText := widget.NewLabel(preview)
			previewText.Wrapping = fyne.TextWrapWord
			previewText.TextStyle = fyne.TextStyle{Monospace: true}

			scroll := container.NewScroll(previewText)
			scroll.SetMinSize(fyne.NewSize(600, 350))

			content := container.NewBorder(titleLabel, nil, nil, nil, scroll)

			d := dialog.NewCustom(T("preview.title"), T("common.close"), content, w)
			d.Resize(fyne.NewSize(650, 450))
			d.Show()
		})
	}()
}
