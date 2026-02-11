package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
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
	logsTitle := widget.NewLabel(T("logs.title"))
	logsTitle.TextStyle = fyne.TextStyle{Bold: true}
	logsTitle.Alignment = fyne.TextAlignCenter

	// Log file path display
	logPathLabel := widget.NewLabel(T("logs.initializing"))
	logPathLabel.Wrapping = fyne.TextWrapWord
	if appLogPath != "" {
		logPathLabel.SetText(T("logs.log_file") + appLogPath)
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
	copyPathBtn := widget.NewButton(T("logs.copy_path"), func() {
		if appLogPath == "" {
			dialog.ShowInformation(T("common.not_available"), T("logs.not_available"), w)
			return
		}
		w.Clipboard().SetContent(appLogPath)
		dialog.ShowInformation(T("common.copied"), T("logs.path_copied"), w)
	})
	copyPathBtn.Importance = widget.MediumImportance

	openFolderBtn := widget.NewButton(T("logs.open_folder"), func() {
		if appLogPath == "" {
			dialog.ShowInformation(T("common.not_available"), T("logs.not_available"), w)
			return
		}
		logDir := filepath.Dir(appLogPath)
		// Use open command on macOS
		cmd := exec.Command("open", logDir)
		if err := cmd.Run(); err != nil {
			dialog.ShowError(fmt.Errorf(T("logs.open_error"), err), w)
		}
	})
	openFolderBtn.Importance = widget.MediumImportance

	refreshBtn := widget.NewButton(T("common.refresh"), func() {
		logViewer.SetText(GetLogBuffer())
		logViewerScroll.ScrollToBottom()
	})
	refreshBtn.Importance = widget.LowImportance

	clearBufferBtn := widget.NewButton(T("logs.clear_display"), func() {
		ClearLogBuffer()
		logViewer.SetText(GetLogBuffer())
	})
	clearBufferBtn.Importance = widget.LowImportance

	copyLogsBtn := widget.NewButton(T("logs.copy_all"), func() {
		logs := GetLogBuffer()
		if logs == "" {
			dialog.ShowInformation(T("common.empty"), T("logs.no_logs"), w)
			return
		}
		w.Clipboard().SetContent(logs)
		dialog.ShowInformation(T("common.copied"), T("logs.all_copied"), w)
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
	infoLabel := widget.NewLabel(T("logs.auto_save_info"))
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

	// Create app with explicit ID and set metadata directly
	a := app.NewWithID("com.gmm.subtitleforge")
	a.SetIcon(theme.FileTextIcon())

	// Apply theme based on saved preference
	savedTheme := a.Preferences().StringWithFallback("theme", "Dark Theme")
	ApplyThemeByName(a, savedTheme)

	// Initialize UI language from saved preference or system locale
	savedLang := a.Preferences().StringWithFallback("ui_language", "")
	if savedLang != "" {
		SetLanguage(savedLang)
	} else {
		SetLanguage(DetectSystemLanguage())
	}

	// Create main window with explicit name
	w := a.NewWindow("Subtitle Forge")
	// Set app metadata on window
	w.SetMaster()

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

	firstLaunch := width == 0 || height == 0
	if firstLaunch {
		width = defaultWidth
		height = defaultHeight
	}

	// Resize window to saved or default size
	AppLog("DEBUG", "Restoring window size: %.0fx%.0f (first launch: %v)", width, height, firstLaunch)
	w.Resize(fyne.NewSize(width, height))
	w.CenterOnScreen() // center first, then override with saved position below
	w.SetFixedSize(false)

	// Restore saved window position (macOS native API)
	savedX := a.Preferences().Float("window_x")
	savedY := a.Preferences().Float("window_y")
	if !firstLaunch && savedX >= 0 && savedY >= 0 {
		// Delay position restore slightly to ensure the window is fully created
		go func() {
			time.Sleep(200 * time.Millisecond)
			fyne.Do(func() {
				SetWindowPosition(savedX, savedY)
				AppLog("DEBUG", "Restored window position: %.0f, %.0f", savedX, savedY)
			})
		}()
	}

	// Save window size and position periodically
	var lastSize fyne.Size
	var lastX, lastY float64
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fyne.Do(func() {
				currentSize := w.Canvas().Size()
				if currentSize.Width != lastSize.Width || currentSize.Height != lastSize.Height {
					a.Preferences().SetFloat("window_width", float64(currentSize.Width))
					a.Preferences().SetFloat("window_height", float64(currentSize.Height))
					lastSize = currentSize
				}

				// Save window position (macOS native)
				x, y := GetWindowPosition()
				if x >= 0 && y >= 0 && (x != lastX || y != lastY) {
					a.Preferences().SetFloat("window_x", x)
					a.Preferences().SetFloat("window_y", y)
					lastX = x
					lastY = y
				}
			})
		}
	}()

	// Also save window size and position when closing
	w.SetCloseIntercept(func() {
		currentSize := w.Canvas().Size()
		a.Preferences().SetFloat("window_width", float64(currentSize.Width))
		a.Preferences().SetFloat("window_height", float64(currentSize.Height))

		// Save position
		x, y := GetWindowPosition()
		if x >= 0 && y >= 0 {
			a.Preferences().SetFloat("window_x", x)
			a.Preferences().SetFloat("window_y", y)
		}

		// Close the window
		w.Close()
	})

	// Create Extract Subtitles tab
	extractTabContent, extractWidgets := createExtractSubtitlesTab(w, a)
	dependencyButtons := extractWidgets.DependencyButtons

	// Setup keyboard shortcuts for extract tab actions
	setupKeyboardShortcuts(extractWidgets.FileOpenFunc, extractWidgets.DirChangeFunc, extractWidgets.LoadTracksFunc, extractWidgets.StartExtractFunc)

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
		container.NewTabItem(T("tab.extract"), extractScroll),
		container.NewTabItem(T("tab.insert"), insertScroll),
		container.NewTabItem(T("tab.convert"), convertScroll),
		container.NewTabItem("🎙️ "+T("tab.whisper"), whisperScroll),
		container.NewTabItem("🌍 "+T("tab.libre"), libreTranslateScroll),
		container.NewTabItem("🤖 "+T("tab.ai_translate"), aiTranslationScroll),
		container.NewTabItem("📋 "+T("tab.logs"), logsScroll),
		container.NewTabItem(T("tab.settings"), settingsScroll),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Set up tab change handler for drag and drop
	tabs.OnChanged = func(tab *container.TabItem) {
		// Default: clear any previous tab's drop handler
		w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {})
		if tab.Text == T("tab.insert") {
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
		} else if tab.Text == "🎙️ "+T("tab.whisper") {
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
		} else if tab.Text == "🌍 "+T("tab.libre") {
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
		} else if tab.Text == "🤖 "+T("tab.ai_translate") {
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
		} else if tab.Text == T("tab.convert") {
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
		} else if tab.Text == T("tab.extract") {
			// Delegate to extract tab's drag-and-drop handler
			w.SetOnDropped(extractWidgets.OnDropped)
		}
	}
	w.SetContent(tabs)

	// Trigger the OnChanged handler for the initial tab
	tabs.OnChanged(tabs.Selected())

	w.ShowAndRun()
}
