package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TrackItem represents a subtitle track with UI elements
type TrackItem struct {
	Num        int
	Lang       string
	Codec      string
	Name       string
	State      string
	FilePath   string // Source MKV file path (for batch processing)
	Check      *widget.Check
	Status     *widget.Label
	ConvertOCR *widget.Check  // Option to convert PGS to SRT using OCR
	LangSelect *widget.Select // Language selection dropdown for OCR
}

// Global debug logger for dependency checks
var debugLogger *os.File

// ============================================================================
// Application Logging System
// ============================================================================

var (
	appLogFile        *os.File
	appLogger         *log.Logger
	appLogPath        string
	appLogBuffer      strings.Builder
	appLogMutex       sync.Mutex
	logUpdateCallback func() // Callback to update UI when new log entry is added
)

// initAppLogger initializes the application logging system
func initAppLogger() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("[WARN] Could not get home directory for logging:", err)
		return
	}

	logDir := filepath.Join(homeDir, ".subtitle-forge", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Println("[WARN] Could not create log directory:", err)
		return
	}

	appLogPath = filepath.Join(logDir, fmt.Sprintf("subtitle_forge_%s.log", time.Now().Format("20060102_150405")))
	f, err := os.OpenFile(appLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("[WARN] Could not create log file:", err)
		appLogPath = ""
		return
	}

	appLogFile = f
	appLogger = log.New(appLogFile, "", log.LstdFlags)

	// Log startup info
	AppLog("INFO", "Subtitle Forge started")
	AppLog("INFO", "Log file: %s", appLogPath)
	AppLog("INFO", "Working Directory: %s", getCurrentDir())
	AppLog("INFO", "Executable: %s", getExecutablePath())
}

// closeAppLogger closes the application log file
func closeAppLogger() {
	if appLogger != nil {
		AppLog("INFO", "Subtitle Forge exiting")
	}
	if appLogFile != nil {
		appLogFile.Close()
		appLogFile = nil
	}
}

// AppLog logs a message to both file and in-memory buffer for UI display
func AppLog(level string, format string, args ...any) {
	appLogMutex.Lock()
	defer appLogMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	// Write to file if available
	if appLogger != nil {
		appLogger.Printf("[%s] %s", level, message)
	}

	// Write to in-memory buffer for UI
	appLogBuffer.WriteString(logLine)

	// Trigger UI update callback if set
	if logUpdateCallback != nil {
		go func() {
			fyne.Do(func() {
				logUpdateCallback()
			})
		}()
	}
}

// AppLogCmd logs an external command execution with its result
func AppLogCmd(cmd *exec.Cmd, output []byte, err error) {
	if cmd == nil {
		return
	}

	cmdStr := strings.Join(cmd.Args, " ")
	AppLog("CMD", "Execute: %s", cmdStr)

	if cmd.Dir != "" {
		AppLog("CMD", "Working dir: %s", cmd.Dir)
	}

	if err != nil {
		AppLog("ERROR", "Command failed: %v", err)
	}

	if len(output) > 0 {
		// Truncate very long output
		outputStr := string(output)
		if len(outputStr) > 5000 {
			outputStr = outputStr[:5000] + "\n... [truncated]"
		}
		AppLog("OUTPUT", "%s", outputStr)
	}
}

// GetLogBuffer returns the current log buffer contents
func GetLogBuffer() string {
	appLogMutex.Lock()
	defer appLogMutex.Unlock()
	return appLogBuffer.String()
}

// ClearLogBuffer clears the in-memory log buffer (not the file)
func ClearLogBuffer() {
	appLogMutex.Lock()
	defer appLogMutex.Unlock()
	appLogBuffer.Reset()
	AppLog("INFO", "Log buffer cleared")
}

// GetLogPath returns the current log file path
func GetLogPath() string {
	return appLogPath
}

// SetLogUpdateCallback sets the callback function for UI updates
func SetLogUpdateCallback(callback func()) {
	logUpdateCallback = callback
}

// createLogsTab creates the dedicated Logs tab
func createLogsTab(w fyne.Window) *fyne.Container {
	// Title
	logsTitle := widget.NewLabel("Application Logs")
	logsTitle.TextStyle = fyne.TextStyle{Bold: true}
	logsTitle.Alignment = fyne.TextAlignCenter

	// Log file path display
	logPathLabel := widget.NewLabel("Log file: (initializing...)")
	logPathLabel.Wrapping = fyne.TextWrapWord
	if appLogPath != "" {
		logPathLabel.SetText("Log file: " + appLogPath)
	}

	// Log viewer (text area - kept enabled for clear text visibility)
	logViewer := widget.NewMultiLineEntry()
	logViewer.Wrapping = fyne.TextWrapWord
	logViewer.SetText(GetLogBuffer())

	logViewerScroll := container.NewScroll(logViewer)
	logViewerScroll.SetMinSize(fyne.NewSize(800, 400))

	// Set up callback to update log viewer when new entries are added
	SetLogUpdateCallback(func() {
		logViewer.SetText(GetLogBuffer())
		// Scroll to bottom
		logViewerScroll.ScrollToBottom()
	})

	// Buttons
	copyPathBtn := widget.NewButton("Copy Log Path", func() {
		if appLogPath == "" {
			dialog.ShowInformation("Not Available", "Log file is not available", w)
			return
		}
		w.Clipboard().SetContent(appLogPath)
		dialog.ShowInformation("Copied", "Log path copied to clipboard", w)
	})
	copyPathBtn.Importance = widget.MediumImportance

	openFolderBtn := widget.NewButton("Open Log Folder", func() {
		if appLogPath == "" {
			dialog.ShowInformation("Not Available", "Log file is not available", w)
			return
		}
		logDir := filepath.Dir(appLogPath)
		// Use open command on macOS
		cmd := exec.Command("open", logDir)
		if err := cmd.Run(); err != nil {
			dialog.ShowError(fmt.Errorf("Could not open folder: %v", err), w)
		}
	})
	openFolderBtn.Importance = widget.MediumImportance

	refreshBtn := widget.NewButton("Refresh", func() {
		logViewer.SetText(GetLogBuffer())
		logViewerScroll.ScrollToBottom()
	})
	refreshBtn.Importance = widget.LowImportance

	clearBufferBtn := widget.NewButton("Clear Display", func() {
		ClearLogBuffer()
		logViewer.SetText(GetLogBuffer())
	})
	clearBufferBtn.Importance = widget.LowImportance

	copyLogsBtn := widget.NewButton("Copy All Logs", func() {
		logs := GetLogBuffer()
		if logs == "" {
			dialog.ShowInformation("Empty", "No logs to copy", w)
			return
		}
		w.Clipboard().SetContent(logs)
		dialog.ShowInformation("Copied", "All logs copied to clipboard", w)
	})
	copyLogsBtn.Importance = widget.MediumImportance

	// Button row
	buttonRow := container.NewHBox(
		copyPathBtn,
		openFolderBtn,
		layout.NewSpacer(),
		refreshBtn,
		clearBufferBtn,
		copyLogsBtn,
	)

	// Info section
	infoLabel := widget.NewLabel("Logs are automatically saved to disk. Share the log file when reporting issues.")
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Assemble the tab
	logsTabContent := container.NewVBox(
		container.NewPadded(logsTitle),
		logPathLabel,
		widget.NewSeparator(),
		buttonRow,
		widget.NewSeparator(),
		logViewerScroll,
		infoLabel,
	)

	return logsTabContent
}

// NOTE: Dependency checking and installation functions have been moved to dependency.go

// setLogMessage formats a log message with an icon and title.
func setLogMessage(logType, title, message string) string {
	var icon string
	switch logType {
	case LogInfo:
		icon = "ℹ️"
	case LogSuccess:
		icon = "✅"
	case LogError:
		icon = "❌"
	case LogExtract:
		icon = "🎬"
	case LogConvert:
		icon = "🔄"
	default:
		icon = "➡️"
	}
	return fmt.Sprintf("%s %s\n%s", icon, title, message)
}

func main() {
	// Initialize logging system
	initAppLogger()
	defer closeAppLogger()

	trackList := container.NewVBox()
	// Create a scrollable container for the track list
	trackListScroll := container.NewScroll(trackList)
	// Set a minimum size for the track list scroll area to show more tracks
	trackListScroll.SetMinSize(fyne.NewSize(850, 250))

	// Create app with explicit ID and set metadata directly
	a := app.NewWithID("com.gmm.subtitleforge")
	a.SetIcon(theme.FileTextIcon())

	// Apply theme based on saved preference
	savedTheme := a.Preferences().StringWithFallback("theme", "Dark Theme")
	ApplyThemeByName(a, savedTheme)

	// Create main window with explicit name
	w := a.NewWindow("Subtitle Forge")
	// Set app metadata on window
	w.SetMaster()
	w.CenterOnScreen()
	// Ensure window has adequate size
	// In Fyne, windows are resizable by default unless explicitly set as fixed size
	w.Resize(fyne.NewSize(1024, 768))
	// Explicitly ensure the window is not fixed size
	w.SetFixedSize(false)

	// Setup keyboard shortcuts
	setupKeyboardShortcuts := func(fileOpenFunc, dirChangeFunc, loadTracksFunc, startExtractFunc func()) {
		// Ctrl+O for opening files
		ctrlO := &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlO, func(shortcut fyne.Shortcut) {
			fileOpenFunc()
		})

		// Ctrl+D for changing directory
		ctrlD := &desktop.CustomShortcut{KeyName: fyne.KeyD, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlD, func(shortcut fyne.Shortcut) {
			dirChangeFunc()
		})

		// Ctrl+L for loading tracks
		ctrlL := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlL, func(shortcut fyne.Shortcut) {
			loadTracksFunc()
		})

		// Ctrl+E for starting extraction
		ctrlE := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlE, func(shortcut fyne.Shortcut) {
			startExtractFunc()
		})
	}

	// Load window size from preferences or use default size
	defaultWidth := float32(900)
	defaultHeight := float32(700)
	width := float32(a.Preferences().Float("window_width"))
	height := float32(a.Preferences().Float("window_height"))

	if width == 0 || height == 0 {
		// Use default size for first launch
		width = defaultWidth
		height = defaultHeight
	}

	// Resize window to saved or default size
	w.Resize(fyne.NewSize(width, height))

	// Save window size when it changes
	// Use a timer to periodically check and save window size
	var lastSize fyne.Size
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fyne.Do(func() {
				currentSize := w.Canvas().Size()
				// Only save if size has changed
				if currentSize.Width != lastSize.Width || currentSize.Height != lastSize.Height {
					a.Preferences().SetFloat("window_width", float64(currentSize.Width))
					a.Preferences().SetFloat("window_height", float64(currentSize.Height))
					lastSize = currentSize
				}
			})
		}
	}()

	// Also save window size when closing
	w.SetCloseIntercept(func() {
		// Save current window size
		currentSize := w.Canvas().Size()
		a.Preferences().SetFloat("window_width", float64(currentSize.Width))
		a.Preferences().SetFloat("window_height", float64(currentSize.Height))

		// Close the window
		w.Close()
	})

	// Check dependencies at startup
	dependencyResults := checkDependencies()

	var mkvPath string
	var outDir string
	var trackItems []*TrackItem

	// Global variables for batch processing
	var mkvFiles []string
	var batchMode bool

	selectedFile := widget.NewLabel("No video file selected.")
	selectedDir := widget.NewLabel("No output directory selected.")
	result := widget.NewLabel("Results will appear here...")
	result.Wrapping = fyne.TextWrapWord
	// Make the result area larger to show more debug information
	resultScroll := container.NewScroll(result)
	resultScroll.SetMinSize(fyne.NewSize(780, 200))

	// Set up file drop handling
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		// Handle key events if needed
	})

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			filePath := uris[0].Path()

			if IsVideoFile(filePath) {
				// Handle video file drop (MKV, MP4, M4V)
				mkvPath = filePath
				a.SendNotification(&fyne.Notification{
					Title:   "File Dropped",
					Content: "Video file loaded: " + filepath.Base(filePath),
				})

				// Update UI
				selectedFile.SetText(mkvPath)

				// Set output directory to the same directory as the video file
				outDir = filepath.Dir(mkvPath)
				selectedDir.SetText(outDir)

				// Clear previous tracks
				trackItems = []*TrackItem{}
				trackList.Objects = nil
				trackList.Refresh()

				result.SetText(setLogMessage(LogInfo, "Video File Loaded", "Video file dropped and loaded. Output directory automatically set to file location. Click 'Load Tracks' to analyze the file."))
			} else {
				a.SendNotification(&fyne.Notification{
					Title:   "Invalid File",
					Content: "Please drop an MKV or MP4 file.",
				})
			}
		}
	})

	// Display dependency check results
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

	// Create a container for dependency-related buttons
	dependencyButtons := container.NewVBox()

	// Create a container for the install all button
	installAllContainer := container.NewHBox()

	// Create a list of missing dependencies
	missingDependencies := []string{}
	for tool, installed := range dependencyResults {
		if !installed {
			missingDependencies = append(missingDependencies, tool)
		}
	}

	// Add individual install buttons for each missing dependency
	if len(missingDependencies) > 0 {
		// Add header for install buttons
		dependencyButtons.Add(widget.NewLabel("Install Missing Dependencies:"))

		// Add buttons for each missing dependency
		for _, tool := range missingDependencies {
			// Create a local copy of the tool name for the closure
			toolName := tool

			// Create button with appropriate label
			buttonLabel := fmt.Sprintf("Install %s", toolName)
			installButton := widget.NewButton(buttonLabel, func() {
				installDependency(w, toolName)
			})

			// Add the install button to the dependency buttons container
			dependencyButtons.Add(installButton)
		}

		// Add an "Install All" button if there are multiple missing dependencies
		if len(missingDependencies) > 1 {
			installAllButton := widget.NewButton("Install All Missing Dependencies", func() {
				// Show confirmation dialog
				dialog.ShowConfirm("Install All Dependencies",
					"This will attempt to install all missing dependencies.\n\nSome installations may require sudo privileges.\n\nDo you want to continue?",
					func(confirmed bool) {
						if confirmed {
							// Create a simple progress dialog
							progress := dialog.NewProgress("Installing Dependencies", "Installing missing dependencies...", w)
							progress.Show()

							// Run installations in a goroutine
							go func() {
								totalTools := len(missingDependencies)
								successCount := 0
								failureCount := 0

								// Install each tool
								for i, tool := range missingDependencies {
									// Update progress value - increment for each tool
									progressValue := float64(i) / float64(totalTools)
									progress.SetValue(progressValue)

									// Prepare the installation command based on the tool
									var cmd *exec.Cmd
									switch tool {
									case "mkvmerge", "mkvextract":
										// MKVToolNix includes both mkvmerge and mkvextract
										cmd = exec.Command("brew", "install", "mkvtoolnix")
									case "deno":
										cmd = exec.Command("brew", "install", "deno")
									case "tesseract":
										cmd = exec.Command("brew", "install", "tesseract")
									case "ffmpeg":
										cmd = exec.Command("brew", "install", "ffmpeg")
									case "vobsub2srt":
										// Get the script path relative to the executable
										execPath, err := os.Executable()
										if err != nil {
											fmt.Println("[ERROR] Failed to get executable path:", err)
											execPath = "."
										}
										execDir := filepath.Dir(execPath)
										scriptPath := filepath.Join(execDir, "install_vobsub2srt.sh")
										cmd = exec.Command("bash", scriptPath)
									case "pgsrip":
										// Get the script path relative to the executable
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

									// Run the installation command
									_, err := cmd.CombinedOutput()
									if err != nil {
										fmt.Printf("[ERROR] Failed to install %s: %v\n", tool, err)
										failureCount++
									} else {
										successCount++
									}
								}

								// Hide the progress dialog
								progress.Hide()

								// Show results
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

								// Update the dependency status
								updateDependencyStatus(w)
							}()
						}
					}, w)
			})

			// Add the install all button to the container
			installAllContainer.Add(installAllButton)
			dependencyButtons.Add(installAllContainer)
		}
	}

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)

	currentTrackLabel := widget.NewLabel("")

	// Create file list widget for batch processing
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
				// Remove file from list
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

	// Create file list container with scroll
	fileListContainer := container.NewBorder(
		widget.NewLabel("Selected Files:"),
		nil, nil, nil,
		container.NewScroll(fileList),
	)
	fileListContainer.Hide() // Initially hidden

	// Button to select single video file
	fileBtn := widget.NewButton("Select Video File", func() {
		// Create a file filter for video files
		filter := storage.NewExtensionFileFilter(VideoFileExtensions())

		// Create a file dialog with explicit styling for readability
		fd := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				return
			}

			filePath := file.URI().Path()

			// Double-check that it's a supported video file
			if !IsVideoFile(filePath) {
				dialog.ShowError(fmt.Errorf("Please select a supported video file (MKV, MP4, M4V)."), w)
				return
			}

			// Reset to single file mode
			batchMode = false
			mkvFiles = []string{}
			fileList.Refresh()
			fileListContainer.Hide()

			mkvPath = filePath
			selectedFile.SetText(mkvPath)

			// Set output directory to the same directory as the video file
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)

			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()

			result.SetText(setLogMessage(LogInfo, "Video File Loaded", "Video file loaded. Output directory automatically set to file location. Click 'Load Tracks' to analyze the file."))
		}, w)

		fd.SetFilter(filter)
		fd.Show()
	})

	// Button to select multiple video files for batch processing
	batchBtn := widget.NewButton("Select Multiple Video Files (Batch)", func() {
		// Use folder selection for batch processing
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			folderPath := uri.Path()

			// Find all supported video files in the selected folder
			var foundFiles []string
			err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Continue walking
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

			// Set batch mode
			batchMode = true
			mkvFiles = foundFiles
			fileList.Refresh()
			fileListContainer.Show()

			// Set output directory to the selected folder
			outDir = folderPath
			selectedDir.SetText(outDir)

			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()

			selectedFile.SetText(fmt.Sprintf("%d video files selected for batch processing", len(mkvFiles)))
			result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Found %d video files. Click 'Start Extraction' to process all files.", len(mkvFiles))))

		}, w)
	})

	// Button to clear file selection
	clearBtn := widget.NewButton("Clear Selection", func() {
		batchMode = false
		mkvFiles = []string{}
		mkvPath = ""
		fileList.Refresh()
		fileListContainer.Hide()
		selectedFile.SetText("No files selected")
		selectedDir.SetText("No output directory selected")
		trackItems = []*TrackItem{}
		trackList.Objects = nil
		trackList.Refresh()
		result.SetText("Select video file(s) to begin.")
	})

	// Button to select output directory (optional, as it's auto-set)
	dirBtn := widget.NewButton("Change Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			outDir = uri.Path()
			selectedDir.SetText(outDir)
		}, w)
	})

	// Button to load tracks from video file
	loadTracksBtn := widget.NewButton("Load Tracks", func() {
		if batchMode {
			// In batch mode, load tracks from all files for user selection
			if len(mkvFiles) == 0 {
				dialog.ShowError(fmt.Errorf("Please select video files for batch processing first."), w)
				return
			}

			// Load tracks from all video files
			go func() {
				fyne.Do(func() {
					result.SetText(setLogMessage(LogInfo, "Loading Batch Tracks", "Analyzing all video files for subtitle tracks..."))
					progress.Max = float64(len(mkvFiles))
					progress.SetValue(0)
				})

				// Clear previous tracks
				trackItems = []*TrackItem{}

				totalTracks := 0
				for fileIndex, videoFile := range mkvFiles {
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Analyzing file %d/%d: %s", fileIndex+1, len(mkvFiles), filepath.Base(videoFile)))
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

							trackItems = append(trackItems, trackItem)
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

								trackItems = append(trackItems, trackItem)
								totalTracks++
							}
						}
					}
				}

				// Update UI with all tracks
				fyne.Do(func() {
					progress.SetValue(float64(len(mkvFiles)))
					currentTrackLabel.SetText(fmt.Sprintf("Found %d subtitle tracks across %d files", totalTracks, len(mkvFiles)))

					// Update track list
					trackList.Objects = nil
					for _, tt := range trackItems {
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

					result.SetText(setLogMessage(LogSuccess, "Batch Tracks Loaded", fmt.Sprintf("Found %d subtitle tracks across %d files. Select the tracks you want to extract, then click 'Start Extraction'.", totalTracks, len(mkvFiles))))
				})
			}()
			return
		}

		// Single file mode
		if mkvPath == "" {
			dialog.ShowError(fmt.Errorf("Please select or drag & drop a video file first."), w)
			return
		}

		// Detect container type and load tracks accordingly
		if IsMP4File(mkvPath) {
			// Use ffprobe for MP4/M4V files
			mp4Tracks, err := LoadMP4SubtitleTracks(mkvPath)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to analyze MP4 file: %v", err), w)
				return
			}

			trackItems = []*TrackItem{}
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

				trackItems = append(trackItems, t)

				trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", mt.Index, mt.Language, MP4CodecDisplayName(mt.Codec), mt.TrackName))

				var row *fyne.Container
				if t.ConvertOCR != nil {
					ocrLabel := widget.NewLabel("Convert to SRT")
					row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel)
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
		cmd := NewMkvmergeCmd("-J", mkvPath)
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
		trackItems = []*TrackItem{}
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

			trackItems = append(trackItems, t)

			// Create row for this track
			trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", trackID, trackLang, trackCodec, trackName))

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
			} else {
				// For other subtitle formats
				row = container.NewHBox(check, status, trackInfo)
			}

			trackList.Add(row)
		}
		trackList.Refresh()

		result.SetText(setLogMessage(LogSuccess, "Tracks Loaded", "Tracks loaded. Select the tracks you want to extract, then click 'Start Extraction'"))
	})

	// Button to start extraction of selected tracks
	startExtractBtn := widget.NewButton("Start Extraction", func() {
		if batchMode {
			// Batch processing mode
			if len(mkvFiles) == 0 || outDir == "" {
				dialog.ShowError(fmt.Errorf("Please select video files and output directory for batch processing."), w)
				return
			}

			// Start batch processing
			go func() {
				totalFiles := len(mkvFiles)
				successCount := 0
				failureCount := 0

				fyne.Do(func() {
					result.SetText(setLogMessage(LogInfo, "Batch Processing Started", fmt.Sprintf("Processing %d video files...", totalFiles)))
					progress.Max = float64(totalFiles)
					progress.SetValue(0)
				})

				for fileIndex, currentMkvPath := range mkvFiles {
					mkvPath = currentMkvPath // Set current file for processing

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
					for _, t := range trackItems {
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
						outPath := filepath.Join(outDir, outFile)

						var output []byte
						var err error

						if IsMP4File(currentMkvPath) {
							// Use ffmpeg for MP4/M4V files
							output, err = ExtractMP4Subtitle(currentMkvPath, track.Num, outPath)
						} else {
							// Use mkvextract for MKV files
							extractCmd := NewMkvextractCmd("tracks", currentMkvPath, fmt.Sprintf("%d:%s", track.Num, outFile))
							extractCmd.Dir = outDir

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
					currentTrackLabel.SetText("Batch processing completed")
					result.SetText(result.Text + fmt.Sprintf("\n\n🎬 Batch Processing Complete\n✅ Success: %d files\n❌ Failed: %d files\n📁 Output: %s", successCount, failureCount, outDir))
				})

				// Show completion notification
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Batch Processing Complete",
					Content: fmt.Sprintf("Processed %d files. %d successful, %d failed.", totalFiles, successCount, failureCount),
				})
			}()
			return
		}

		// Single file processing mode
		if mkvPath == "" || outDir == "" {
			dialog.ShowError(fmt.Errorf("Please select both a video file and output directory."), w)
			return
		}

		go func() {
			selected := []*TrackItem{}
			for _, t := range trackItems {
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
				mkvBaseName := filepath.Base(mkvPath)
				mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))

				// Check if this is a PGS track with OCR conversion requested
				if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS") {
					// First extract as PGS
					fyne.Do(func() {
						result.SetText(setLogMessage(LogConvert, "PGS to SRT Conversion", "Starting PGS extraction process..."))
					})
					tempPgsFile := fmt.Sprintf("%s.track%d_%s.sup", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absPgsPath := filepath.Join(outDir, tempPgsFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting PGS track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== PGS Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("PGS file: %s\n", tempPgsFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absPgsPath))
					})

					// Extract PGS first
					var cmd *exec.Cmd
					if IsMP4File(mkvPath) {
						// Use ffmpeg for MP4/M4V files
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracting PGS track using ffmpeg...")
						})
						output, err = ExtractMP4Subtitle(mkvPath, t.Num, absPgsPath)
					} else {
						// Use mkvextract for MKV files
						cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempPgsFile)
						fyne.Do(func() {
							result.SetText(result.Text + "\nRunning: " + cmdStr)
						})

						cmd = NewMkvextractCmd("tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
						AppLog("DEBUG", "PGS track extraction using mkvextract")
						cmd.Dir = outDir

						AppLog("EXTRACT", "PGS extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(mkvPath))
						output, err = cmd.CombinedOutput()
						AppLogCmd(cmd, output, err)
					}

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					pgsFilePath := filepath.Join(outDir, tempPgsFile)
					fileInfo, statErr := os.Stat(pgsFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty (0 bytes)")
					} else {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted PGS file (%d bytes)", fileInfo.Size()))
						})
					}

					if err == nil {
						// Debug point after successful extraction
						// Create a detailed progress bar for the conversion process
						conversionProgress := widget.NewProgressBar()
						conversionProgress.Min = 0
						conversionProgress.Max = 100 // Percentage-based progress
						conversionProgress.SetValue(0)

						conversionLabel := widget.NewLabel("Converting PGS to SRT...")
						statusLabel := widget.NewLabel("Initializing OCR process...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Estimated time remaining: calculating...")

						// Track conversion start time and progress data
						conversionStartTime := time.Now()
						var progressMutex sync.Mutex
						var progressData = struct {
							currentFrame int
							totalFrames  int
							frameRate    float64 // frames processed per second
							lastUpdate   time.Time
						}{
							currentFrame: 0,
							totalFrames:  0, // Will be updated when we parse output
							frameRate:    0,
							lastUpdate:   time.Now(),
						}

						// Create a ticker to update elapsed time and estimated remaining time
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastElapsedText, lastRemainingText string

							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

								// Calculate estimated time remaining
								progressMutex.Lock()
								currentFrame := progressData.currentFrame
								totalFrames := progressData.totalFrames
								frameRate := progressData.frameRate
								progressMutex.Unlock()

								var newRemainingText string
								var progressValue float64

								if totalFrames > 0 && currentFrame > 0 && frameRate > 0 {
									// Calculate percentage complete
									progressValue = float64(currentFrame) / float64(totalFrames) * 100

									// Calculate remaining time
									framesRemaining := totalFrames - currentFrame
									secondsRemaining := float64(framesRemaining) / frameRate
									remaining := time.Duration(secondsRemaining * float64(time.Second))
									remaining = remaining.Round(time.Second)

									newRemainingText = fmt.Sprintf("Estimated time remaining: %s", remaining)
								} else {
									newRemainingText = "Estimated time remaining: calculating..."
									progressValue = 0
								}

								// Only update UI if text has changed to reduce UI operations
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

							// Show the conversion progress bar and labels
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

						// Try to use pgsrip first, fall back to pgs-to-srt if not available
						// Get language from user selection or use track language as default
						langCode := "eng" // Default to English
						if t.Lang != "" {
							langCode = t.Lang
						}

						// Check if user has selected a specific language
						if t.LangSelect != nil && t.LangSelect.Selected != "" && !strings.HasPrefix(t.LangSelect.Selected, "Auto") {
							// Extract the language code from the selection (format: "Language (code)")
							selection := t.LangSelect.Selected
							// Extract the code part between parentheses
							if start := strings.LastIndex(selection, "("); start != -1 {
								if end := strings.LastIndex(selection, ")"); end != -1 && end > start {
									// Extract the 2-letter code
									twoLetterCode := selection[start+1 : end]
									fyne.Do(func() {
										result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] User selected OCR language: %s (code: %s)", selection, twoLetterCode))
									})

									// Map 2-letter code to 3-letter code for Tesseract
									langCodeMap := map[string]string{
										"en": "eng", // English
										"fr": "fra", // French
										"de": "deu", // German
										"it": "ita", // Italian
										"es": "spa", // Spanish
										"pt": "por", // Portuguese
										"nl": "nld", // Dutch
										"sv": "swe", // Swedish
										"no": "nor", // Norwegian
										"da": "dan", // Danish
										"fi": "fin", // Finnish
										"ja": "jpn", // Japanese
										"ko": "kor", // Korean
										"zh": "chi", // Chinese
										"ru": "rus", // Russian
										"pl": "pol", // Polish
										"cs": "ces", // Czech
										"hu": "hun", // Hungarian
										"el": "ell", // Greek
										"tr": "tur", // Turkish
										"ar": "ara", // Arabic
										"he": "heb", // Hebrew
										"th": "tha", // Thai
									}

									// Convert 2-letter code to 3-letter code if a mapping exists
									if threeLetterCode, exists := langCodeMap[twoLetterCode]; exists {
										langCode = threeLetterCode
										fyne.Do(func() {
											result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code for OCR: %s -> %s", twoLetterCode, langCode))
										})
									} else {
										// If no mapping exists, use the 2-letter code directly
										langCode = twoLetterCode
										fyne.Do(func() {
											result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code as-is for OCR: %s", langCode))
										})
									}
								}
							}
						}

						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempPgsFile)
						absOutputPath := filepath.Join(outDir, outFile)

						// Check if pgsrip is available and use it if possible
						pgsripAvailable := checkPgsrip()
						if pgsripAvailable {
							fyne.Do(func() {
								result.SetText(result.Text + "\n\n=== Using pgsrip for conversion ===\n")
								statusLabel.SetText("Starting pgsrip conversion...")
							})

							// Set up conversion settings - simplified
							convSettings := PgsConversionSettings{
								Verbose: true, // Enable verbose logging
							}

							// Call our pgsrip conversion function
							err = convertPgsWithPgsrip(absInputPath, absOutputPath, langCode, result, statusLabel, conversionProgress, convSettings)
							if err == nil {
								fyne.Do(func() {
									result.SetText(result.Text + "\n\n✅ PGS to SRT conversion with pgsrip completed successfully!")
									statusLabel.SetText("Conversion complete!")
									conversionProgress.SetValue(100)
								})
								return
							} else {
								fyne.Do(func() {
									result.SetText(result.Text + "\n⚠️ pgsrip conversion failed: " + err.Error() + "\nFalling back to pgs-to-srt...")
								})
								// Fall back to pgs-to-srt
							}
						}

						// Fall back to pgs-to-srt if pgsrip not available or failed
						// Use the configured PGS-to-SRT script with Deno
						pgsToSrtScript := pgsToSrtScriptPath

						// Define the path to the trained data file with the selected language
						trainedDataPath := filepath.Join(filepath.Dir(pgsToSrtScript), "tessdata_fast", langCode+".traineddata")

						// Check if the script exists
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
						})

						// Test if Deno is working correctly
						fyne.Do(func() {
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

						// Show detailed file information
						// Build text updates in memory before applying to UI
						textUpdate := fmt.Sprintf("\nInput SUP file: %s\nOutput SRT file: %s\nTessdata file: %s\n",
							absInputPath, absOutputPath, trainedDataPath)

						fyne.Do(func() {
							result.SetText(result.Text + textUpdate)

							// Check if input file exists and show size
							if fileInfo, err := os.Stat(absInputPath); err == nil {
								result.SetText(result.Text + fmt.Sprintf("Input file size: %d bytes\n", fileInfo.Size()))
							} else {
								result.SetText(result.Text + fmt.Sprintf("Input file check error: %v\n", err))
							}
						})

						// Variables to track file copy status
						var copyErr error
						var copySuccess bool

						// Create a temporary file for the output to avoid permission issues
						tmpOutputFile, tmpErr := os.CreateTemp("", "pgs_to_srt_*.srt")
						if tmpErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create temporary file: %v", tmpErr))
							})
							return
						}
						tmpOutputPath := tmpOutputFile.Name()
						tmpOutputFile.Close() // Close it so the script can write to it

						// Build and show the command - the script expects trained data path and input file, with output redirected
						cmdStr := fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"", pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
						// Combine text updates to reduce UI operations
						updateText := fmt.Sprintf("\n\n=== Executing Command ===\n%s\n\nConversion started at: %s\n",
							cmdStr, time.Now().Format("15:04:05"))

						fyne.Do(func() {
							result.SetText(result.Text + updateText)
						})

						// Create a log file for real-time monitoring of the PGS to SRT conversion process
						logFileName := filepath.Join(outDir, fmt.Sprintf("%s.track%d_%s.conversion.log", mkvBaseName, t.Num, t.Lang))
						logFile, logErr := os.Create(logFileName)

						// Create a logger that will be used throughout this function
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

						// Run the conversion tool with Deno - using shell to enable output redirection
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

						// Set the working directory to ensure relative paths work correctly
						cmd.Dir = filepath.Dir(pgsToSrtScript)

						// Print the environment and command for debugging
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

						// Set up pipes to capture output in real-time
						stdoutPipe, _ := cmd.StdoutPipe()
						stderrPipe, _ := cmd.StderrPipe()

						// Start the command
						startErr := cmd.Start()
						if startErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n❌ Failed to start command: %v", startErr))
							})
							if logFile != nil && logger != nil {
								logger.Printf("Failed to start command: %v\n", startErr)
							}
							err = startErr
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + "\n\n=== Starting Conversion Process ===\n")
								result.SetText(result.Text + "Check the log file for real-time output\n")
							})

							// Create a multi-writer to write to both the log file and capture the output
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

							// Regular expressions to extract progress information from the output
							frameProgressRegex := regexp.MustCompile(`Processing frame (\d+)/(\d+)`)
							statusUpdateRegex := regexp.MustCompile(`Status: (.+)`)

							// Copy stdout and stderr to the writers in a buffered way to reduce UI updates
							go func() {
								bufReader := bufio.NewReaderSize(stdoutPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"

									// Check for progress information in the output
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Extract current frame and total frames
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
										if progressData.totalFrames == 0 {
											progressData.totalFrames = totalFrames
										}

										// Calculate frame rate
										if progressData.currentFrame > 0 {
											timeDiff := time.Since(progressData.lastUpdate).Seconds()
											frameDiff := currentFrame - progressData.currentFrame
											if timeDiff > 0 && frameDiff > 0 {
												// Smooth the frame rate calculation with a weighted average
												newFrameRate := float64(frameDiff) / timeDiff
												if progressData.frameRate > 0 {
													// 70% old rate, 30% new rate for smoother estimates
													progressData.frameRate = progressData.frameRate*0.7 + newFrameRate*0.3
												} else {
													progressData.frameRate = newFrameRate
												}
											}
										}

										progressData.currentFrame = currentFrame
										progressData.lastUpdate = time.Now()
										progressMutex.Unlock()

										// Update status label
										percentComplete := float64(currentFrame) / float64(totalFrames) * 100
										fyne.Do(func() {
											statusLabel.SetText(fmt.Sprintf("Processing frame %d of %d (%.1f%%)",
												currentFrame, totalFrames, percentComplete))
										})
									} else if matches := statusUpdateRegex.FindStringSubmatch(line); len(matches) == 2 {
										// Update status message
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
								bufReader := bufio.NewReaderSize(stderrPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"

									// Also check stderr for progress information
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Process frame progress from stderr (same as stdout handler)
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
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

							// Wait for the command to complete
							err = cmd.Wait()
							output = []byte(outputBuffer.String())

							// Log the completion status
							if logFile != nil && logger != nil {
								if err != nil {
									logger.Printf("\n\nCommand completed with error: %v\n", err)
								} else {
									logger.Printf("\n\nCommand completed successfully\n")
								}
								logger.Printf("Finished at: %s\n", time.Now().Format("15:04:05"))
							}

							// Copy the temporary file to the final destination regardless of command success/failure
							// This allows us to potentially recover partial conversions even if the command had issues

							// Check if the temporary file exists before attempting to copy
							if _, statErr := os.Stat(tmpOutputPath); statErr == nil {
								if logFile != nil && logger != nil {
									logger.Printf("Copying temporary file %s to final destination %s\n", tmpOutputPath, absOutputPath)
								}

								// Create the parent directory for the output file if it doesn't exist
								outputDir := filepath.Dir(absOutputPath)
								if mkdirErr := os.MkdirAll(outputDir, 0755); mkdirErr != nil {
									copyErr = fmt.Errorf("failed to create output directory: %v", mkdirErr)
									if logFile != nil && logger != nil {
										logger.Printf("Error creating output directory: %v\n", mkdirErr)
									}
								} else {
									// Read the temporary file
									tmpContent, readErr := os.ReadFile(tmpOutputPath)
									if readErr != nil {
										copyErr = fmt.Errorf("failed to read temporary file: %v", readErr)
										if logFile != nil && logger != nil {
											logger.Printf("Error reading temporary file: %v\n", readErr)
										}
									} else {
										// Write to the final destination
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

											// Clean up the temporary file
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

							// If the command succeeded but copy failed, update the error
							if err == nil && copyErr != nil {
								err = copyErr
							}
						}

						// Prepare output text in memory before updating UI
						var outputText strings.Builder
						outputText.WriteString("\nFull command output:\n")

						// Limit output size to prevent UI sluggishness with very large outputs
						outputStr := string(output)
						const maxOutputLen = 10000 // Limit output to 10K chars
						if len(outputStr) > maxOutputLen {
							outputText.WriteString(outputStr[:maxOutputLen])
							outputText.WriteString("\n... [Output truncated, full output in log file] ...")
						} else {
							outputText.WriteString(outputStr)
						}

						// Add error message if needed
						if err != nil {
							outputText.WriteString("\n\n❌ Command error: " + err.Error())
						}

						// Update UI in a single operation
						fyne.Do(func() {
							result.SetText(result.Text + outputText.String())
						})

						// Show output
						fyne.Do(func() {
							// Calculate total conversion time
							conversionTime := time.Since(conversionStartTime).Round(time.Second)

							// Update status based on success or failure
							if err != nil {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion failed after %s", conversionTime))
							} else {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion completed in %s", conversionTime))
							}
							progress.Show()

							// Stop the ticker by removing the spinner container
							// Find and remove the conversion spinner container
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

							// Always show the full output for better debugging
							outputStr := string(output)
							result.SetText(result.Text + "\nFull output: \n" + outputStr + "\n")

							if err != nil {
								result.SetText(result.Text + "\n❌ Error: " + err.Error() + "\n")
							} else {
								result.SetText(result.Text + "\n✅ Command completed successfully\n")

								// Show file copy operation status
								result.SetText(result.Text + "\n=== File Operations ===\n")
								result.SetText(result.Text + fmt.Sprintf("✓ Temporary file created: %s\n", tmpOutputPath))
								if copySuccess {
									result.SetText(result.Text + fmt.Sprintf("✓ Copied to final destination: %s\n", absOutputPath))
									result.SetText(result.Text + "✓ Temporary file cleaned up\n")
								} else if copyErr != nil {
									result.SetText(result.Text + fmt.Sprintf("❌ Failed to copy to final destination: %v\n", copyErr))
								}
							}

							// Ensure the text area scrolls to the bottom to show the latest output
							// No need to set cursor position for Label widget
						})

						// Check current directory for debugging
						currentDir, _ := os.Getwd()
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Path Debugging ===\n")
							result.SetText(result.Text + fmt.Sprintf("Current working directory: %s\n", currentDir))
							result.SetText(result.Text + fmt.Sprintf("Looking for output file at: %s\n", absOutputPath))
						})

						// List files in output directory to see what was created
						files, _ := os.ReadDir(outDir)
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nFiles in output directory (%s):\n", outDir))
							for _, file := range files {
								result.SetText(result.Text + fmt.Sprintf("- %s\n", file.Name()))
							}
						})

						// Check if SRT file was created and show details
						if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
							fyne.Do(func() {
								result.SetText(result.Text + "\n✅ SRT file created successfully!")
								result.SetText(result.Text + fmt.Sprintf("\n   - Path: %s", absOutputPath))
								result.SetText(result.Text + fmt.Sprintf("\n   - Size: %d bytes", fileInfo.Size()))
								result.SetText(result.Text + fmt.Sprintf("\n   - Modified: %s", fileInfo.ModTime().Format("15:04:05")))

								// Try to count lines in SRT file
								if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
									lines := strings.Split(string(srtContent), "\n")
									result.SetText(result.Text + fmt.Sprintf("\n   - Lines: %d", len(lines)))

									// Count subtitle entries (every 4 lines is typically one subtitle)
									subtitleCount := (len(lines) + 3) / 4 // rough estimate
									result.SetText(result.Text + fmt.Sprintf("\n   - Estimated subtitles: ~%d", subtitleCount))
								}
							})
						} else {
							err = fmt.Errorf("SRT file was not created: %v", statErr)
							fyne.Do(func() {
								result.SetText(result.Text + "\n❌ Error: " + err.Error())
							})
						}
					}
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") || strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station")) {
					// ASS/SSA to SRT conversion
					fyne.Do(func() {
						result.SetText(setLogMessage(LogConvert, "ASS/SSA to SRT Conversion", "Starting ASS/SSA to SRT conversion process..."))
					})
					tempAssFile := fmt.Sprintf("%s.track%d_%s.ass", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absAssPath := filepath.Join(outDir, tempAssFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting ASS/SSA track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== ASS/SSA Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("ASS/SSA file: %s\n", tempAssFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absAssPath))
					})

					// Extract ASS/SSA first
					var cmd *exec.Cmd
					if IsMP4File(mkvPath) {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracting ASS/SSA track using ffmpeg...")
						})
						output, err = ExtractMP4Subtitle(mkvPath, t.Num, absAssPath)
					} else {
						cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempAssFile)
						fyne.Do(func() {
							result.SetText(result.Text + "\nRunning: " + cmdStr)
						})

						cmd = NewMkvextractCmd("tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
						AppLog("DEBUG", "ASS track extraction using mkvextract")
						cmd.Dir = outDir

						AppLog("EXTRACT", "ASS/SSA extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(mkvPath))
						output, err = cmd.CombinedOutput()
						AppLogCmd(cmd, output, err)
					}

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					assFilePath := filepath.Join(outDir, tempAssFile)
					fileInfo, statErr := os.Stat(assFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty (0 bytes)")
					} else {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted ASS/SSA file (%d bytes)", fileInfo.Size()))
						})
					}

					if err == nil {
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
						absInputPath := filepath.Join(outDir, tempAssFile)
						absOutputPath := filepath.Join(outDir, outFile)

						// Use ffmpeg to convert ASS/SSA to SRT
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] Using ffmpeg to convert ASS/SSA to SRT")
							statusLabel.SetText("Running ffmpeg conversion...")
						})

						// Get ffmpeg path - prioritize Homebrew version
						ffmpegPath := "ffmpeg" // Default fallback path

						// First check Homebrew path (preferred)
						homebrewPath := "/opt/homebrew/bin/ffmpeg"
						if _, err := os.Stat(homebrewPath); err == nil {
							ffmpegPath = homebrewPath
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] Using Homebrew ffmpeg: " + homebrewPath)
							})
						} else {
							// If Homebrew not found, check Miniconda as fallback
							homeDir, err := os.UserHomeDir()
							if err == nil {
								minicondaPath := filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg")
								if _, err := os.Stat(minicondaPath); err == nil {
									ffmpegPath = minicondaPath
									fyne.Do(func() {
										result.SetText(result.Text + "\n[DEBUG] Using Miniconda ffmpeg: " + minicondaPath)
									})
								}
							}
						}

						// Create the ffmpeg command with the appropriate path
						cmd = exec.Command(ffmpegPath, "-i", absInputPath, "-f", "srt", absOutputPath)
						cmd.Dir = outDir

						AppLog("CONVERT", "ASS/SSA to SRT conversion: %s -> %s", filepath.Base(absInputPath), filepath.Base(absOutputPath))
						// Run the command and capture output
						output, err = cmd.CombinedOutput()
						AppLogCmd(cmd, output, err)

						// Stop the ticker
						ticker.Stop()

						// Update UI with results
						fyne.Do(func() {
							result.SetText(result.Text + "\nffmpeg output: " + string(output))

							if err != nil {
								AppLog("ERROR", "ASS/SSA to SRT conversion failed: %v", err)
								result.SetText(result.Text + "\nError converting ASS/SSA to SRT: " + err.Error())
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
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "vobsub" || t.Codec == "VobSub") {
					// VobSub to SRT conversion
					fyne.Do(func() {
						result.SetText(setLogMessage(LogConvert, "VobSub to SRT Conversion", "Starting VobSub to SRT conversion process..."))
					})

					// For VobSub, we extract both .idx and .sub files
					// The .idx file is the main file that contains timing and positioning information
					// The .sub file contains the actual subtitle images
					idxFile := fmt.Sprintf("%s.track%d_%s.idx", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absIdxPath := filepath.Join(outDir, idxFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting VobSub track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== VobSub Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("IDX file: %s\n", idxFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absIdxPath))
					})

					// Extract VobSub first
					var cmd *exec.Cmd
					var cmdStr string
					if IsMP4File(mkvPath) {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracting VobSub track using ffmpeg...")
						})
						output, err = ExtractMP4Subtitle(mkvPath, t.Num, absIdxPath)
					} else {
						cmdStr = fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, idxFile)
						fyne.Do(func() {
							result.SetText(result.Text + "\nRunning: " + cmdStr)
						})

						cmd = NewMkvextractCmd("tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
						AppLog("DEBUG", "VobSub track extraction using mkvextract")
						cmd.Dir = outDir

						AppLog("EXTRACT", "VobSub extraction: Track %d (%s) from %s", t.Num, t.Lang, filepath.Base(mkvPath))
						output, err = cmd.CombinedOutput()
						AppLogCmd(cmd, output, err)
					}

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					idxFilePath := filepath.Join(outDir, idxFile)
					fileInfo, statErr := os.Stat(idxFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty")
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
						idxFile := basePath + ".idx"
						subFile := basePath + ".sub"

						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for IDX file: %s", idxFile))
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for SUB file: %s", subFile))
						})

						// Check if the files exist
						var filesExist bool = true
						if _, err := os.Stat(idxFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file exists: %s", idxFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file does not exist: %s - %v", idxFile, err))
							})
						}

						if _, err := os.Stat(subFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file exists: %s", subFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file does not exist: %s - %v", subFile, err))
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
								"eng": "en", // English
								"fre": "fr", // French
								"fra": "fr", // French (alternate)
								"ger": "de", // German
								"deu": "de", // German (alternate)
								"ita": "it", // Italian
								"spa": "es", // Spanish
								"por": "pt", // Portuguese
								"dut": "nl", // Dutch
								"nld": "nl", // Dutch (alternate)
								"swe": "sv", // Swedish
								"nor": "no", // Norwegian
								"dan": "da", // Danish
								"fin": "fi", // Finnish
								"jpn": "ja", // Japanese
								"kor": "ko", // Korean
								"chi": "zh", // Chinese
								"zho": "zh", // Chinese (alternate)
								"rus": "ru", // Russian
								"pol": "pl", // Polish
								"cze": "cs", // Czech
								"ces": "cs", // Czech (alternate)
								"hun": "hu", // Hungarian
								"gre": "el", // Greek
								"ell": "el", // Greek (alternate)
								"tur": "tr", // Turkish
								"ara": "ar", // Arabic
								"heb": "he", // Hebrew
								"tha": "th", // Thai
							}

							// Convert 3-letter code to 2-letter code if a mapping exists
							if twoLetterCode, exists := langCodeMap[strings.ToLower(langCode)]; exists {
								fyne.Do(func() {
									// Use bold formatting for important debug information to improve readability
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code: ** %s -> %s **", langCode, twoLetterCode))
								})
								langCode = twoLetterCode
							} else {
								fyne.Do(func() {
									// Use bold formatting for important debug information to improve readability
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] No mapping found for language code: ** %s **, using as-is", langCode))
								})
							}
						}

						// Use vobsub2srt binary for conversion
						conversionScript := "/usr/local/bin/vobsub2srt"

						// Check if the binary exists
						if _, err := os.Stat(conversionScript); err != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[ERROR] vobsub2srt binary not found at %s", conversionScript))
							})
							err = fmt.Errorf("vobsub2srt binary not found at %s", conversionScript)
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using vobsub2srt binary: %s", conversionScript))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code: %s for VobSub conversion", langCode))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Base path for vobsub2srt: %s", basePath))
							})

							// Check if the output SRT file already exists and delete it if it does
							outputSrtFile := basePath + ".srt"
							if _, err := os.Stat(outputSrtFile); err == nil {
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
							cmd.Dir = outDir

							// Run the command and capture output
							output, err = cmd.CombinedOutput()

							// Stop the ticker
							ticker.Stop()

							// Update UI with results
							fyne.Do(func() {
								result.SetText(result.Text + "\nvobsub2srt output: " + string(output))

								if err != nil {
									result.SetText(result.Text + "\nError converting VobSub to SRT: " + err.Error())
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
				} else {
					// Normal extraction without conversion
					// Use proper file extension based on codec
					var fileExt string
					if IsMP4File(mkvPath) {
						fileExt = MP4CodecToExtension(t.Codec)
					} else {
						fileExt = CodecToExtensionForExtract(t.Codec)
					}

					// Debug output for file naming
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n=== Track Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s - %s)\n", t.Num, t.Lang, t.Codec))
					})

					outFile = fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, t.Num, t.Lang, fileExt)

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("Output file: %s\n", outFile))
					})
					// Use absolute paths for all subtitle extractions to avoid directory creation issues
					absOutFile := filepath.Join(outDir, outFile)

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\nExtracting to: %s", absOutFile))
					})

					if IsMP4File(mkvPath) {
						// Use ffmpeg for MP4/M4V files
						AppLog("EXTRACT", "Generic extraction (ffmpeg): Track %d (%s/%s) from %s", t.Num, t.Lang, t.Codec, filepath.Base(mkvPath))
						output, err = ExtractMP4Subtitle(mkvPath, t.Num, absOutFile)
					} else {
						// Use mkvextract for MKV files
						cmd := NewMkvextractCmd("tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))
						AppLog("DEBUG", "Generic track extraction using mkvextract")

						AppLog("EXTRACT", "Generic extraction: Track %d (%s/%s) from %s", t.Num, t.Lang, t.Codec, filepath.Base(mkvPath))
						output, err = cmd.CombinedOutput()
						AppLogCmd(cmd, output, err)
					}

					// Set proper file permissions for subtitle files (read/write for user, read for group/others)
					if err == nil {
						outFilePath := filepath.Join(outDir, outFile)
						os.Chmod(outFilePath, 0644) // rw-r--r--
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
					for _, tt := range trackItems {
						trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", tt.Num, tt.Lang, tt.Codec, tt.Name))

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
				})

				tracksDone++
			}

			// Final UI update on main thread
			fyne.Do(func() {
				currentTrackLabel.SetText("")
				if tracksDone == len(selected) {
					result.SetText(setLogMessage(LogSuccess, "Extraction Complete", "All selected tracks have been processed."))
					progress.SetValue(progress.Max)
				} else {
					result.SetText(fmt.Sprintf("Extraction stopped after %d of %d tracks", tracksDone, len(selected)))
				}
			})
		}()
	})

	// Create Support button with improved UX
	supportBtn := widget.NewButton("Donate ", func() {
		// Show a confirmation dialog with information about the donation
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

	// Create button row for better layout
	buttonRow := container.NewHBox(loadTracksBtn, startExtractBtn, layout.NewSpacer(), supportBtn)

	// Setup keyboard shortcuts for main actions
	setupKeyboardShortcuts(fileBtn.OnTapped, dirBtn.OnTapped, loadTracksBtn.OnTapped, startExtractBtn.OnTapped)

	// Use app.NewWithID for better performance and to avoid preferences API warnings
	// This was already set at the beginning of main()

	// Use a more efficient layout with container.NewBorder for better performance
	// Create app title with version
	titleLabel := widget.NewLabel(fmt.Sprintf("Subtitle Forge %s", AppVersion))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create file selection button row
	fileButtonRow := container.NewHBox(
		fileBtn,
		batchBtn,
		clearBtn,
	)

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

	// Track control buttons (select/deselect all)
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

	// Track filter
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter tracks by language, codec, name, or filename...")

	// Function to filter and sort tracks based on search text
	filterTracks := func(filterText string) {
		// Clear the track list UI
		trackList.Objects = nil

		// Create a copy of trackItems for sorting
		tracksToShow := make([]*TrackItem, len(trackItems))
		copy(tracksToShow, trackItems)

		// Sorting will be implemented after sortSelect is created

		// If no filter, show all tracks
		if filterText == "" {
			for _, t := range tracksToShow {
				// Create row for this track
				var trackInfoText string
				if t.FilePath != "" {
					// Include filename for batch processing
					filename := filepath.Base(t.FilePath)
					trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
				} else {
					// Single file mode
					trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
				}
				trackInfo := widget.NewLabel(trackInfoText)

				var row *fyne.Container
				if t.ConvertOCR != nil {
					// For PGS/VobSub subtitles, show OCR option and language selection
					ocrLabel := widget.NewLabel("Convert to SRT")

					if t.LangSelect != nil {
						// Add language selection dropdown for OCR-based conversion
						langLabel := widget.NewLabel("OCR Language:")
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
					} else {
						// For ASS/SSA conversion (no OCR language needed)
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
					}
				} else {
					// For other subtitle formats
					row = container.NewHBox(t.Check, t.Status, trackInfo)
				}

				trackList.Add(row)
			}
		} else {
			// Convert filter text to lowercase for case-insensitive comparison
			lowerFilter := strings.ToLower(filterText)

			// Add only tracks that match the filter
			for _, t := range trackItems {
				// Check if the track matches the filter criteria
				matchesFilter := strings.Contains(strings.ToLower(t.Lang), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Codec), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Name), lowerFilter) ||
					strings.Contains(strings.ToLower(fmt.Sprintf("Track %d", t.Num)), lowerFilter) ||
					strings.Contains(strings.ToLower(filepath.Base(t.FilePath)), lowerFilter)

				if matchesFilter {
					// Create row for this track
					var trackInfoText string
					if t.FilePath != "" {
						// Include filename for batch processing
						filename := filepath.Base(t.FilePath)
						trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
					} else {
						// Single file mode
						trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
					}
					trackInfo := widget.NewLabel(trackInfoText)

					var row *fyne.Container
					if t.ConvertOCR != nil {
						// For PGS/VobSub subtitles, show OCR option and language selection
						ocrLabel := widget.NewLabel("Convert to SRT")

						if t.LangSelect != nil {
							// Add language selection dropdown for OCR-based conversion
							langLabel := widget.NewLabel("OCR Language:")
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
						} else {
							// For ASS/SSA conversion (no OCR language needed)
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
						}
					} else {
						// For other subtitle formats
						row = container.NewHBox(t.Check, t.Status, trackInfo)
					}

					trackList.Add(row)
				}
			}
		}

		trackList.Refresh()
	}

	// Set up filter entry change handler
	filterEntry.OnChanged = func(text string) {
		filterTracks(text)
	}

	// Track sorting dropdown
	sortSelect := widget.NewSelect([]string{
		"Default Order",
		"By Filename",
		"By Language",
		"By Codec",
		"By Track Number",
	}, func(value string) {
		// Re-apply current filter with new sort order
		filterTracks(filterEntry.Text)
	})
	sortSelect.SetSelected("Default Order")

	// Now update filterTracks to include sorting
	filterTracks = func(filterText string) {
		// Clear the track list UI
		trackList.Objects = nil

		// Get tracks to display (filtered or all)
		var tracksToShow []*TrackItem
		if filterText == "" {
			// Show all tracks
			tracksToShow = make([]*TrackItem, len(trackItems))
			copy(tracksToShow, trackItems)
		} else {
			// Convert filter text to lowercase for case-insensitive comparison
			lowerFilter := strings.ToLower(filterText)

			// Add only tracks that match the filter
			for _, t := range trackItems {
				// Check if the track matches the filter criteria
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

		// Apply sorting based on current selection
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

		// Display the sorted/filtered tracks
		for _, t := range tracksToShow {
			// Create row for this track
			var trackInfoText string
			if t.FilePath != "" {
				// Include filename for batch processing
				filename := filepath.Base(t.FilePath)
				trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s [%s]", t.Num, t.Lang, t.Codec, t.Name, filename)
			} else {
				// Single file mode
				trackInfoText = fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name)
			}
			trackInfo := widget.NewLabel(trackInfoText)

			var row *fyne.Container
			if t.ConvertOCR != nil {
				// For PGS/VobSub subtitles, show OCR option and language selection
				ocrLabel := widget.NewLabel("Convert to SRT")

				if t.LangSelect != nil {
					// Add language selection dropdown for OCR-based conversion
					langLabel := widget.NewLabel("OCR Language:")
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
				} else {
					// For ASS/SSA conversion (no OCR language needed)
					row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
				}
			} else {
				// For other subtitle formats
				row = container.NewHBox(t.Check, t.Status, trackInfo)
			}

			trackList.Add(row)
		}

		trackList.Refresh()
	}

	// Track control container with buttons and filter
	// Make the filter entry take more space by setting its placeholder to be longer
	filterEntry.SetPlaceHolder("Filter tracks by language, codec, name, filename, or track number...                                                 ")

	// Using a grid layout to give the filter entry more space
	filterBox := container.New(
		layout.NewFormLayout(),
		widget.NewLabel("Filter:"),
		filterEntry,
	)

	// Sort box
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

	// Create tab for subtitle extraction (existing functionality)
	extractTabContent := container.NewBorder(
		topContent,
		bottomContent,
		nil,
		nil,
		middleContent,
	)

	// Create tab for subtitle insertion
	insertTabContent, insertWidgets := createInsertSubtitlesTab(w)
	insertMkvFileLabel := insertWidgets.MkvFileLabel
	insertSubtitleFileLabel := insertWidgets.SubtitleFileLabel
	mkvDropLabel := insertWidgets.MkvDropLabel
	subtitleDropLabel := insertWidgets.SubtitleDropLabel
	mkvDropArea := insertWidgets.MkvDropArea
	subtitleDropArea := insertWidgets.SubtitleDropArea

	// Create settings tab content
	settingsTabContent := createSettingsTab(a, w, dependencyButtons)
	updateDependencyStatus(w)

	// Create convert subtitles tab content
	convertTabContent, loadConvertFile, loadConvertFiles := createConvertSubtitlesTab(w)

	// Wrap each tab content in a scroll container to ensure proper resizability
	extractScroll := container.NewScroll(extractTabContent)
	insertScroll := container.NewScroll(insertTabContent)
	convertScroll := container.NewScroll(convertTabContent)
	settingsScroll := container.NewScroll(settingsTabContent)

	// Create AI Translation tab
	aiTranslationTabContent := createAITranslationTab(w, a)
	aiTranslationScroll := container.NewScroll(aiTranslationTabContent)

	// Create Whisper transcription tab
	whisperTabContent := createWhisperTranscribeTab(w, a)
	whisperScroll := container.NewScroll(whisperTabContent)

	// Create LibreTranslate tab
	libreTranslateTabContent := createLibreTranslateTab(w, a)
	libreTranslateScroll := container.NewScroll(libreTranslateTabContent)

	// Create Logs tab
	logsTabContent := createLogsTab(w)
	logsScroll := container.NewScroll(logsTabContent)

	// Create tabs with scrollable content
	tabs := container.NewAppTabs(
		container.NewTabItem("Extract Subtitles", extractScroll),
		container.NewTabItem("Insert Subtitles", insertScroll),
		container.NewTabItem("Convert Subtitles", convertScroll),
		container.NewTabItem("🎙️ Transcribe", whisperScroll),
		container.NewTabItem("🌍 LibreTranslate", libreTranslateScroll),
		container.NewTabItem("🤖 AI Translate", aiTranslationScroll),
		container.NewTabItem("📋 Logs", logsScroll),
		container.NewTabItem("Settings", settingsScroll),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Set up tab change handler for drag and drop
	tabs.OnChanged = func(tab *container.TabItem) {
		// Default: clear any previous tab's drop handler
		w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {})
		if tab.Text == "Insert Subtitles" {
			// Set up drag and drop for Insert Subtitles tab
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) > 0 {
					filePath := uris[0].Path()

					if IsVideoFile(filePath) {
						// Handle video file drop (MKV, MP4, M4V)
						insertMkvFileLabel.SetText(filePath)
						mkvDropLabel.SetText(filepath.Base(filePath))
						mkvDropArea.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
						mkvDropArea.Refresh()
						a.SendNotification(&fyne.Notification{
							Title:   "File Dropped",
							Content: "Video file loaded: " + filepath.Base(filePath),
						})
					} else if IsSubtitleFile(filePath) {
						// Handle subtitle file drop
						insertSubtitleFileLabel.SetText(filePath)
						subtitleDropLabel.SetText(filepath.Base(filePath))
						subtitleDropArea.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
						subtitleDropArea.Refresh()
						a.SendNotification(&fyne.Notification{
							Title:   "File Dropped",
							Content: "Subtitle file loaded: " + filepath.Base(filePath),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Invalid File",
							Content: "Please drop an MKV/MP4 or subtitle file.",
						})
					}
				}
			})
		} else if tab.Text == "🎙️ Transcribe" {
			// Drag & drop for Whisper transcription tab
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}
				filePath := uris[0].Path()
				if filePath == "" {
					return
				}
				if whisperTranscribeSetInputFile != nil {
					whisperTranscribeSetInputFile(filePath)
					a.SendNotification(&fyne.Notification{
						Title:   "Media File Loaded",
						Content: "Loaded for transcription: " + filepath.Base(filePath),
					})
				}
			})
		} else if tab.Text == "🌍 LibreTranslate" {
			// Drag & drop for LibreTranslate tab (SRT only)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}

				var srtFiles []string
				for _, uri := range uris {
					filePath := uri.Path()
					if filePath == "" {
						continue
					}
					if strings.ToLower(filepath.Ext(filePath)) == ".srt" {
						srtFiles = append(srtFiles, filePath)
					}
				}

				if len(srtFiles) == 0 {
					a.SendNotification(&fyne.Notification{
						Title:   "Invalid Files",
						Content: "Please drop .srt subtitle file(s).",
					})
					return
				}

				if len(srtFiles) == 1 {
					if libreTranslateSetInputFile != nil {
						libreTranslateSetInputFile(srtFiles[0])
					}
					a.SendNotification(&fyne.Notification{
						Title:   "Subtitle File Loaded",
						Content: "Loaded for LibreTranslate: " + filepath.Base(srtFiles[0]),
					})
					return
				}

				if libreTranslateAddInputFile != nil {
					for _, filePath := range srtFiles {
						libreTranslateAddInputFile(filePath)
					}
				} else if libreTranslateSetInputFile != nil {
					libreTranslateSetInputFile(srtFiles[0])
				}

				a.SendNotification(&fyne.Notification{
					Title:   "Batch Mode Enabled",
					Content: fmt.Sprintf("Added %d SRT files for LibreTranslate", len(srtFiles)),
				})
			})
		} else if tab.Text == "🤖 AI Translate" {
			// Set up drag and drop for AI Translation tab (supports multiple files)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}

				// Accept subtitle files for AI translation
				var subtitleFiles []string

				// Process all dropped files
				for _, uri := range uris {
					filePath := uri.Path()

					if IsSubtitleFile(filePath) {
						subtitleFiles = append(subtitleFiles, filePath)
					}
				}

				// Add all valid subtitle files
				if len(subtitleFiles) > 0 {
					if aiTranslationAddFile != nil {
						for _, filePath := range subtitleFiles {
							aiTranslationAddFile(filePath)
						}
					}

					if len(subtitleFiles) == 1 {
						a.SendNotification(&fyne.Notification{
							Title:   "Subtitle File Added",
							Content: "Added to translation queue: " + filepath.Base(subtitleFiles[0]),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Batch Mode Enabled",
							Content: fmt.Sprintf("Added %d subtitle files to translation queue", len(subtitleFiles)),
						})
					}
				} else {
					a.SendNotification(&fyne.Notification{
						Title:   "Invalid Files",
						Content: "Please drop subtitle file(s) (.srt, .ass, .ssa, .vtt, .sub) for AI translation.",
					})
				}
			})
		} else if tab.Text == "Convert Subtitles" {
			// Enhanced drag and drop for Convert Subtitles tab (supports batch processing)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}

				// Check if it's a single file or multiple files
				if len(uris) == 1 {
					// Single file - use existing single file logic
					filePath := uris[0].Path()
					fileExt := strings.ToLower(filepath.Ext(filePath))

					// Check if it's a supported subtitle format
					if IsSubtitleFileWithIdx(filePath) {
						// Load the file into the Convert tab
						loadConvertFile(filePath)
						a.SendNotification(&fyne.Notification{
							Title:   "Subtitle File Loaded",
							Content: "Loaded " + strings.ToUpper(fileExt[1:]) + " file: " + filepath.Base(filePath),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Invalid File",
							Content: "Please drop a subtitle file (.srt, .ass, .vtt, .sub, etc.)",
						})
					}
				} else {
					// Multiple files - use batch mode
					var subtitleFiles []string

					for _, uri := range uris {
						filePath := uri.Path()
						if IsSubtitleFileWithIdx(filePath) {
							subtitleFiles = append(subtitleFiles, filePath)
						}
					}

					if len(subtitleFiles) > 0 {
						// Load multiple files for batch conversion
						loadConvertFiles(subtitleFiles)
						a.SendNotification(&fyne.Notification{
							Title:   "Batch Mode Enabled",
							Content: fmt.Sprintf("Loaded %d subtitle files for batch conversion", len(subtitleFiles)),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "No Valid Files",
							Content: "Please drop subtitle files (.srt, .ass, .vtt, .sub, etc.)",
						})
					}
				}
			})
		} else if tab.Text == "Extract Subtitles" {
			// Enhanced drag and drop for Extract Subtitles tab (supports batch processing)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}

				// Filter for supported video files
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
					// Single file mode
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

					// Update UI
					selectedFile.SetText(mkvPath)

					// Set output directory to the same directory as the video file
					outDir = filepath.Dir(mkvPath)
					selectedDir.SetText(outDir)

					// Clear previous tracks
					trackItems = []*TrackItem{}
					trackList.Objects = nil
					trackList.Refresh()

					result.SetText(setLogMessage(LogInfo, "Video File Loaded", "Video file dropped and loaded. Output directory automatically set to file location. Click 'Load Tracks' to analyze the file."))
				} else {
					// Multiple files - batch mode
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

					// Set output directory to the directory of the first file
					if len(mkvFiles) > 0 {
						outDir = filepath.Dir(mkvFiles[0])
						selectedDir.SetText(outDir)
					}

					// Clear previous tracks
					trackItems = []*TrackItem{}
					trackList.Objects = nil
					trackList.Refresh()

					selectedFile.SetText(fmt.Sprintf("%d video files selected for batch processing", len(mkvFiles)))
					result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Dropped %d video files. Click 'Load Tracks' to analyze all files and select which tracks to extract.", len(mkvFiles))))
				}
			})
		}
	}
	w.SetContent(tabs)

	// Trigger the OnChanged handler for the initial tab
	tabs.OnChanged(tabs.Selected())

	w.ShowAndRun()
}
