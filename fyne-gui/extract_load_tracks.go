package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// extractLoadTracks handles the "Load Tracks" button logic for the extract tab.
func extractLoadTracks(
	w fyne.Window,
	mkvPath *string,
	outDir *string,
	trackItems *[]*TrackItem,
	mkvFiles *[]string,
	batchMode *bool,
	selectedFile *widget.Label,
	result *widget.Label,
	progress *widget.ProgressBar,
	currentTrackLabel *widget.Label,
	trackList *fyne.Container,
	trackListScroll *container.Scroll,
) {
	if *batchMode {
		// In batch mode, load tracks from all files for user selection
		if len(*mkvFiles) == 0 {
			dialog.ShowError(fmt.Errorf("Please select video files for batch processing first."), w)
			return
		}

		// Load tracks from all video files
		go func() {
			fyne.Do(func() {
				result.SetText(setLogMessage(LogInfo, "Loading Batch Tracks", "Analyzing all video files for subtitle tracks..."))
				progress.Max = float64(len(*mkvFiles))
				progress.SetValue(0)
				progress.Show()
				currentTrackLabel.Show()
			})

			// Clear previous tracks
			*trackItems = []*TrackItem{}

			totalTracks := 0
			for fileIndex, videoFile := range *mkvFiles {
				fyne.Do(func() {
					currentTrackLabel.SetText(fmt.Sprintf("Analyzing file %d/%d: %s", fileIndex+1, len(*mkvFiles), filepath.Base(videoFile)))
					progress.SetValue(float64(fileIndex))
				})

				if IsMP4File(videoFile) {
					// Use ffprobe for MP4/M4V files
					mp4Tracks, err := LoadMP4SubtitleTracks(videoFile)
					if err != nil {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to analyze: %s", filepath.Base(videoFile)))
						})
						continue
					}

					for _, mt := range mp4Tracks {
						trackItem := &TrackItem{
							Num:      mt.Index,
							Lang:     mt.Language,
							Codec:    mt.Codec,
							Name:     mt.TrackName,
							FilePath: videoFile,
							Check:    widget.NewCheck("", nil),
							Status:   widget.NewLabel("Ready"),
						}

						if mt.Codec == "hdmv_pgs_subtitle" {
							trackItem.ConvertOCR = widget.NewCheck("", nil)
						}

						*trackItems = append(*trackItems, trackItem)
						totalTracks++
					}
				} else {
					// Use mkvmerge for MKV files
					cmd := NewMkvmergeCmd("-J", videoFile)

					output, err := cmd.Output()
					if err != nil {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to analyze: %s", filepath.Base(videoFile)))
						})
						continue
					}

					// Parse JSON output
					var mkvInfo struct {
						Tracks []struct {
							ID         int    `json:"id"`
							Type       string `json:"type"`
							Codec      string `json:"codec"`
							Properties struct {
								Language  string `json:"language"`
								TrackName string `json:"track_name"`
							} `json:"properties"`
						} `json:"tracks"`
					}

					err = json.Unmarshal(output, &mkvInfo)
					if err != nil {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to parse: %s", filepath.Base(videoFile)))
						})
						continue
					}

					// Add subtitle tracks to the list
					for _, track := range mkvInfo.Tracks {
						if track.Type == "subtitles" {
							lang := track.Properties.Language
							if lang == "" {
								lang = "und"
							}

							trackName := track.Properties.TrackName
							if trackName == "" {
								trackName = "Untitled"
							}

							trackItem := &TrackItem{
								Num:      track.ID,
								Lang:     lang,
								Codec:    track.Codec,
								Name:     trackName,
								FilePath: videoFile,
								Check:    widget.NewCheck("", nil),
								Status:   widget.NewLabel("Ready"),
							}

							if track.Codec == "hdmv_pgs_subtitle" {
								trackItem.ConvertOCR = widget.NewCheck("", nil)
							}

							*trackItems = append(*trackItems, trackItem)
							totalTracks++
						}
					}
				}
			}

			// Update UI with all tracks
			fyne.Do(func() {
				progress.SetValue(float64(len(*mkvFiles)))
				progress.Hide()
				currentTrackLabel.SetText(fmt.Sprintf("Found %d subtitle tracks across %d files", totalTracks, len(*mkvFiles)))

				// Update track list
				trackList.Objects = nil
				for _, tt := range *trackItems {
					// Show file name + track info
					fileName := filepath.Base(tt.FilePath)
					trackInfo := widget.NewLabel(fmt.Sprintf("%s - Track %d: %s (%s) %s", fileName, tt.Num, tt.Lang, tt.Codec, tt.Name))

					if tt.ConvertOCR != nil {
						// For PGS subtitles, show OCR option
						ocrLabel := widget.NewLabel("Convert to SRT")
						row := container.NewHBox(tt.Check, tt.Status, trackInfo, tt.ConvertOCR, ocrLabel)
						trackList.Add(row)
					} else {
						// For other subtitle formats
						row := container.NewHBox(tt.Check, tt.Status, trackInfo)
						trackList.Add(row)
					}
				}
				trackList.Refresh()

				result.SetText(setLogMessage(LogSuccess, "Batch Tracks Loaded", fmt.Sprintf("Found %d subtitle tracks across %d files. Select the tracks you want to extract, then click 'Start Extraction'.", totalTracks, len(*mkvFiles))))
			})
		}()
		return
	}

	// Single file mode
	if *mkvPath == "" {
		dialog.ShowError(fmt.Errorf("Please select or drag & drop a video file first."), w)
		return
	}

	// Detect container type and load tracks accordingly
	if IsMP4File(*mkvPath) {
		// Use ffprobe for MP4/M4V files
		mp4Tracks, err := LoadMP4SubtitleTracks(*mkvPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to analyze MP4 file: %v", err), w)
			return
		}

		*trackItems = []*TrackItem{}
		trackList.Objects = nil

		for _, mt := range mp4Tracks {
			check := widget.NewCheck("", nil)
			check.SetChecked(true)
			status := widget.NewLabel("[ ]")

			t := &TrackItem{
				Num:    mt.Index,
				Lang:   mt.Language,
				Codec:  mt.Codec,
				Name:   mt.TrackName,
				State:  "Pending",
				Check:  check,
				Status: status,
			}

			if mt.Codec == "hdmv_pgs_subtitle" {
				t.ConvertOCR = widget.NewCheck("", nil)
				t.ConvertOCR.SetChecked(true)
			}

			*trackItems = append(*trackItems, t)

			trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", mt.Index, mt.Language, MP4CodecDisplayName(mt.Codec), mt.TrackName))

			// Preview button for text-based codecs
			var previewBtn *widget.Button
			if isPreviewableCodec(mt.Codec) {
				capturedTrack := t
				capturedPath := *mkvPath
				previewBtn = widget.NewButton("👁 Preview", func() {
					ShowSubtitlePreview(capturedPath, capturedTrack, w)
				})
				previewBtn.Importance = widget.LowImportance
			}

			var row *fyne.Container
			if t.ConvertOCR != nil {
				ocrLabel := widget.NewLabel("Convert to SRT")
				row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel)
			} else if previewBtn != nil {
				row = container.NewHBox(check, status, trackInfo, previewBtn)
			} else {
				row = container.NewHBox(check, status, trackInfo)
			}

			trackList.Add(row)
		}
		trackList.Refresh()

		if len(mp4Tracks) == 0 {
			result.SetText(setLogMessage(LogInfo, "No Subtitles", "No subtitle tracks found in this MP4 file."))
		} else {
			result.SetText(setLogMessage(LogSuccess, "Tracks Loaded", fmt.Sprintf("Found %d subtitle tracks. Select the tracks you want to extract, then click 'Start Extraction'.", len(mp4Tracks))))
		}
		return
	}

	// Run mkvmerge to get track info (MKV files)
	cmd := NewMkvmergeCmd("-J", *mkvPath)
	AppLog("DEBUG", "Loading tracks using mkvmerge")

	output, err := cmd.Output()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Error running mkvmerge: %v (path: %s)", err, mkvmergeBinaryPath), w)
		return
	}

	// Parse JSON output
	var mkvInfo map[string]interface{}
	err = json.Unmarshal(output, &mkvInfo)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Error parsing mkvmerge output: %v", err), w)
		return
	}

	// Extract tracks
	tracks, ok := mkvInfo["tracks"].([]interface{})
	if !ok {
		dialog.ShowError(fmt.Errorf("No tracks found in video file."), w)
		return
	}

	// Clear previous tracks
	*trackItems = []*TrackItem{}
	trackList.Objects = nil

	// Process subtitle tracks
	for _, track := range tracks {
		trackMap, ok := track.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a subtitle track
		trackType, ok := trackMap["type"].(string)
		if !ok || trackType != "subtitles" {
			continue
		}

		// Get track properties
		properties, ok := trackMap["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		trackID := int(trackMap["id"].(float64))

		// Get language with nil check
		var trackLang string
		if properties != nil {
			if lang, ok := properties["language"].(string); ok {
				trackLang = lang
			} else {
				trackLang = "und" // undefined language code
			}
		} else {
			trackLang = "und" // undefined language code
		}

		trackCodec := trackMap["codec"].(string)

		// Get track name if available
		var trackName string
		if name, ok := properties["track_name"].(string); ok {
			trackName = name
		} else {
			trackName = ""
		}

		// Create UI elements for this track
		check := widget.NewCheck("", nil)
		check.SetChecked(true)
		status := widget.NewLabel("[ ]")

		// Create track item
		t := &TrackItem{
			Num:    trackID,
			Lang:   trackLang,
			Codec:  trackCodec,
			Name:   trackName,
			State:  "Pending",
			Check:  check,
			Status: status,
		}

		// Add OCR option for PGS subtitles, ASS/SSA subtitles, and VobSub subtitles
		if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" ||
			strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") ||
			strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station") ||
			t.Codec == "vobsub" || t.Codec == "VobSub" {
			t.ConvertOCR = widget.NewCheck("", nil)
			t.ConvertOCR.SetChecked(true)

			// Add language selection for OCR conversion
			if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" || t.Codec == "vobsub" || t.Codec == "VobSub" {
				// Create language options
				langOptions := []string{
					"Auto (" + t.Lang + ")", // Auto option with detected language
					"English (en)",
					"French (fr)",
					"German (de)",
					"Spanish (es)",
					"Italian (it)",
					"Portuguese (pt)",
					"Dutch (nl)",
					"Russian (ru)",
					"Japanese (ja)",
					"Chinese (zh)",
					"Korean (ko)",
					"Czech (cs)",
					"Polish (pl)",
					"Swedish (sv)",
					"Danish (da)",
					"Finnish (fi)",
					"Norwegian (no)",
					"Hungarian (hu)",
					"Greek (el)",
					"Turkish (tr)",
					"Arabic (ar)",
					"Hebrew (he)",
					"Thai (th)",
				}

				// Create language dropdown
				t.LangSelect = widget.NewSelect(langOptions, nil)
				t.LangSelect.SetSelected("Auto (" + t.Lang + ")")
			} else {
				t.LangSelect = nil
			}
		} else {
			t.ConvertOCR = nil
			t.LangSelect = nil
		}

		*trackItems = append(*trackItems, t)

		// Create row for this track
		trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", trackID, trackLang, trackCodec, trackName))

		// Preview button for text-based codecs
		var previewBtn *widget.Button
		if isPreviewableCodec(trackCodec) {
			capturedTrack := t
			capturedPath := *mkvPath
			previewBtn = widget.NewButton("👁 Preview", func() {
				ShowSubtitlePreview(capturedPath, capturedTrack, w)
			})
			previewBtn.Importance = widget.LowImportance
		}

		var row *fyne.Container
		if t.ConvertOCR != nil {
			// For PGS/VobSub subtitles, show OCR option and language selection
			ocrLabel := widget.NewLabel("Convert to SRT")

			if t.LangSelect != nil {
				// Add language selection dropdown for OCR-based conversion
				langLabel := widget.NewLabel("OCR Language:")
				row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
			} else {
				// For ASS/SSA conversion (no OCR language needed)
				row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel)
			}
		} else if previewBtn != nil {
			row = container.NewHBox(check, status, trackInfo, previewBtn)
		} else {
			// For other subtitle formats
			row = container.NewHBox(check, status, trackInfo)
		}

		trackList.Add(row)
	}
	trackList.Refresh()

	result.SetText(setLogMessage(LogSuccess, "Tracks Loaded", "Tracks loaded. Select the tracks you want to extract, then click 'Start Extraction'"))
}
