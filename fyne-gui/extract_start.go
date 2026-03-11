package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func extractCompletionActions(w fyne.Window, outDir string) *fyne.Container {
	openFolderBtn := widget.NewButton(T("common.open_output_folder"), func() {
		openFolderPath(w, outDir)
	})
	openFolderBtn.Importance = widget.MediumImportance

	copyPathBtn := widget.NewButton(T("common.copy_output_path"), func() {
		copyTextToClipboard(w, outDir)
	})
	copyPathBtn.Importance = widget.MediumImportance

	return container.NewHBox(openFolderBtn, copyPathBtn)
}

// extractStartExtraction handles the "Start Extraction" button logic for the extract tab.
func extractStartExtraction(
	w fyne.Window,
	mkvPath *string,
	outDir *string,
	trackItems *[]*TrackItem,
	mkvFiles *[]string,
	batchMode *bool,
	result *widget.Label,
	progress *widget.ProgressBar,
	currentTrackLabel *widget.Label,
	trackList *fyne.Container,
) {
	if *batchMode {
		// Batch processing mode
		if len(*mkvFiles) == 0 || *outDir == "" {
			dialog.ShowError(fmt.Errorf("Please select video files and output directory for batch processing."), w)
			return
		}

		// Start batch processing
		go func() {
			totalFiles := len(*mkvFiles)
			successCount := 0
			failureCount := 0

			fyne.Do(func() {
				result.SetText(setLogMessage(LogInfo, "Batch Processing Started", fmt.Sprintf("Processing %d video files...", totalFiles)))
				progress.Max = float64(totalFiles)
				progress.SetValue(0)
				progress.Show()
				currentTrackLabel.Show()
			})

			for fileIndex, currentMkvPath := range *mkvFiles {
				*mkvPath = currentMkvPath // Set current file for processing

				fyne.Do(func() {
					currentTrackLabel.SetText(fmt.Sprintf("Processing file %d/%d: %s", fileIndex+1, totalFiles, filepath.Base(currentMkvPath)))
					progress.SetValue(float64(fileIndex))
				})

				// Load tracks for current file (simplified check)
				if !loadTracksForFile(currentMkvPath) {
					failureCount++
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to load tracks for: %s", filepath.Base(currentMkvPath)))
					})
					continue
				}

				// Extract selected tracks for this file
				selectedTracksForFile := []*TrackItem{}
				for _, t := range *trackItems {
					if t.Check.Checked && t.FilePath == currentMkvPath {
						selectedTracksForFile = append(selectedTracksForFile, t)
					}
				}

				if len(selectedTracksForFile) == 0 {
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n⏭️ Skipped (no tracks selected): %s", filepath.Base(currentMkvPath)))
					})
					continue
				}

				// Extract selected tracks
				fileSuccess := true
				for _, track := range selectedTracksForFile {
					fyne.Do(func() {
						track.Status.SetText("Extracting...")
					})

					// Determine file extension based on codec
					convertOCR := track.ConvertOCR != nil && track.ConvertOCR.Checked
					var ext string
					if IsMP4File(currentMkvPath) {
						ext = MP4CodecToExtension(track.Codec)
					} else {
						ext = CodecToExtension(track.Codec, convertOCR)
					}

					videoBaseName := filepath.Base(currentMkvPath)
					videoBaseName = strings.TrimSuffix(videoBaseName, filepath.Ext(videoBaseName))
					outFile := fmt.Sprintf("%s.track%d_%s.%s", videoBaseName, track.Num, track.Lang, ext)
					outPath := filepath.Join(*outDir, outFile)

					var output []byte
					var err error

					if IsMP4File(currentMkvPath) {
						// Use ffmpeg for MP4/M4V files
						output, err = ExtractMP4Subtitle(currentMkvPath, track.Num, outPath)
					} else {
						// Use mkvextract for MKV files
						extractCmd := NewMkvextractCmd("tracks", currentMkvPath, fmt.Sprintf("%d:%s", track.Num, outFile))
						extractCmd.Dir = *outDir

						AppLog("EXTRACT", "Batch extraction: Track %d (%s) from %s", track.Num, track.Lang, filepath.Base(currentMkvPath))
						output, err = extractCmd.CombinedOutput()
						AppLogCmd(extractCmd, output, err)
					}

					if err != nil {
						fileSuccess = false
						AppLog("ERROR", "Batch extraction failed for track %d: %v", track.Num, err)
						fyne.Do(func() {
							track.Status.SetText("Failed")
						})
					} else {
						AppLog("SUCCESS", "Batch extraction completed: %s", outFile)
						fyne.Do(func() {
							track.Status.SetText("Done")
						})
					}
				}

				if fileSuccess {
					successCount++
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n✅ Successfully processed: %s (%d tracks)", filepath.Base(currentMkvPath), len(selectedTracksForFile)))
					})
				} else {
					failureCount++
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n❌ Partially failed: %s", filepath.Base(currentMkvPath)))
					})
				}
			}

			// Final batch processing results
			fyne.Do(func() {
				progress.SetValue(float64(totalFiles))
				progress.Hide()
				currentTrackLabel.SetText(T("extract.batch_completed"))
				result.SetText(result.Text + fmt.Sprintf(T("extract.batch_complete_result_actions"), successCount, failureCount, *outDir))
				dialog.ShowCustom(
					T("extract.batch_processing_complete_title"),
					T("common.close"),
					container.NewVBox(
						widget.NewLabel(fmt.Sprintf(T("extract.batch_processing_complete_body"), totalFiles, successCount, failureCount, *outDir)),
						extractCompletionActions(w, *outDir),
					),
					w,
				)
			})

			// Show completion notification
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   T("extract.batch_processing_complete_title"),
				Content: fmt.Sprintf("Processed %d files. %d successful, %d failed.", totalFiles, successCount, failureCount),
			})
		}()
		return
	}

	// Single file processing mode
	if *mkvPath == "" || *outDir == "" {
		dialog.ShowError(fmt.Errorf("Please select both a video file and output directory."), w)
		return
	}

	go func() {
		selected := []*TrackItem{}
		for _, t := range *trackItems {
			if t.Check.Checked {
				selected = append(selected, t)
			}
		}
		if len(selected) == 0 {
			// Thread-safe UI update
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "No Tracks",
				Content: "No tracks selected.",
			})
			return
		}

		// Set up progress bar
		fyne.Do(func() {
			result.SetText(setLogMessage(LogInfo, "Extraction Started", "Extracting selected tracks..."))
			progress.Max = float64(len(selected))
			progress.SetValue(0)
			progress.Show()
			currentTrackLabel.Show()
		})

		tracksDone := 0
		var output []byte
		var err error

		for i, t := range selected {
			// Update UI on main thread
			fyne.Do(func() {
				currentTrackLabel.SetText(setLogMessage(LogExtract, fmt.Sprintf("Extracting Track %d/%d", i+1, len(selected)), t.Name))
			})

			// Extract the subtitle track
			var outFile string

			// Get base filename without extension
			mkvBaseName := filepath.Base(*mkvPath)
			mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))

			// Check if this is a PGS track with OCR conversion requested
			if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS") {
				extractPgsTrack(w, t, mkvPath, outDir, mkvBaseName, result, progress, currentTrackLabel, trackList, &output, &err)
			} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") || strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station")) {
				extractAssTrack(w, t, mkvPath, outDir, mkvBaseName, result, progress, currentTrackLabel, trackList, &output, &err)
			} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "vobsub" || t.Codec == "VobSub") {
				extractVobsubTrack(w, t, mkvPath, outDir, mkvBaseName, result, progress, currentTrackLabel, trackList, &output, &err)
			} else {
				// Normal extraction without conversion
				var fileExt string
				if IsMP4File(*mkvPath) {
					fileExt = MP4CodecToExtension(t.Codec)
				} else {
					fileExt = CodecToExtensionForExtract(t.Codec)
				}

				fyne.Do(func() {
					result.SetText(result.Text + "\n\n=== Track Extraction ===\n")
					result.SetText(result.Text + fmt.Sprintf("Track: %d (%s - %s)\n", t.Num, t.Lang, t.Codec))
				})

				outFile = fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, t.Num, t.Lang, fileExt)

				fyne.Do(func() {
					result.SetText(result.Text + fmt.Sprintf("Output file: %s\n", outFile))
				})
				absOutFile := filepath.Join(*outDir, outFile)

				fyne.Do(func() {
					result.SetText(result.Text + fmt.Sprintf("\nExtracting to: %s", absOutFile))
				})

				if IsMP4File(*mkvPath) {
					AppLog("EXTRACT", "Generic extraction (ffmpeg): Track %d (%s/%s) from %s", t.Num, t.Lang, t.Codec, filepath.Base(*mkvPath))
					output, err = ExtractMP4Subtitle(*mkvPath, t.Num, absOutFile)
				} else {
					cmd := NewMkvextractCmd("tracks", *mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))
					AppLog("DEBUG", "Generic track extraction using mkvextract")
					AppLog("EXTRACT", "Generic extraction: Track %d (%s/%s) from %s", t.Num, t.Lang, t.Codec, filepath.Base(*mkvPath))
					output, err = cmd.CombinedOutput()
					AppLogCmd(cmd, output, err)
				}

				if err == nil {
					outFilePath := filepath.Join(*outDir, outFile)
					os.Chmod(outFilePath, 0644)
				}
			}

			// Update UI on main thread
			fyne.Do(func() {
				if err != nil {
					t.State = "Error"
					AppLog("ERROR", "Track %d (%s) extraction/conversion failed: %v", t.Num, t.Name, err)
					t.Status.SetText(setLogMessage(LogError, fmt.Sprintf("Error Extracting Track %s", t.Name), err.Error()))
					if t.ConvertOCR != nil && t.ConvertOCR.Checked {
						result.SetText(result.Text + setLogMessage(LogError, "Conversion Failed", err.Error()))
					} else {
						result.SetText(result.Text + setLogMessage(LogError, "Extraction Failed", err.Error()))
					}
				} else {
					t.State = "Done"
					if t.ConvertOCR != nil && t.ConvertOCR.Checked {
						AppLog("SUCCESS", "Track %d (%s) converted to SRT", t.Num, t.Name)
						t.Status.SetText("Converted")
						result.SetText(result.Text + setLogMessage(LogSuccess, "Conversion Complete", fmt.Sprintf("Successfully converted %s to SRT.", t.Name)))
					} else {
						AppLog("SUCCESS", "Track %d (%s) extracted", t.Num, t.Name)
						t.Status.SetText("Extracted")
						result.SetText(result.Text + setLogMessage(LogSuccess, "Track Extracted", t.Name))
					}
				}

				// Update track list
				trackList.Objects = nil
				for _, tt := range *trackItems {
					trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", tt.Num, tt.Lang, tt.Codec, tt.Name))

					if tt.ConvertOCR != nil {
						ocrLabel := widget.NewLabel("Convert to SRT")
						row := container.NewHBox(tt.Check, tt.Status, trackInfo, tt.ConvertOCR, ocrLabel)
						trackList.Add(row)
					} else {
						row := container.NewHBox(tt.Check, tt.Status, trackInfo)
						trackList.Add(row)
					}
				}
				trackList.Refresh()
			})

			tracksDone++
			fyne.Do(func() {
				progress.SetValue(float64(tracksDone))
			})
		}

		// Final UI update on main thread
		fyne.Do(func() {
			currentTrackLabel.SetText("")
			currentTrackLabel.Hide()
			progress.Hide()
			if tracksDone == len(selected) {
				result.SetText(setLogMessage(LogSuccess, T("extract.extraction_complete"), fmt.Sprintf(T("extract.extraction_complete_body"), *outDir)))
				progress.SetValue(progress.Max)
				dialog.ShowCustom(
					T("extract.extraction_complete"),
					T("common.close"),
					container.NewVBox(
						widget.NewLabel(fmt.Sprintf(T("extract.extraction_complete_body"), *outDir)),
						extractCompletionActions(w, *outDir),
					),
					w,
				)
			} else {
				result.SetText(fmt.Sprintf("Extraction stopped after %d of %d tracks", tracksDone, len(selected)))
			}
		})
	}()
}

// extractPgsTrack handles PGS to SRT conversion extraction
func extractPgsTrack(
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
	// First extract as PGS
	fyne.Do(func() {
		result.SetText(setLogMessage(LogConvert, "PGS to SRT Conversion", "Starting PGS extraction process..."))
	})
	tempPgsFile := fmt.Sprintf("%s.track%d_%s.sup", mkvBaseName, t.Num, t.Lang)
	outFile := fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang)

	absPgsPath := filepath.Join(*outDir, tempPgsFile)

	fyne.Do(func() {
		currentTrackLabel.SetText(fmt.Sprintf("Extracting PGS track %d...", t.Num))
		result.SetText(result.Text + "\n\n=== PGS Extraction ===\n")
		result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
		result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", *outDir))
		result.SetText(result.Text + fmt.Sprintf("PGS file: %s\n", tempPgsFile))
		result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absPgsPath))
	})

	var cmd *exec.Cmd
	if IsMP4File(*mkvPath) {
		fyne.Do(func() {
			result.SetText(result.Text + "\nExtracting PGS track using ffmpeg...")
		})
		*output, *err = ExtractMP4Subtitle(*mkvPath, t.Num, absPgsPath)
	} else {
		cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", *mkvPath, t.Num, tempPgsFile)
		fyne.Do(func() {
			result.SetText(result.Text + "\nRunning: " + cmdStr)
		})
		cmd = NewMkvextractCmd("tracks", *mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
		AppLog("DEBUG", "PGS track extraction using mkvextract")
		cmd.Dir = *outDir
		AppLog("EXTRACT", "PGS extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(*mkvPath))
		*output, *err = cmd.CombinedOutput()
		AppLogCmd(cmd, *output, *err)
	}

	fyne.Do(func() {
		result.SetText(result.Text + "\nCommand output: " + string(*output))
		if *err != nil {
			result.SetText(result.Text + "\nError: " + (*err).Error())
		}
	})

	pgsFilePath := filepath.Join(*outDir, tempPgsFile)
	fileInfo, statErr := os.Stat(pgsFilePath)
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
			result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted PGS file (%d bytes)", fileInfo.Size()))
		})
	}

	if *err == nil {
		conversionProgress := widget.NewProgressBar()
		conversionProgress.Min = 0
		conversionProgress.Max = 100
		conversionProgress.SetValue(0)

		conversionLabel := widget.NewLabel("Converting PGS to SRT...")
		statusLabel := widget.NewLabel("Initializing OCR process...")
		elapsedLabel := widget.NewLabel("Elapsed: 0s")
		remainingLabel := widget.NewLabel("Estimated time remaining: calculating...")

		conversionStartTime := time.Now()
		var progressMutex sync.Mutex
		var progressData = struct {
			currentFrame int
			totalFrames  int
			frameRate    float64
			lastUpdate   time.Time
		}{
			currentFrame: 0,
			totalFrames:  0,
			frameRate:    0,
			lastUpdate:   time.Now(),
		}

		ticker := time.NewTicker(500 * time.Millisecond)
		go func() {
			defer ticker.Stop()
			var lastElapsedText, lastRemainingText string
			for range ticker.C {
				elapsed := time.Since(conversionStartTime).Round(time.Second)
				newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)
				progressMutex.Lock()
				currentFrame := progressData.currentFrame
				totalFrames := progressData.totalFrames
				frameRate := progressData.frameRate
				progressMutex.Unlock()
				var newRemainingText string
				var progressValue float64
				if totalFrames > 0 && currentFrame > 0 && frameRate > 0 {
					progressValue = float64(currentFrame) / float64(totalFrames) * 100
					framesRemaining := totalFrames - currentFrame
					secondsRemaining := float64(framesRemaining) / frameRate
					remaining := time.Duration(secondsRemaining * float64(time.Second))
					remaining = remaining.Round(time.Second)
					newRemainingText = fmt.Sprintf("Estimated time remaining: %s", remaining)
				} else {
					newRemainingText = "Estimated time remaining: calculating..."
					progressValue = 0
				}
				if newElapsedText != lastElapsedText || newRemainingText != lastRemainingText {
					lastElapsedText = newElapsedText
					lastRemainingText = newRemainingText
					fyne.Do(func() {
						elapsedLabel.SetText(newElapsedText)
						remainingLabel.SetText(newRemainingText)
						conversionProgress.SetValue(progressValue)
					})
				}
			}
		}()

		fyne.Do(func() {
			result.SetText(result.Text + "\n\n[DEBUG] PGS extraction completed successfully, starting conversion process")
			currentTrackLabel.SetText("Converting PGS to SRT...")
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

		langCode := "eng"
		if t.Lang != "" {
			langCode = t.Lang
		}

		if t.LangSelect != nil && t.LangSelect.Selected != "" && !strings.HasPrefix(t.LangSelect.Selected, "Auto") {
			selection := t.LangSelect.Selected
			if start := strings.LastIndex(selection, "("); start != -1 {
				if end := strings.LastIndex(selection, ")"); end != -1 && end > start {
					twoLetterCode := selection[start+1 : end]
					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] User selected OCR language: %s (code: %s)", selection, twoLetterCode))
					})
					langCodeMap := map[string]string{
						"en": "eng", "fr": "fra", "de": "deu", "it": "ita", "es": "spa",
						"pt": "por", "nl": "nld", "sv": "swe", "no": "nor", "da": "dan",
						"fi": "fin", "ja": "jpn", "ko": "kor", "zh": "chi", "ru": "rus",
						"pl": "pol", "cs": "ces", "hu": "hun", "el": "ell", "tr": "tur",
						"ar": "ara", "he": "heb", "th": "tha",
					}
					if threeLetterCode, exists := langCodeMap[twoLetterCode]; exists {
						langCode = threeLetterCode
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code for OCR: %s -> %s", twoLetterCode, langCode))
						})
					} else {
						langCode = twoLetterCode
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code as-is for OCR: %s", langCode))
						})
					}
				}
			}
		}

		absInputPath := filepath.Join(*outDir, tempPgsFile)
		absOutputPath := filepath.Join(*outDir, outFile)

		pgsripAvailable := checkPgsrip()
		if pgsripAvailable {
			fyne.Do(func() {
				result.SetText(result.Text + "\n\n=== Using pgsrip for conversion ===\n")
				statusLabel.SetText("Starting pgsrip conversion...")
			})
			convSettings := PgsConversionSettings{Verbose: true}
			*err = convertPgsWithPgsrip(absInputPath, absOutputPath, langCode, result, statusLabel, conversionProgress, convSettings)
			if *err == nil {
				fyne.Do(func() {
					result.SetText(result.Text + "\n\n✅ PGS to SRT conversion with pgsrip completed successfully!")
					statusLabel.SetText("Conversion complete!")
					conversionProgress.SetValue(100)
				})
				return
			} else {
				fyne.Do(func() {
					result.SetText(result.Text + "\n⚠️ pgsrip conversion failed: " + (*err).Error() + "\nFalling back to pgs-to-srt...")
				})
			}
		}

		pgsToSrtScript := pgsToSrtScriptPath
		trainedDataPath := filepath.Join(filepath.Dir(pgsToSrtScript), "tessdata_fast", langCode+".traineddata")

		fyne.Do(func() {
			result.SetText(result.Text + fmt.Sprintf("\n\n[DEBUG] Checking if script exists at: %s", pgsToSrtScript))
		})

		if _, statErr := os.Stat(pgsToSrtScript); statErr != nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Script NOT found: %v", statErr))
			})
			return
		}

		fyne.Do(func() {
			result.SetText(result.Text + "\n[DEBUG] Script found!")
			result.SetText(result.Text + "\n[DEBUG] Running Deno version test...")
		})
		testCmd := exec.Command("deno", "--version")
		testOutput, testErr := testCmd.CombinedOutput()
		fyne.Do(func() {
			result.SetText(result.Text + "\n\n=== Deno Version Test ===\n")
			if testErr != nil {
				result.SetText(result.Text + fmt.Sprintf("Deno test error: %v\n", testErr))
			} else {
				result.SetText(result.Text + fmt.Sprintf("Deno version: %s\n", string(testOutput)))
			}
		})

		textUpdate := fmt.Sprintf("\nInput SUP file: %s\nOutput SRT file: %s\nTessdata file: %s\n",
			absInputPath, absOutputPath, trainedDataPath)
		fyne.Do(func() {
			result.SetText(result.Text + textUpdate)
			if fileInfo, err := os.Stat(absInputPath); err == nil {
				result.SetText(result.Text + fmt.Sprintf("Input file size: %d bytes\n", fileInfo.Size()))
			} else {
				result.SetText(result.Text + fmt.Sprintf("Input file check error: %v\n", err))
			}
		})

		var copyErr error
		var copySuccess bool

		tmpOutputFile, tmpErr := os.CreateTemp("", "pgs_to_srt_*.srt")
		if tmpErr != nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create temporary file: %v", tmpErr))
			})
			return
		}
		tmpOutputPath := tmpOutputFile.Name()
		tmpOutputFile.Close()

		cmdStr := fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"", pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
		updateText := fmt.Sprintf("\n\n=== Executing Command ===\n%s\n\nConversion started at: %s\n",
			cmdStr, time.Now().Format("15:04:05"))
		fyne.Do(func() {
			result.SetText(result.Text + updateText)
		})

		logFileName := filepath.Join(*outDir, fmt.Sprintf("%s.track%d_%s.conversion.log", mkvBaseName, t.Num, t.Lang))
		logFile, logErr := os.Create(logFileName)
		var logger *log.Logger
		if logErr != nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create log file: %v", logErr))
			})
		} else {
			defer logFile.Close()
			logger = log.New(logFile, "", log.LstdFlags)
			logger.Printf("=== PGS to SRT Conversion Log ===\n")
			logger.Printf("Started at: %s\n", time.Now().Format("15:04:05"))
			logger.Printf("Input file: %s\n", absInputPath)
			logger.Printf("Final output file: %s\n", absOutputPath)
			logger.Printf("Temporary output file: %s\n", tmpOutputPath)
			logger.Printf("Script: %s\n", pgsToSrtScript)
			logger.Printf("Trained data: %s\n", trainedDataPath)
			logger.Printf("Working directory: %s\n", filepath.Dir(pgsToSrtScript))
			logger.Printf("PATH: %s\n\n", os.Getenv("PATH"))
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n📝 Created log file: %s", logFileName))
				result.SetText(result.Text + fmt.Sprintf("\n📂 Using temporary file: %s", tmpOutputPath))
			})
		}

		var denoCmd string
		if denoBinaryPath != "" {
			denoCmd = fmt.Sprintf("%s run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"",
				denoBinaryPath, pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
			logger.Printf("Using stored Deno path: %s\n", denoBinaryPath)
		} else {
			denoCmd = fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"",
				pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
			logger.Printf("No stored Deno path, using default 'deno' command\n")
		}
		cmd = exec.Command("sh", "-c", denoCmd)
		cmd.Dir = filepath.Dir(pgsToSrtScript)

		fyne.Do(func() {
			result.SetText(result.Text + "\n\n=== Environment ===\n")
			result.SetText(result.Text + fmt.Sprintf("Working directory: %s\n", cmd.Dir))
			result.SetText(result.Text + fmt.Sprintf("PATH: %s\n", os.Getenv("PATH")))
			result.SetText(result.Text + "\n=== Command ===\n")
			if denoBinaryPath != "" {
				result.SetText(result.Text + fmt.Sprintf("%s run --allow-read --allow-write %s %s %s > %s\n",
					denoBinaryPath, pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
			} else {
				result.SetText(result.Text + fmt.Sprintf("deno run --allow-read --allow-write %s %s %s > %s\n",
					pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
			}
		})

		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		startErr := cmd.Start()
		if startErr != nil {
			fyne.Do(func() {
				result.SetText(result.Text + fmt.Sprintf("\n\n❌ Failed to start command: %v", startErr))
			})
			if logFile != nil && logger != nil {
				logger.Printf("Failed to start command: %v\n", startErr)
			}
			*err = startErr
		} else {
			fyne.Do(func() {
				result.SetText(result.Text + "\n\n=== Starting Conversion Process ===\n")
				result.SetText(result.Text + "Check the log file for real-time output\n")
			})

			var outputBuffer strings.Builder
			var stdoutWriter, stderrWriter io.Writer
			if logFile != nil && logger != nil {
				stdoutWriter = io.MultiWriter(logFile, &outputBuffer)
				stderrWriter = io.MultiWriter(logFile, &outputBuffer)
				logger.Printf("Command started successfully\n")
			} else {
				stdoutWriter = &outputBuffer
				stderrWriter = &outputBuffer
			}

			frameProgressRegex := regexp.MustCompile(`Processing frame (\d+)/(\d+)`)
			statusUpdateRegex := regexp.MustCompile(`Status: (.+)`)

			go func() {
				bufReader := bufio.NewReaderSize(stdoutPipe, 4096)
				scanner := bufio.NewScanner(bufReader)
				for scanner.Scan() {
					line := scanner.Text() + "\n"
					if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
						currentFrame := 0
						totalFrames := 0
						fmt.Sscanf(matches[1], "%d", &currentFrame)
						fmt.Sscanf(matches[2], "%d", &totalFrames)
						progressMutex.Lock()
						if progressData.totalFrames == 0 {
							progressData.totalFrames = totalFrames
						}
						if progressData.currentFrame > 0 {
							timeDiff := time.Since(progressData.lastUpdate).Seconds()
							frameDiff := currentFrame - progressData.currentFrame
							if timeDiff > 0 && frameDiff > 0 {
								newFrameRate := float64(frameDiff) / timeDiff
								if progressData.frameRate > 0 {
									progressData.frameRate = progressData.frameRate*0.7 + newFrameRate*0.3
								} else {
									progressData.frameRate = newFrameRate
								}
							}
						}
						progressData.currentFrame = currentFrame
						progressData.lastUpdate = time.Now()
						progressMutex.Unlock()
						percentComplete := float64(currentFrame) / float64(totalFrames) * 100
						fyne.Do(func() {
							statusLabel.SetText(fmt.Sprintf("Processing frame %d of %d (%.1f%%)",
								currentFrame, totalFrames, percentComplete))
						})
					} else if matches := statusUpdateRegex.FindStringSubmatch(line); len(matches) == 2 {
						statusMsg := matches[1]
						fyne.Do(func() {
							statusLabel.SetText(statusMsg)
						})
					}
					if _, writeErr := stdoutWriter.Write([]byte(line)); writeErr != nil {
						break
					}
				}
			}()

			go func() {
				bufReader := bufio.NewReaderSize(stderrPipe, 4096)
				scanner := bufio.NewScanner(bufReader)
				for scanner.Scan() {
					line := scanner.Text() + "\n"
					if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
						currentFrame := 0
						totalFrames := 0
						fmt.Sscanf(matches[1], "%d", &currentFrame)
						fmt.Sscanf(matches[2], "%d", &totalFrames)
						progressMutex.Lock()
						if progressData.totalFrames == 0 {
							progressData.totalFrames = totalFrames
						}
						progressData.currentFrame = currentFrame
						progressMutex.Unlock()
					}
					if _, writeErr := stderrWriter.Write([]byte(line)); writeErr != nil {
						break
					}
				}
			}()

			*err = cmd.Wait()
			*output = []byte(outputBuffer.String())

			if logFile != nil && logger != nil {
				if *err != nil {
					logger.Printf("\n\nCommand completed with error: %v\n", *err)
				} else {
					logger.Printf("\n\nCommand completed successfully\n")
				}
				logger.Printf("Finished at: %s\n", time.Now().Format("15:04:05"))
			}

			if _, statErr := os.Stat(tmpOutputPath); statErr == nil {
				if logFile != nil && logger != nil {
					logger.Printf("Copying temporary file %s to final destination %s\n", tmpOutputPath, absOutputPath)
				}
				outputDir := filepath.Dir(absOutputPath)
				if mkdirErr := os.MkdirAll(outputDir, 0755); mkdirErr != nil {
					copyErr = fmt.Errorf("failed to create output directory: %v", mkdirErr)
					if logFile != nil && logger != nil {
						logger.Printf("Error creating output directory: %v\n", mkdirErr)
					}
				} else {
					tmpContent, readErr := os.ReadFile(tmpOutputPath)
					if readErr != nil {
						copyErr = fmt.Errorf("failed to read temporary file: %v", readErr)
						if logFile != nil && logger != nil {
							logger.Printf("Error reading temporary file: %v\n", readErr)
						}
					} else {
						writeErr := os.WriteFile(absOutputPath, tmpContent, 0644)
						if writeErr != nil {
							copyErr = fmt.Errorf("failed to write to final destination: %v", writeErr)
							if logFile != nil && logger != nil {
								logger.Printf("Error writing to final destination: %v\n", writeErr)
							}
						} else {
							copySuccess = true
							if logFile != nil && logger != nil {
								logger.Printf("Successfully copied temporary file to final destination\n")
							}
							removeErr := os.Remove(tmpOutputPath)
							if removeErr != nil && logFile != nil && logger != nil {
								logger.Printf("Warning: Could not remove temporary file: %v\n", removeErr)
							} else if logFile != nil && logger != nil {
								logger.Printf("Removed temporary file\n")
							}
						}
					}
				}
			} else {
				copyErr = fmt.Errorf("temporary file not found: %v", statErr)
				if logFile != nil && logger != nil {
					logger.Printf("Error: Temporary file not found: %v\n", statErr)
				}
			}

			if *err == nil && copyErr != nil {
				*err = copyErr
			}
		}

		var outputText strings.Builder
		outputText.WriteString("\nFull command output:\n")
		outputStr := string(*output)
		const maxOutputLen = 10000
		if len(outputStr) > maxOutputLen {
			outputText.WriteString(outputStr[:maxOutputLen])
			outputText.WriteString("\n... [Output truncated, full output in log file] ...")
		} else {
			outputText.WriteString(outputStr)
		}
		if *err != nil {
			outputText.WriteString("\n\n❌ Command error: " + (*err).Error())
		}
		fyne.Do(func() {
			result.SetText(result.Text + outputText.String())
		})

		fyne.Do(func() {
			conversionTime := time.Since(conversionStartTime).Round(time.Second)
			if *err != nil {
				currentTrackLabel.SetText(fmt.Sprintf("Conversion failed after %s", conversionTime))
			} else {
				currentTrackLabel.SetText(fmt.Sprintf("Conversion completed in %s", conversionTime))
			}
			progress.Show()
			for i, obj := range trackList.Objects {
				if box, ok := obj.(*fyne.Container); ok {
					for _, child := range box.Objects {
						if label, ok := child.(*widget.Label); ok && label.Text == "Converting PGS to SRT..." {
							trackList.Objects = append(trackList.Objects[:i], trackList.Objects[i+1:]...)
							break
						}
					}
				}
			}
			trackList.Refresh()
			result.SetText(result.Text + "\n\n=== Conversion Results ===\n")
			result.SetText(result.Text + "Completed at: " + time.Now().Format("15:04:05") + "\n")
			outputStr := string(*output)
			result.SetText(result.Text + "\nFull output: \n" + outputStr + "\n")
			if *err != nil {
				result.SetText(result.Text + "\n❌ Error: " + (*err).Error() + "\n")
			} else {
				result.SetText(result.Text + "\n✅ Command completed successfully\n")
				result.SetText(result.Text + "\n=== File Operations ===\n")
				result.SetText(result.Text + fmt.Sprintf("✓ Temporary file created: %s\n", tmpOutputPath))
				if copySuccess {
					result.SetText(result.Text + fmt.Sprintf("✓ Copied to final destination: %s\n", absOutputPath))
					result.SetText(result.Text + "✓ Temporary file cleaned up\n")
				} else if copyErr != nil {
					result.SetText(result.Text + fmt.Sprintf("❌ Failed to copy to final destination: %v\n", copyErr))
				}
			}
		})

		currentDir, _ := os.Getwd()
		fyne.Do(func() {
			result.SetText(result.Text + "\n\n=== Path Debugging ===\n")
			result.SetText(result.Text + fmt.Sprintf("Current working directory: %s\n", currentDir))
			result.SetText(result.Text + fmt.Sprintf("Looking for output file at: %s\n", absOutputPath))
		})

		files, _ := os.ReadDir(*outDir)
		fyne.Do(func() {
			result.SetText(result.Text + fmt.Sprintf("\nFiles in output directory (%s):\n", *outDir))
			for _, file := range files {
				result.SetText(result.Text + fmt.Sprintf("- %s\n", file.Name()))
			}
		})

		if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
			fyne.Do(func() {
				result.SetText(result.Text + "\n✅ SRT file created successfully!")
				result.SetText(result.Text + fmt.Sprintf("\n   - Path: %s", absOutputPath))
				result.SetText(result.Text + fmt.Sprintf("\n   - Size: %d bytes", fileInfo.Size()))
				result.SetText(result.Text + fmt.Sprintf("\n   - Modified: %s", fileInfo.ModTime().Format("15:04:05")))
				if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
					lines := strings.Split(string(srtContent), "\n")
					result.SetText(result.Text + fmt.Sprintf("\n   - Lines: %d", len(lines)))
					subtitleCount := (len(lines) + 3) / 4
					result.SetText(result.Text + fmt.Sprintf("\n   - Estimated subtitles: ~%d", subtitleCount))
				}
			})
		} else {
			*err = fmt.Errorf("SRT file was not created: %v", statErr)
			fyne.Do(func() {
				result.SetText(result.Text + "\n❌ Error: " + (*err).Error())
			})
		}
	}
}
