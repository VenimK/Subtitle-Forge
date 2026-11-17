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

			// Fill in the languages the containers left untagged before rendering,
			// so the list, the filter and the OCR defaults all see the result.
			detected := detectTrackLanguages("", *trackItems, func(done, total int) {
				fyne.Do(func() {
					progress.Max = float64(total)
					progress.SetValue(float64(done))
					progress.Show()
					currentTrackLabel.SetText(fmt.Sprintf("Detecting languages %d/%d", done, total))
				})
			})

			// Update UI with all tracks
			fyne.Do(func() {
				progress.Hide()
				summary := fmt.Sprintf("Found %d subtitle tracks across %d files", totalTracks, len(*mkvFiles))
				if detected > 0 {
					summary += fmt.Sprintf(" (%d languages detected)", detected)
				}
				currentTrackLabel.SetText(summary)

				// Update track list
				trackList.Objects = nil
				for _, tt := range *trackItems {
					// Show file name + track info
					fileName := filepath.Base(tt.FilePath)
					trackInfo := widget.NewLabel(fmt.Sprintf("%s - Track %d: %s (%s) %s", fileName, tt.Num, trackLangDisplay(tt), tt.Codec, tt.Name))

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
				refreshTrackList(trackList, trackListScroll)

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

	singlePath := *mkvPath
	isMP4 := IsMP4File(singlePath)

	// Container parsing and language sampling run off the UI thread, and rows are
	// rendered only afterwards so the list never shows a stale "und" that then
	// changes underneath the user.
	go func() {
		type subtitleMeta struct {
			id    int
			lang  string
			codec string
			name  string
		}
		var metas []subtitleMeta

		if isMP4 {
			// Use ffprobe for MP4/M4V files
			mp4Tracks, err := LoadMP4SubtitleTracks(singlePath)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(fmt.Errorf("Failed to analyze MP4 file: %v", err), w) })
				return
			}
			for _, mt := range mp4Tracks {
				metas = append(metas, subtitleMeta{id: mt.Index, lang: mt.Language, codec: mt.Codec, name: mt.TrackName})
			}
		} else {
			// Run mkvmerge to get track info (MKV files)
			AppLog("DEBUG", "Loading tracks using mkvmerge")
			output, err := NewMkvmergeCmd("-J", singlePath).Output()
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("Error running mkvmerge: %v (path: %s)", err, mkvmergeBinaryPath), w)
				})
				return
			}

			var mkvInfo map[string]interface{}
			if err := json.Unmarshal(output, &mkvInfo); err != nil {
				fyne.Do(func() { dialog.ShowError(fmt.Errorf("Error parsing mkvmerge output: %v", err), w) })
				return
			}

			tracks, ok := mkvInfo["tracks"].([]interface{})
			if !ok {
				fyne.Do(func() { dialog.ShowError(fmt.Errorf("No tracks found in video file."), w) })
				return
			}

			for _, track := range tracks {
				trackMap, ok := track.(map[string]interface{})
				if !ok {
					continue
				}

				// Check if this is a subtitle track
				if trackType, ok := trackMap["type"].(string); !ok || trackType != "subtitles" {
					continue
				}

				// Get track properties
				properties, ok := trackMap["properties"].(map[string]interface{})
				if !ok {
					continue
				}
				trackID, ok := trackMap["id"].(float64)
				if !ok {
					continue
				}

				lang := "und" // undefined language code
				if v, ok := properties["language"].(string); ok && v != "" {
					lang = v
				}
				codec, _ := trackMap["codec"].(string)
				name, _ := properties["track_name"].(string)

				metas = append(metas, subtitleMeta{id: int(trackID), lang: lang, codec: codec, name: name})
			}
		}

		items := make([]*TrackItem, 0, len(metas))
		for _, m := range metas {
			check := widget.NewCheck("", nil)
			check.SetChecked(true)
			items = append(items, &TrackItem{
				Num:    m.id,
				Lang:   m.lang,
				Codec:  m.codec,
				Name:   m.name,
				State:  "Pending",
				Check:  check,
				Status: widget.NewLabel("[ ]"),
			})
		}

		detected := detectTrackLanguages(singlePath, items, func(done, total int) {
			fyne.Do(func() {
				progress.Max = float64(total)
				progress.SetValue(float64(done))
				progress.Show()
				currentTrackLabel.SetText(fmt.Sprintf("Detecting languages %d/%d", done, total))
			})
		})

		fyne.Do(func() {
			progress.Hide()
			currentTrackLabel.Hide()

			*trackItems = items
			trackList.Objects = nil

			for _, t := range items {
				// OCR conversion offered for PGS/VobSub/ASS; MP4 keeps the narrower
				// PGS-only option it has always had.
				ocrCodec := t.Codec == "hdmv_pgs_subtitle"
				if !isMP4 {
					ocrCodec = IsPGSCodec(t.Codec) || IsVobSubCodec(t.Codec) ||
						strings.Contains(strings.ToLower(t.Codec), "ass") ||
						strings.Contains(strings.ToLower(t.Codec), "ssa") ||
						strings.Contains(strings.ToLower(t.Codec), "substation") ||
						strings.Contains(strings.ToLower(t.Codec), "sub station")
				}
				if ocrCodec {
					t.ConvertOCR = widget.NewCheck("", nil)
					t.ConvertOCR.SetChecked(true)
					if IsPGSCodec(t.Codec) || IsVobSubCodec(t.Codec) {
						t.LangSelect = widget.NewSelect(ocrLanguageOptions(t.Lang), nil)
						t.LangSelect.SetSelected("Auto (" + t.Lang + ")")
					}
				}

				codecLabel := t.Codec
				if isMP4 {
					codecLabel = MP4CodecDisplayName(t.Codec)
				}
				trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", t.Num, trackLangDisplay(t), codecLabel, t.Name))

				var row *fyne.Container
				switch {
				case t.ConvertOCR != nil && t.LangSelect != nil:
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR,
						widget.NewLabel("Convert to SRT"), widget.NewLabel("OCR Language:"), t.LangSelect)
				case t.ConvertOCR != nil:
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, widget.NewLabel("Convert to SRT"))
				case isPreviewableCodec(t.Codec):
					capturedTrack := t
					previewBtn := widget.NewButton("👁 Preview", func() {
						ShowSubtitlePreview(singlePath, capturedTrack, w)
					})
					previewBtn.Importance = widget.LowImportance
					row = container.NewHBox(t.Check, t.Status, trackInfo, previewBtn)
				default:
					row = container.NewHBox(t.Check, t.Status, trackInfo)
				}
				trackList.Add(row)
			}
			refreshTrackList(trackList, trackListScroll)

			switch {
			case len(items) == 0 && isMP4:
				result.SetText(setLogMessage(LogInfo, "No Subtitles", "No subtitle tracks found in this MP4 file."))
			case len(items) == 0:
				result.SetText(setLogMessage(LogInfo, "No Subtitles", "No subtitle tracks found in this file."))
			default:
				summary := fmt.Sprintf("Found %d subtitle tracks. Select the tracks you want to extract, then click 'Start Extraction'.", len(items))
				if detected > 0 {
					summary += fmt.Sprintf(" Detected the language of %d untagged track(s).", detected)
				}
				result.SetText(setLogMessage(LogSuccess, "Tracks Loaded", summary))
			}
		})
	}()
}

// ocrLanguageOptions builds the OCR language dropdown, offering the track's own
// language as the auto default.
func ocrLanguageOptions(lang string) []string {
	return []string{
		"Auto (" + lang + ")", // Auto option with the track's language
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
}
