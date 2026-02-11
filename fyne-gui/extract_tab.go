package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ExtractTabWidgets holds references to widgets and callbacks needed by main()
type ExtractTabWidgets struct {
	DependencyButtons *fyne.Container
	OnDropped         func(pos fyne.Position, uris []fyne.URI)
	FileOpenFunc      func()
	DirChangeFunc     func()
	LoadTracksFunc    func()
	StartExtractFunc  func()
}

func createExtractSubtitlesTab(w fyne.Window, a fyne.App) (*fyne.Container, *ExtractTabWidgets) {
	trackList := container.NewVBox()
	trackListScroll := container.NewScroll(trackList)
	trackListScroll.SetMinSize(fyne.NewSize(850, 250))

	dependencyResults := checkDependencies()

	var mkvPath string
	var outDir string
	var trackItems []*TrackItem
	var mkvFiles []string
	var batchMode bool

	selectedFile := widget.NewLabel(T("extract.no_file"))
	selectedDir := widget.NewLabel(T("extract.no_dir"))
	result := widget.NewLabel(T("extract.results_placeholder"))
	result.Wrapping = fyne.TextWrapWord
	resultScroll := container.NewScroll(result)
	resultScroll.SetMinSize(fyne.NewSize(780, 200))

	dependencyStatus := "System Dependency Check:\n"
	allDependenciesInstalled := true
	for tool, installed := range dependencyResults {
		status := "✅ Installed"
		if !installed {
			status = "❌ Not found"
			allDependenciesInstalled = false
		}
		dependencyStatus += fmt.Sprintf("- %s: %s\n", tool, status)
	}
	if !allDependenciesInstalled {
		dependencyStatus += "\n⚠️ Some required tools are missing. Please install them before using all features.\n"
	} else {
		dependencyStatus += "\n✅ All required tools are installed.\n"
	}
	result.SetText(dependencyStatus)

	dependencyButtons := container.NewVBox()
	installAllContainer := container.NewHBox()
	missingDependencies := []string{}
	for tool, installed := range dependencyResults {
		if !installed {
			missingDependencies = append(missingDependencies, tool)
		}
	}

	if len(missingDependencies) > 0 {
		dependencyButtons.Add(widget.NewLabel("Install Missing Dependencies:"))
		for _, tool := range missingDependencies {
			toolName := tool
			buttonLabel := fmt.Sprintf("Install %s", toolName)
			installButton := widget.NewButton(buttonLabel, func() {
				installDependency(w, toolName)
			})
			dependencyButtons.Add(installButton)
		}

		if len(missingDependencies) > 1 {
			installAllButton := widget.NewButton("Install All Missing Dependencies", func() {
				dialog.ShowConfirm("Install All Dependencies",
					"This will attempt to install all missing dependencies.\n\nSome installations may require sudo privileges.\n\nDo you want to continue?",
					func(confirmed bool) {
						if confirmed {
							progress := dialog.NewProgress("Installing Dependencies", "Installing missing dependencies...", w)
							progress.Show()
							go func() {
								totalTools := len(missingDependencies)
								successCount := 0
								failureCount := 0
								for i, tool := range missingDependencies {
									progressValue := float64(i) / float64(totalTools)
									progress.SetValue(progressValue)
									var cmd *exec.Cmd
									switch tool {
									case "mkvmerge", "mkvextract":
										cmd = exec.Command("brew", "install", "mkvtoolnix")
									case "deno":
										cmd = exec.Command("brew", "install", "deno")
									case "tesseract":
										cmd = exec.Command("brew", "install", "tesseract")
									case "ffmpeg":
										cmd = exec.Command("brew", "install", "ffmpeg")
									case "vobsub2srt":
										execPath, err := os.Executable()
										if err != nil {
											fmt.Println("[ERROR] Failed to get executable path:", err)
											execPath = "."
										}
										execDir := filepath.Dir(execPath)
										scriptPath := filepath.Join(execDir, "install_vobsub2srt.sh")
										cmd = exec.Command("bash", scriptPath)
									case "pgsrip":
										execPath, err := os.Executable()
										if err != nil {
											fmt.Println("[ERROR] Failed to get executable path:", err)
											execPath = "."
										}
										execDir := filepath.Dir(execPath)
										scriptPath := filepath.Join(execDir, "install_pgsrip.sh")
										cmd = exec.Command("bash", scriptPath)
									default:
										fmt.Printf("[ERROR] Unknown tool: %s\n", tool)
										failureCount++
										continue
									}
									_, err := cmd.CombinedOutput()
									if err != nil {
										fmt.Printf("[ERROR] Failed to install %s: %v\n", tool, err)
										failureCount++
									} else {
										successCount++
									}
								}
								progress.Hide()
								if failureCount == 0 {
									dialog.ShowInformation("Installation Complete",
										fmt.Sprintf("All %d dependencies have been successfully installed.\n\nPlease restart the application to use all features.", successCount),
										w)
								} else {
									dialog.ShowInformation("Installation Results",
										fmt.Sprintf("%d dependencies installed successfully.\n%d dependencies failed to install.\n\nPlease check the logs for details and try installing the failed dependencies individually.",
											successCount, failureCount),
										w)
								}
								updateDependencyStatus(w)
							}()
						}
					}, w)
			})
			installAllContainer.Add(installAllButton)
			dependencyButtons.Add(installAllContainer)
		}
	}

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)
	progress.Hide()

	currentTrackLabel := widget.NewLabel("")
	currentTrackLabel.Hide()

	var fileList *widget.List
	fileList = widget.NewList(
		func() int { return len(mkvFiles) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(mkvFiles) {
				return
			}
			container := item.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			removeBtn := container.Objects[1].(*widget.Button)
			label.SetText(filepath.Base(mkvFiles[id]))
			removeBtn.OnTapped = func() {
				mkvFiles = append(mkvFiles[:id], mkvFiles[id+1:]...)
				fileList.Refresh()
				if len(mkvFiles) == 0 {
					batchMode = false
					selectedFile.SetText("No files selected")
				} else {
					selectedFile.SetText(fmt.Sprintf("%d video files selected for batch processing", len(mkvFiles)))
				}
			}
		},
	)

	fileListContainer := container.NewBorder(
		widget.NewLabel("Selected Files:"),
		nil, nil, nil,
		container.NewScroll(fileList),
	)
	fileListContainer.Hide()

	fileBtn := widget.NewButton(T("extract.select_file"), func() {
		filter := storage.NewExtensionFileFilter(VideoFileExtensions())
		fd := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				return
			}
			filePath := file.URI().Path()
			if !IsVideoFile(filePath) {
				dialog.ShowError(fmt.Errorf("Please select a supported video file (MKV, MP4, M4V)."), w)
				return
			}
			batchMode = false
			mkvFiles = []string{}
			fileList.Refresh()
			fileListContainer.Hide()
			mkvPath = filePath
			selectedFile.SetText(mkvPath)
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()
			result.SetText(setLogMessage(LogInfo, "Video File Loaded", "Video file loaded. Output directory automatically set to file location. Click 'Load Tracks' to analyze the file."))
		}, w)
		fd.SetFilter(filter)
		fd.Show()
	})

	batchBtn := widget.NewButton(T("extract.add_files"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			folderPath := uri.Path()
			var foundFiles []string
			err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && IsVideoFile(path) {
					foundFiles = append(foundFiles, path)
				}
				return nil
			})
			if err != nil {
				dialog.ShowError(fmt.Errorf("Error scanning folder: %v", err), w)
				return
			}
			if len(foundFiles) == 0 {
				dialog.ShowInformation("No Video Files", "No MKV or MP4 files found in the selected folder.", w)
				return
			}
			batchMode = true
			mkvFiles = foundFiles
			fileList.Refresh()
			fileListContainer.Show()
			outDir = folderPath
			selectedDir.SetText(outDir)
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()
			selectedFile.SetText(fmt.Sprintf("%d video files selected for batch processing", len(mkvFiles)))
			result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Found %d video files. Click 'Start Extraction' to process all files.", len(mkvFiles))))
		}, w)
	})

	clearBtn := widget.NewButton(T("extract.clear_files"), func() {
		batchMode = false
		mkvFiles = []string{}
		mkvPath = ""
		fileList.Refresh()
		fileListContainer.Hide()
		selectedFile.SetText(T("extract.no_file"))
		selectedDir.SetText(T("extract.no_dir"))
		trackItems = []*TrackItem{}
		trackList.Objects = nil
		trackList.Refresh()
		result.SetText("Select video file(s) to begin.")
	})

	dirBtn := widget.NewButton(T("extract.select_dir"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outDir = uri.Path()
			selectedDir.SetText(outDir)
		}, w)
	})

	// loadTracksBtn and startExtractBtn defined via extractLoadTracks / extractStartExtraction
	loadTracksBtn := widget.NewButton(T("extract.load_tracks"), nil)
	startExtractBtn := widget.NewButton(T("extract.start_extraction"), nil)

	// Load tracks handler
	loadTracksBtn.OnTapped = func() {
		extractLoadTracks(w, &mkvPath, &outDir, &trackItems, &mkvFiles, &batchMode,
			selectedFile, result, progress, currentTrackLabel, trackList, trackListScroll)
	}

	// Start extraction handler
	startExtractBtn.OnTapped = func() {
		extractStartExtraction(w, &mkvPath, &outDir, &trackItems, &mkvFiles, &batchMode,
			result, progress, currentTrackLabel, trackList)
	}

	// Create Support button with improved UX
	supportBtn := widget.NewButton("Donate ", func() {
		confirm := dialog.NewConfirm(
			"Support Subtitle Forge",
			"Your donation helps maintain and improve Subtitle Forge. Would you like to proceed to PayPal?",
			func(ok bool) {
				if ok {
					supportURL, _ := url.Parse("https://paypal.me/VenimK")
					fyne.CurrentApp().OpenURL(supportURL)
				}
			},
			w,
		)
		confirm.SetDismissText("Cancel")
		confirm.SetConfirmText("Donate")
		confirm.Show()
	})
	supportBtn.Importance = widget.HighImportance

	buttonRow := container.NewHBox(loadTracksBtn, startExtractBtn, layout.NewSpacer(), supportBtn)

	titleLabel := widget.NewLabel(fmt.Sprintf("Subtitle Forge %s", AppVersion))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	fileButtonRow := container.NewHBox(fileBtn, batchBtn, clearBtn)

	topContent := container.NewVBox(
		titleLabel,
		fileButtonRow,
		selectedFile,
		fileListContainer,
		dirBtn,
		selectedDir,
		buttonRow,
		currentTrackLabel,
		progress,
	)

	selectAllBtn := widget.NewButton("Select All", func() {
		for _, t := range trackItems {
			t.Check.SetChecked(true)
		}
	})
	deselectAllBtn := widget.NewButton("Deselect All", func() {
		for _, t := range trackItems {
			t.Check.SetChecked(false)
		}
	})

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter tracks by language, codec, name, or filename...")

	filterTracks := func(filterText string) {
		trackList.Objects = nil
		tracksToShow := make([]*TrackItem, len(trackItems))
		copy(tracksToShow, trackItems)
		if filterText == "" {
			for _, t := range tracksToShow {
				var trackInfoText string
				if t.FilePath != "" {
					filename := filepath.Base(t.FilePath)
					trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
				} else {
					trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
				}
				trackInfo := widget.NewLabel(trackInfoText)
				var row *fyne.Container
				if t.ConvertOCR != nil {
					ocrLabel := widget.NewLabel("Convert to SRT")
					if t.LangSelect != nil {
						langLabel := widget.NewLabel("OCR Language:")
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
					} else {
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
					}
				} else {
					row = container.NewHBox(t.Check, t.Status, trackInfo)
				}
				trackList.Add(row)
			}
		} else {
			lowerFilter := strings.ToLower(filterText)
			for _, t := range trackItems {
				matchesFilter := strings.Contains(strings.ToLower(t.Lang), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Codec), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Name), lowerFilter) ||
					strings.Contains(strings.ToLower(fmt.Sprintf("Track %d", t.Num)), lowerFilter) ||
					strings.Contains(strings.ToLower(filepath.Base(t.FilePath)), lowerFilter)
				if matchesFilter {
					var trackInfoText string
					if t.FilePath != "" {
						filename := filepath.Base(t.FilePath)
						trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
					} else {
						trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
					}
					trackInfo := widget.NewLabel(trackInfoText)
					var row *fyne.Container
					if t.ConvertOCR != nil {
						ocrLabel := widget.NewLabel("Convert to SRT")
						if t.LangSelect != nil {
							langLabel := widget.NewLabel("OCR Language:")
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
						} else {
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
						}
					} else {
						row = container.NewHBox(t.Check, t.Status, trackInfo)
					}
					trackList.Add(row)
				}
			}
		}
		trackList.Refresh()
	}

	filterEntry.OnChanged = func(text string) {
		filterTracks(text)
	}

	sortSelect := widget.NewSelect([]string{
		"Default Order",
		"By Filename",
		"By Language",
		"By Codec",
		"By Track Number",
	}, func(value string) {
		filterTracks(filterEntry.Text)
	})
	sortSelect.SetSelected("Default Order")

	// Now update filterTracks to include sorting
	filterTracks = func(filterText string) {
		trackList.Objects = nil
		var tracksToShow []*TrackItem
		if filterText == "" {
			tracksToShow = make([]*TrackItem, len(trackItems))
			copy(tracksToShow, trackItems)
		} else {
			lowerFilter := strings.ToLower(filterText)
			for _, t := range trackItems {
				matchesFilter := strings.Contains(strings.ToLower(t.Lang), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Codec), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Name), lowerFilter) ||
					strings.Contains(strings.ToLower(fmt.Sprintf("Track %d", t.Num)), lowerFilter) ||
					strings.Contains(strings.ToLower(filepath.Base(t.FilePath)), lowerFilter)
				if matchesFilter {
					tracksToShow = append(tracksToShow, t)
				}
			}
		}
		if sortSelect.Selected != "Default Order" {
			switch sortSelect.Selected {
			case "By Filename":
				sort.Slice(tracksToShow, func(i, j int) bool {
					filenameI := filepath.Base(tracksToShow[i].FilePath)
					filenameJ := filepath.Base(tracksToShow[j].FilePath)
					if filenameI == filenameJ {
						return tracksToShow[i].Num < tracksToShow[j].Num
					}
					return filenameI < filenameJ
				})
			case "By Language":
				sort.Slice(tracksToShow, func(i, j int) bool {
					if tracksToShow[i].Lang == tracksToShow[j].Lang {
						filenameI := filepath.Base(tracksToShow[i].FilePath)
						filenameJ := filepath.Base(tracksToShow[j].FilePath)
						if filenameI == filenameJ {
							return tracksToShow[i].Num < tracksToShow[j].Num
						}
						return filenameI < filenameJ
					}
					return tracksToShow[i].Lang < tracksToShow[j].Lang
				})
			case "By Codec":
				sort.Slice(tracksToShow, func(i, j int) bool {
					if tracksToShow[i].Codec == tracksToShow[j].Codec {
						filenameI := filepath.Base(tracksToShow[i].FilePath)
						filenameJ := filepath.Base(tracksToShow[j].FilePath)
						if filenameI == filenameJ {
							return tracksToShow[i].Num < tracksToShow[j].Num
						}
						return filenameI < filenameJ
					}
					return tracksToShow[i].Codec < tracksToShow[j].Codec
				})
			case "By Track Number":
				sort.Slice(tracksToShow, func(i, j int) bool {
					if tracksToShow[i].Num == tracksToShow[j].Num {
						filenameI := filepath.Base(tracksToShow[i].FilePath)
						filenameJ := filepath.Base(tracksToShow[j].FilePath)
						return filenameI < filenameJ
					}
					return tracksToShow[i].Num < tracksToShow[j].Num
				})
			}
		}
		for _, t := range tracksToShow {
			var trackInfoText string
			if t.FilePath != "" {
				filename := filepath.Base(t.FilePath)
				trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
			} else {
				trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
			}
			trackInfo := widget.NewLabel(trackInfoText)
			var row *fyne.Container
			if t.ConvertOCR != nil {
				ocrLabel := widget.NewLabel("Convert to SRT")
				if t.LangSelect != nil {
					langLabel := widget.NewLabel("OCR Language:")
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
				} else {
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
				}
			} else {
				row = container.NewHBox(t.Check, t.Status, trackInfo)
			}
			trackList.Add(row)
		}
		trackList.Refresh()
	}

	filterEntry.SetPlaceHolder("Filter tracks by language, codec, name, filename, or track number...                                                 ")
	filterBox := container.New(
		layout.NewFormLayout(),
		widget.NewLabel("Filter:"),
		filterEntry,
	)
	sortBox := container.New(
		layout.NewFormLayout(),
		widget.NewLabel("Sort by:"),
		sortSelect,
	)
	trackControlsContainer := container.NewVBox(
		container.NewHBox(selectAllBtn, deselectAllBtn),
		filterBox,
		sortBox,
	)

	middleContent := container.NewVBox(
		widget.NewLabel("Subtitle Tracks:"),
		trackControlsContainer,
		trackListScroll,
	)

	bottomContent := container.NewVBox(
		widget.NewLabel("Results:"),
		resultScroll,
		dependencyButtons,
	)

	extractTabContent := container.NewBorder(
		topContent,
		bottomContent,
		nil,
		nil,
		middleContent,
	)

	// Build the drag-and-drop handler
	onDropped := func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		var mkvUris []fyne.URI
		for _, uri := range uris {
			filePath := uri.Path()
			if IsVideoFile(filePath) {
				mkvUris = append(mkvUris, uri)
			}
		}
		if len(mkvUris) == 0 {
			a.SendNotification(&fyne.Notification{
				Title:   "Invalid Files",
				Content: "Please drop MKV or MP4 files.",
			})
			return
		}
		if len(mkvUris) == 1 {
			filePath := mkvUris[0].Path()
			batchMode = false
			mkvFiles = []string{}
			fileList.Refresh()
			fileListContainer.Hide()
			mkvPath = filePath
			a.SendNotification(&fyne.Notification{
				Title:   "Video File Dropped",
				Content: "Video file loaded: " + filepath.Base(filePath),
			})
			selectedFile.SetText(mkvPath)
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()
			result.SetText(setLogMessage(LogInfo, "Video File Loaded", "Video file dropped and loaded. Output directory automatically set to file location. Click 'Load Tracks' to analyze the file."))
		} else {
			batchMode = true
			mkvFiles = []string{}
			for _, uri := range mkvUris {
				mkvFiles = append(mkvFiles, uri.Path())
			}
			fileList.Refresh()
			fileListContainer.Show()
			a.SendNotification(&fyne.Notification{
				Title:   "Multiple Video Files Dropped",
				Content: fmt.Sprintf("%d video files loaded for batch processing", len(mkvFiles)),
			})
			if len(mkvFiles) > 0 {
				outDir = filepath.Dir(mkvFiles[0])
				selectedDir.SetText(outDir)
			}
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()
			selectedFile.SetText(fmt.Sprintf("%d video files selected for batch processing", len(mkvFiles)))
			result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Dropped %d video files. Click 'Load Tracks' to analyze all files and select which tracks to extract.", len(mkvFiles))))
		}
	}

	widgets := &ExtractTabWidgets{
		DependencyButtons: dependencyButtons,
		OnDropped:         onDropped,
		FileOpenFunc:      fileBtn.OnTapped,
		DirChangeFunc:     dirBtn.OnTapped,
		LoadTracksFunc:    loadTracksBtn.OnTapped,
		StartExtractFunc:  startExtractBtn.OnTapped,
	}

	return extractTabContent, widgets
}
