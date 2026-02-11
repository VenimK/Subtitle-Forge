package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// extractAssTrack handles ASS/SSA to SRT conversion extraction
func extractAssTrack(
	w fyne.Window,
	t *TrackItem,
	mkvPath *string,
	outDir *string,
	mkvBaseName string,
	result *widget.Label,
	progress *widget.ProgressBar,
	currentTrackLabel *widget.Label,
	trackList *fyne.Container,
	output *[]byte,
	err *error,
) {
	// ASS/SSA to SRT conversion
	fyne.Do(func() {
		result.SetText(setLogMessage(LogConvert, "ASS/SSA to SRT Conversion", "Starting ASS/SSA to SRT conversion process..."))
	})
	tempAssFile := fmt.Sprintf("%s.track%d_%s.ass", mkvBaseName, t.Num, t.Lang)
	outFile := fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

	// Get absolute paths for extraction
	absAssPath := filepath.Join(*outDir, tempAssFile)

	// Debug output
	fyne.Do(func() {
		currentTrackLabel.SetText(fmt.Sprintf("Extracting ASS/SSA track %d...", t.Num))
		result.SetText(result.Text + "\n\n=== ASS/SSA Extraction ===\n")
		result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
		result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", *outDir))
		result.SetText(result.Text + fmt.Sprintf("ASS/SSA file: %s\n", tempAssFile))
		result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absAssPath))
	})

	// Extract ASS/SSA first
	var cmd *exec.Cmd
	if IsMP4File(*mkvPath) {
		fyne.Do(func() {
			result.SetText(result.Text + "\nExtracting ASS/SSA track using ffmpeg...")
		})
		*output, *err = ExtractMP4Subtitle(*mkvPath, t.Num, absAssPath)
	} else {
		cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", *mkvPath, t.Num, tempAssFile)
		fyne.Do(func() {
			result.SetText(result.Text + "\nRunning: " + cmdStr)
		})

		cmd = NewMkvextractCmd("tracks", *mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
		AppLog("DEBUG", "ASS track extraction using mkvextract")
		cmd.Dir = *outDir

		AppLog("EXTRACT", "ASS/SSA extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(*mkvPath))
		*output, *err = cmd.CombinedOutput()
		AppLogCmd(cmd, *output, *err)
	}

	// Debug output - show command result
	fyne.Do(func() {
		result.SetText(result.Text + "\nCommand output: " + string(*output))
		if *err != nil {
			result.SetText(result.Text + "\nError: " + (*err).Error())
		}
	})

	// Check if the file was created and has content
	assFilePath := filepath.Join(*outDir, tempAssFile)
	fileInfo, statErr := os.Stat(assFilePath)
	if statErr != nil {
		fyne.Do(func() {
			result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
		})
		*err = statErr
	} else if fileInfo.Size() == 0 {
		fyne.Do(func() {
			result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
		})
		*err = fmt.Errorf("extracted file is empty (0 bytes)")
	} else {
		fyne.Do(func() {
			result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted ASS/SSA file (%d bytes)", fileInfo.Size()))
		})
	}

	if *err == nil {
		// Create a progress bar for the conversion process
		conversionProgress := widget.NewProgressBar()
		conversionProgress.Min = 0
		conversionProgress.Max = 100
		conversionProgress.SetValue(0)

		conversionLabel := widget.NewLabel("Converting ASS/SSA to SRT...")
		statusLabel := widget.NewLabel("Processing ASS/SSA file...")
		elapsedLabel := widget.NewLabel("Elapsed: 0s")
		remainingLabel := widget.NewLabel("Converting...")

		// Track conversion start time
		conversionStartTime := time.Now()

		// Create a ticker to update elapsed time
		ticker := time.NewTicker(500 * time.Millisecond)
		go func() {
			defer ticker.Stop()
			var lastElapsedText string

			for range ticker.C {
				elapsed := time.Since(conversionStartTime).Round(time.Second)
				newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

				// Only update UI if text has changed
				if newElapsedText != lastElapsedText {
					lastElapsedText = newElapsedText
					fyne.Do(func() {
						elapsedLabel.SetText(newElapsedText)
						conversionProgress.SetValue(50) // Simple indeterminate progress
					})
				}
			}
		}()

		fyne.Do(func() {
			result.SetText(result.Text + "\n\n[DEBUG] ASS/SSA extraction completed successfully, starting conversion process")

			// Show the conversion progress bar and labels
			currentTrackLabel.SetText("Converting ASS/SSA to SRT...")
			progress.Hide()
			trackList.Add(container.NewVBox(
				conversionLabel,
				statusLabel,
				conversionProgress,
				container.NewHBox(
					elapsedLabel,
					widget.NewLabel("|"),
					remainingLabel,
				),
			))
			trackList.Refresh()
		})

		// Get absolute paths for input and output
		absInputPath := filepath.Join(*outDir, tempAssFile)
		absOutputPath := filepath.Join(*outDir, outFile)

		// Use ffmpeg to convert ASS/SSA to SRT
		fyne.Do(func() {
			result.SetText(result.Text + "\n\n[DEBUG] Using ffmpeg to convert ASS/SSA to SRT")
			statusLabel.SetText("Running ffmpeg conversion...")
		})

		// Get ffmpeg path - prioritize Homebrew version
		ffmpegPath := "ffmpeg" // Default fallback path

		// First check Homebrew path (preferred)
		homebrewPath := "/opt/homebrew/bin/ffmpeg"
		if _, ffErr := os.Stat(homebrewPath); ffErr == nil {
			ffmpegPath = homebrewPath
			fyne.Do(func() {
				result.SetText(result.Text + "\n[DEBUG] Using Homebrew ffmpeg: " + homebrewPath)
			})
		} else {
			// If Homebrew not found, check Miniconda as fallback
			homeDir, hdErr := os.UserHomeDir()
			if hdErr == nil {
				minicondaPath := filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg")
				if _, mcErr := os.Stat(minicondaPath); mcErr == nil {
					ffmpegPath = minicondaPath
					fyne.Do(func() {
						result.SetText(result.Text + "\n[DEBUG] Using Miniconda ffmpeg: " + minicondaPath)
					})
				}
			}
		}

		// Create the ffmpeg command with the appropriate path
		cmd = exec.Command(ffmpegPath, "-i", absInputPath, "-f", "srt", absOutputPath)
		cmd.Dir = *outDir

		AppLog("CONVERT", "ASS/SSA to SRT conversion: %s -> %s", filepath.Base(absInputPath), filepath.Base(absOutputPath))
		// Run the command and capture output
		*output, *err = cmd.CombinedOutput()
		AppLogCmd(cmd, *output, *err)

		// Stop the ticker
		ticker.Stop()

		// Update UI with results
		fyne.Do(func() {
			result.SetText(result.Text + "\nffmpeg output: " + string(*output))

			if *err != nil {
				AppLog("ERROR", "ASS/SSA to SRT conversion failed: %v", *err)
				result.SetText(result.Text + "\nError converting ASS/SSA to SRT: " + (*err).Error())
				statusLabel.SetText("Conversion failed!")
				conversionProgress.SetValue(0)
			} else {
				result.SetText(result.Text + "\nSuccessfully converted ASS/SSA to SRT")
				statusLabel.SetText("Conversion completed!")
				conversionProgress.SetValue(100)

				// Check if the output file was created
				if _, statErr := os.Stat(absOutputPath); statErr == nil {
					result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
				} else {
					result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
				}
			}

			// Update elapsed time one last time
			elapsed := time.Since(conversionStartTime).Round(time.Second)
			elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
			remainingLabel.SetText("Completed")
		})
	}
}

// extractVobsubTrack handles VobSub to SRT conversion extraction
func extractVobsubTrack(
	w fyne.Window,
	t *TrackItem,
	mkvPath *string,
	outDir *string,
	mkvBaseName string,
	result *widget.Label,
	progress *widget.ProgressBar,
	currentTrackLabel *widget.Label,
	trackList *fyne.Container,
	output *[]byte,
	err *error,
) {
	// VobSub to SRT conversion
	fyne.Do(func() {
		result.SetText(setLogMessage(LogConvert, "VobSub to SRT Conversion", "Starting VobSub to SRT conversion process..."))
	})

	// For VobSub, we extract both .idx and .sub files
	idxFile := fmt.Sprintf("%s.track%d_%s.idx", mkvBaseName, t.Num, t.Lang)
	_ = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

	// Get absolute paths for extraction
	absIdxPath := filepath.Join(*outDir, idxFile)

	// Debug output
	fyne.Do(func() {
		currentTrackLabel.SetText(fmt.Sprintf("Extracting VobSub track %d...", t.Num))
		result.SetText(result.Text + "\n\n=== VobSub Extraction ===\n")
		result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
		result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", *outDir))
		result.SetText(result.Text + fmt.Sprintf("IDX file: %s\n", idxFile))
		result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absIdxPath))
	})

	// Extract VobSub first
	var cmd *exec.Cmd
	var cmdStr string
	if IsMP4File(*mkvPath) {
		fyne.Do(func() {
			result.SetText(result.Text + "\nExtracting VobSub track using ffmpeg...")
		})
		*output, *err = ExtractMP4Subtitle(*mkvPath, t.Num, absIdxPath)
	} else {
		cmdStr = fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", *mkvPath, t.Num, idxFile)
		fyne.Do(func() {
			result.SetText(result.Text + "\nRunning: " + cmdStr)
		})

		cmd = NewMkvextractCmd("tracks", *mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
		AppLog("DEBUG", "VobSub track extraction using mkvextract")
		cmd.Dir = *outDir

		AppLog("EXTRACT", "VobSub extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(*mkvPath))
		*output, *err = cmd.CombinedOutput()
		AppLogCmd(cmd, *output, *err)
	}

	// Debug output - show command result
	fyne.Do(func() {
		result.SetText(result.Text + "\nCommand output: " + string(*output))
		if *err != nil {
			result.SetText(result.Text + "\nError: " + (*err).Error())
		}
	})

	// Check if the file was created and has content
	idxFilePath := filepath.Join(*outDir, idxFile)
	fileInfo, statErr := os.Stat(idxFilePath)
	if statErr != nil {
		fyne.Do(func() {
			result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
		})
		*err = statErr
	} else if fileInfo.Size() == 0 {
		fyne.Do(func() {
			result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
		})
		*err = fmt.Errorf("extracted file is empty")
	} else {
		// File exists and has content, proceed with conversion
		fyne.Do(func() {
			result.SetText(result.Text + fmt.Sprintf("\nIDX file extracted successfully (%d bytes)", fileInfo.Size()))
			result.SetText(result.Text + "\n\n=== VobSub to SRT Conversion ===\n")
		})

		// Create UI elements for conversion progress
		conversionStartTime := time.Now()
		conversionLabel := widget.NewLabel("Converting VobSub to SRT...")
		statusLabel := widget.NewLabel("Starting conversion...")
		conversionProgress := widget.NewProgressBar()
		elapsedLabel := widget.NewLabel("Elapsed: 0s")
		remainingLabel := widget.NewLabel("Estimating...")

		// Start a ticker to update the elapsed time
		ticker := time.NewTicker(time.Second)
		go func() {
			for range ticker.C {
				elapsed := time.Since(conversionStartTime).Round(time.Second)
				fyne.Do(func() {
					elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
				})
			}
		}()

		// Show the conversion progress bar and labels
		fyne.Do(func() {
			currentTrackLabel.SetText("Converting VobSub to SRT...")
			progress.Hide()
			trackList.Add(container.NewVBox(
				conversionLabel,
				statusLabel,
				conversionProgress,
				container.NewHBox(
					elapsedLabel,
					widget.NewLabel("|"),
					remainingLabel,
				),
			))
			trackList.Refresh()
		})

		// Get absolute paths for input and output
		// For vobsub2srt, we need the base path without extension
		basePath := strings.TrimSuffix(idxFilePath, filepath.Ext(idxFilePath))
		absOutputPath := basePath + ".srt" // vobsub2srt will create this file

		// Check if both .idx and .sub files exist
		idxFileCheck := basePath + ".idx"
		subFile := basePath + ".sub"

		fyne.Do(func() {
			result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for IDX file: %s", idxFileCheck))
			result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for SUB file: %s", subFile))
		})

		// Check if the files exist
		var filesExist bool = true
		if _, chkErr := os.Stat(idxFileCheck); chkErr == nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file exists: %s", idxFileCheck))
			})
		} else {
			filesExist = false
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file does not exist: %s - %v", idxFileCheck, chkErr))
			})
		}

		if _, chkErr := os.Stat(subFile); chkErr == nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file exists: %s", subFile))
			})
		} else {
			filesExist = false
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file does not exist: %s - %v", subFile, chkErr))
			})
		}

		// If either file is missing, show a warning
		if !filesExist {
			fyne.Do(func() {
				result.SetText(result.Text + "\n[DEBUG] ⚠️ Warning: IDX or SUB file is missing, conversion may fail")
			})
		}

		// Get language from user selection or use track language as default
		langCode := t.Lang
		if langCode == "" {
			langCode = "eng" // Default to English if no language code is available
		}

		// Check if user has selected a specific language
		if t.LangSelect != nil && t.LangSelect.Selected != "" && !strings.HasPrefix(t.LangSelect.Selected, "Auto") {
			// Extract the language code from the selection (format: "Language (code)")
			selection := t.LangSelect.Selected
			// Extract the code part between parentheses
			if start := strings.LastIndex(selection, "("); start != -1 {
				if end := strings.LastIndex(selection, ")"); end != -1 && end > start {
					// Extract the 2-letter code directly
					twoLetterCode := selection[start+1 : end]
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] User selected language: %s (code: %s)", selection, twoLetterCode))
					})
					langCode = twoLetterCode
				}
			}
		} else {
			// Using auto-detected language, map 3-letter code to 2-letter code
			langCodeMap := map[string]string{
				"eng": "en", "fre": "fr", "fra": "fr", "ger": "de", "deu": "de",
				"ita": "it", "spa": "es", "por": "pt", "dut": "nl", "nld": "nl",
				"swe": "sv", "nor": "no", "dan": "da", "fin": "fi", "jpn": "ja",
				"kor": "ko", "chi": "zh", "zho": "zh", "rus": "ru", "pol": "pl",
				"cze": "cs", "ces": "cs", "hun": "hu", "gre": "el", "ell": "el",
				"tur": "tr", "ara": "ar", "heb": "he", "tha": "th",
			}

			// Convert 3-letter code to 2-letter code if a mapping exists
			if twoLetterCode, exists := langCodeMap[strings.ToLower(langCode)]; exists {
				fyne.Do(func() {
					result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code: ** %s -> %s **", langCode, twoLetterCode))
				})
				langCode = twoLetterCode
			} else {
				fyne.Do(func() {
					result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] No mapping found for language code: ** %s **, using as-is", langCode))
				})
			}
		}

		// Use vobsub2srt binary for conversion
		conversionScript := "/usr/local/bin/vobsub2srt"

		// Check if the binary exists
		if _, binErr := os.Stat(conversionScript); binErr != nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[ERROR] vobsub2srt binary not found at %s", conversionScript))
			})
			*err = fmt.Errorf("vobsub2srt binary not found at %s", conversionScript)
		} else {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using vobsub2srt binary: %s", conversionScript))
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code: %s for VobSub conversion", langCode))
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Base path for vobsub2srt: %s", basePath))
			})

			// Check if the output SRT file already exists and delete it if it does
			outputSrtFile := basePath + ".srt"
			if _, srtErr := os.Stat(outputSrtFile); srtErr == nil {
				fyne.Do(func() {
					result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Removing existing SRT file: %s", outputSrtFile))
				})
				os.Remove(outputSrtFile)
			}

			// Run vobsub2srt with the language parameter
			cmdStr = fmt.Sprintf("%s --lang %s \"%s\"", conversionScript, langCode, basePath)
			fyne.Do(func() {
				result.SetText(result.Text + "\n[DEBUG] Running command: " + cmdStr)
				statusLabel.SetText("Running vobsub2srt conversion...")
			})

			// Create the command
			cmd = exec.Command(conversionScript, "--lang", langCode, basePath)
			cmd.Dir = *outDir

			// Run the command and capture output
			*output, *err = cmd.CombinedOutput()

			// Stop the ticker
			ticker.Stop()

			// Update UI with results
			fyne.Do(func() {
				result.SetText(result.Text + "\nvobsub2srt output: " + string(*output))

				if *err != nil {
					result.SetText(result.Text + "\nError converting VobSub to SRT: " + (*err).Error())
					statusLabel.SetText("Conversion failed!")
					conversionProgress.SetValue(0)
				} else {
					result.SetText(result.Text + "\nSuccessfully ran vobsub2srt command")
					statusLabel.SetText("Conversion completed!")
					conversionProgress.SetValue(100)

					// Check if the output file was created
					if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
						result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
						result.SetText(result.Text + fmt.Sprintf("\nSRT file size: %d bytes", fileInfo.Size()))

						// Try to count lines in SRT file
						if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
							lines := strings.Split(string(srtContent), "\n")
							result.SetText(result.Text + fmt.Sprintf("\nSRT file lines: %d", len(lines)))

							// Count subtitle entries (every 4 lines is typically one subtitle)
							subtitleCount := (len(lines) + 3) / 4 // rough estimate
							result.SetText(result.Text + fmt.Sprintf("\nEstimated subtitles: ~%d", subtitleCount))
						}
					} else {
						result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
					}
				}

				// Update elapsed time one last time
				elapsed := time.Since(conversionStartTime).Round(time.Second)
				elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
				remainingLabel.SetText("Completed")
			})
		}
	}
}
