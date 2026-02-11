package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func createUtilitiesTab(result *widget.Label) *fyne.Container {
	// Create a new Label for utilities tab results
	utilitiesResult := widget.NewLabel(T("extract.results_placeholder"))
	utilitiesResult.Wrapping = fyne.TextWrapWord
	utilitiesResultScroll := container.NewScroll(utilitiesResult)
	utilitiesResultScroll.SetMinSize(fyne.NewSize(850, 200))

	// Create file selection widgets for MKV operations
	mkvFileLabel := widget.NewLabel(T("utilities.no_mkv"))
	selectMkvBtn := widget.NewButton(T("utilities.select_mkv"), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".mkv") {
				dialog.ShowInformation(T("common.error"), T("insert.invalid_video"), fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}

			mkvFileLabel.SetText(filePath)
			utilitiesResult.SetText(setLogMessage(LogInfo, "MKV File Selected", "Selected MKV file: "+filePath))
		}, fyne.CurrentApp().Driver().AllWindows()[0])
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mkv"}))
		fd.Show()
	})

	// Create file selection widgets for SRT operations
	srtFileLabel := widget.NewLabel("No SRT file selected")
	selectSrtBtn := widget.NewButton("Select SRT File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".srt") {
				dialog.ShowInformation("Invalid File", "Please select an SRT file", fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}

			srtFileLabel.SetText(filePath)
			utilitiesResult.SetText(setLogMessage(LogInfo, "SRT File Selected", "Selected SRT file: "+filePath))
		}, fyne.CurrentApp().Driver().AllWindows()[0])
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".srt"}))
		fd.Show()
	})

	// Create MKV utility operations
	mkvInfoBtn := widget.NewButton(T("utilities.mkv_info"), func() {
		mkvPath := mkvFileLabel.Text
		if mkvPath == T("utilities.no_mkv") {
			dialog.ShowInformation(T("common.error"), T("utilities.no_mkv"), fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		utilitiesResult.SetText(setLogMessage(LogInfo, "Getting MKV Information", "Getting MKV information...\n"))

		// Run mkvinfo command
		go func() {
			var cmd *exec.Cmd
			// Since mkvinfo is part of the same package as mkvextract and mkvmerge,
			// we can derive its path from the mkvextract path if available
			if mkvextractBinaryPath != "" {
				// Get the directory of mkvextract and use it for mkvinfo
				mkvToolsDir := filepath.Dir(mkvextractBinaryPath)
				mkvinfoPath := filepath.Join(mkvToolsDir, "mkvinfo")

				// Check if mkvinfo exists at the expected path
				if _, err := os.Stat(mkvinfoPath); err == nil {
					cmd = exec.Command(mkvinfoPath, mkvPath)
					fmt.Println("[DEBUG] Using derived mkvinfo path:", mkvinfoPath)
				} else {
					// Fallback to PATH lookup
					cmd = exec.Command("mkvinfo", mkvPath)
					fmt.Println("[DEBUG] Could not find mkvinfo at derived path, using default PATH lookup")
				}
			} else {
				// Fallback to PATH lookup
				cmd = exec.Command("mkvinfo", mkvPath)
				fmt.Println("[DEBUG] No stored mkvextract path to derive mkvinfo path from, using default PATH lookup")
			}
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText(setLogMessage(LogSuccess, "MKV Information", "MKV Information for: "+mkvPath+"\n\n"+string(output)))
			})
		}()
	})

	mkvExtractChaptersBtn := widget.NewButton(T("utilities.extract_chapters"), func() {
		mkvPath := mkvFileLabel.Text
		if mkvPath == T("utilities.no_mkv") {
			dialog.ShowInformation(T("common.error"), T("utilities.no_mkv"), fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		// Get output directory (same as MKV file)
		dir := filepath.Dir(mkvPath)
		baseName := filepath.Base(mkvPath)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		outputPath := filepath.Join(dir, baseName+"_chapters.txt")

		utilitiesResult.SetText(setLogMessage(LogInfo, "Extracting Chapters", "Extracting chapters to: "+outputPath+"\n"))

		// Run mkvextract command for chapters
		go func() {
			cmd := NewMkvextractCmd(mkvPath, "chapters", outputPath)
			AppLog("DEBUG", "Chapters extraction using mkvextract")
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText(setLogMessage(LogSuccess, "Chapters Extracted", utilitiesResult.Text+"\nChapters extracted successfully to: "+outputPath+"\n"+string(output)))
			})
		}()
	})

	// Create SRT utility operations
	srtFixEncodingBtn := widget.NewButton("Fix SRT Encoding", func() {
		srtPath := srtFileLabel.Text
		if srtPath == "No SRT file selected" {
			dialog.ShowInformation("No File Selected", "Please select an SRT file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		utilitiesResult.SetText(setLogMessage(LogInfo, "Fixing SRT Encoding", "Fixing SRT encoding...\n"))

		// Run iconv command to fix encoding
		go func() {
			// Create a backup of the original file
			backupPath := srtPath + ".bak"
			if err := copyFile(srtPath, backupPath); err != nil {
				fyne.Do(func() {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError creating backup: " + err.Error())
				})
				return
			}

			// Try to detect and convert encoding to UTF-8
			cmd := exec.Command("iconv", "-f", "ISO-8859-1", "-t", "UTF-8", srtPath, "-o", srtPath+".tmp")
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				// Replace original with converted file
				if err := os.Rename(srtPath+".tmp", srtPath); err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError replacing file: " + err.Error())
					return
				}

				utilitiesResult.SetText(setLogMessage(LogSuccess, "SRT Encoding Fixed", utilitiesResult.Text+"\nSRT encoding fixed successfully.\nOriginal backup saved to: "+backupPath+"\n"+string(output)))
			})
		}()
	})

	srtFixTimingBtn := widget.NewButton("Fix SRT Timing", func() {
		srtPath := srtFileLabel.Text
		if srtPath == "No SRT file selected" {
			dialog.ShowInformation("No File Selected", "Please select an SRT file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		// Show dialog to get timing offset
		offsetEntry := widget.NewEntry()
		offsetEntry.SetPlaceHolder("e.g., +1.5 or -2.3 (seconds)")

		dialog.ShowCustomConfirm("Adjust SRT Timing", "Apply", "Cancel",
			container.NewVBox(
				widget.NewLabel("Enter timing offset in seconds:"),
				offsetEntry,
			),
			func(confirmed bool) {
				if !confirmed || offsetEntry.Text == "" {
					return
				}

				offset := offsetEntry.Text
				utilitiesResult.SetText(setLogMessage(LogInfo, "Adjusting SRT Timing", "Adjusting SRT timing with offset: "+offset+" seconds...\n"))

				go func() {
					// Create a backup of the original file
					backupPath := srtPath + ".bak"
					if err := copyFile(srtPath, backupPath); err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError creating backup: " + err.Error())
						})
						return
					}

					// Read the SRT file
					content, err := os.ReadFile(srtPath)
					if err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError reading SRT file: " + err.Error())
						})
						return
					}

					// Parse offset
					offsetFloat, err := strconv.ParseFloat(offset, 64)
					if err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nInvalid offset format: " + err.Error())
						})
						return
					}

					// Apply offset to timing
					adjustedContent := adjustSRTTiming(string(content), offsetFloat)

					// Write back to file
					if err := os.WriteFile(srtPath, []byte(adjustedContent), 0644); err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError writing adjusted SRT file: " + err.Error())
						})
						return
					}

					fyne.Do(func() {
						utilitiesResult.SetText(setLogMessage(LogSuccess, "SRT Timing Adjusted", utilitiesResult.Text+"\nSRT timing adjusted successfully.\nOriginal backup saved to: "+backupPath))
					})
				}()
			},
			fyne.CurrentApp().Driver().AllWindows()[0],
		)
	})

	// Create layout for the Utilities tab
	mkvSection := container.NewVBox(
		widget.NewLabelWithStyle(T("utilities.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(selectMkvBtn, mkvFileLabel),
		container.NewHBox(mkvInfoBtn, mkvExtractChaptersBtn),
	)

	srtSection := container.NewVBox(
		widget.NewLabelWithStyle("SRT Utilities", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(selectSrtBtn, srtFileLabel),
		container.NewHBox(srtFixEncodingBtn, srtFixTimingBtn),
	)

	utilitiesTabContent := container.NewVBox(
		mkvSection,
		widget.NewSeparator(),
		srtSection,
		widget.NewSeparator(),
		widget.NewLabel(T("utilities.results")),
		utilitiesResultScroll,
	)

	return utilitiesTabContent
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// loadTracksForFile loads tracks for a specific MKV file (for batch processing)
func loadTracksForFile(mkvPath string) bool {
	cmd := NewMkvmergeCmd("-J", mkvPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Parse JSON output (simplified - just check if we have tracks)
	return len(output) > 0 && strings.Contains(string(output), "tracks")
}

// extractAllSubtitleTracks extracts all subtitle tracks from an MKV file (for batch processing)
func extractAllSubtitleTracks(mkvPath, outDir string) bool {
	// Get track info first
	cmd := NewMkvmergeCmd("-J", mkvPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Parse JSON to find subtitle tracks
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
		return false
	}

	// Extract subtitle tracks
	mkvBaseName := filepath.Base(mkvPath)
	mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))

	successCount := 0
	for _, track := range mkvInfo.Tracks {
		if track.Type == "subtitles" {
			// Determine file extension based on codec
			ext := CodecToExtension(track.Codec, false)

			lang := track.Properties.Language
			if lang == "" {
				lang = "und"
			}

			outFile := fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, track.ID, lang, ext)

			// Extract the track
			extractCmd := NewMkvextractCmd("tracks", mkvPath, fmt.Sprintf("%d:%s", track.ID, outFile))
			extractCmd.Dir = outDir

			_, err := extractCmd.CombinedOutput()
			if err == nil {
				successCount++
			}
		}
	}

	return successCount > 0
}

// Helper function to adjust SRT timing

func adjustSRTTiming(content string, offsetSeconds float64) string {
	lines := strings.Split(content, "\n")
	result := []string{}

	// Regular expression to match SRT timestamp format: 00:00:00,000 --> 00:00:00,000
	re := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3}) --> (\d{2}):(\d{2}):(\d{2}),(\d{3})`)

	for _, line := range lines {
		// Check if the line contains timestamps
		if re.MatchString(line) {
			// Apply offset to both start and end timestamps
			adjustedLine := re.ReplaceAllStringFunc(line, func(match string) string {
				parts := re.FindStringSubmatch(match)
				if len(parts) != 9 {
					return match
				}

				// Parse start time
				startHour, _ := strconv.Atoi(parts[1])
				startMin, _ := strconv.Atoi(parts[2])
				startSec, _ := strconv.Atoi(parts[3])
				startMs, _ := strconv.Atoi(parts[4])

				// Parse end time
				endHour, _ := strconv.Atoi(parts[5])
				endMin, _ := strconv.Atoi(parts[6])
				endSec, _ := strconv.Atoi(parts[7])
				endMs, _ := strconv.Atoi(parts[8])

				// Convert to milliseconds and apply offset
				startTimeMs := startHour*3600000 + startMin*60000 + startSec*1000 + startMs
				endTimeMs := endHour*3600000 + endMin*60000 + endSec*1000 + endMs

				offsetMs := int(offsetSeconds * 1000)
				startTimeMs += offsetMs
				endTimeMs += offsetMs

				// Ensure times don't go negative
				if startTimeMs < 0 {
					startTimeMs = 0
				}
				if endTimeMs < 0 {
					endTimeMs = 0
				}

				// Convert back to SRT format
				startHour = startTimeMs / 3600000
				startTimeMs %= 3600000
				startMin = startTimeMs / 60000
				startTimeMs %= 60000
				startSec = startTimeMs / 1000
				startMs = startTimeMs % 1000

				endHour = endTimeMs / 3600000
				endTimeMs %= 3600000
				endMin = endTimeMs / 60000
				endTimeMs %= 60000
				endSec = endTimeMs / 1000
				endMs = endTimeMs % 1000

				return fmt.Sprintf("%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d",
					startHour, startMin, startSec, startMs,
					endHour, endMin, endSec, endMs)
			})

			result = append(result, adjustedLine)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// Log types for styling the output
const (
	LogInfo    = "INFO"
	LogSuccess = "SUCCESS"
	LogError   = "ERROR"
	LogExtract = "EXTRACT"
	LogConvert = "CONVERT"
)
