package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// AITranslationConfig holds all GST translation settings
type AITranslationConfig struct {
	GSTPath         string // Path to gst binary
	APIKey          string // API key for GST
	SecondaryAPIKey string // For quota management
	Model           string // Model name for GST
	Temperature     float64
	BatchSize       int
	Description     string
	ResumeMode      bool
	ProgressLog     bool
	ThoughtsLog     bool
	ThinkingBudget  int    // For Gemini 2.5 models (0-24576 for Flash, 128-32768 for Pro). 0 means unset.
	ThinkingLevel   string // For Gemini 3.0 models (minimal, low, medium, high)
}

// findGSTPath tries to find the gst executable. Prefers the user's venv path, then PATH.
func findGSTPath() string {
	if gstPath, err := exec.LookPath("gst"); err == nil {
		return gstPath
	}

	candidates := []string{
		"/opt/homebrew/bin/gst",
		"/usr/local/bin/gst",
		"/usr/bin/gst",
		"/bin/gst",
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(homeDir, "bin", "gst"),
			filepath.Join(homeDir, ".local", "bin", "gst"),
			filepath.Join(homeDir, ".subtitle-forge", "gst-venv", "bin", "gst"),
			filepath.Join(homeDir, "subsvenv", "bin", "gst"),
			filepath.Join(homeDir, "venv", "bin", "gst"),
			filepath.Join(homeDir, ".venv", "bin", "gst"),
		)

		pythonLibPath := filepath.Join(homeDir, "Library", "Python")
		if pythonDirs, err := os.ReadDir(pythonLibPath); err == nil {
			for _, pythonDir := range pythonDirs {
				if pythonDir.IsDir() {
					candidates = append(candidates, filepath.Join(pythonLibPath, pythonDir.Name(), "bin", "gst"))
				}
			}
		}
	}

	for _, candidate := range candidates {
		if fileInfo, err := os.Stat(candidate); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
			return candidate
		}
	}

	return "gst"
}

// translateWithGSTInTerminal opens a terminal window and runs gst translate there
func translateWithGSTInTerminal(inputFile, outputFile, targetLang string, config AITranslationConfig) error {
	gstPath := config.GSTPath
	if strings.TrimSpace(gstPath) == "" {
		gstPath = findGSTPath()
	}

	// Build gst command arguments
	args := []string{"translate"}
	if inputFile != "" {
		args = append(args, "-i", inputFile)
	}
	if targetLang != "" {
		args = append(args, "-l", targetLang)
	}
	if config.APIKey != "" {
		args = append(args, "-k", config.APIKey)
	}
	if config.SecondaryAPIKey != "" {
		args = append(args, "-k2", config.SecondaryAPIKey)
	}
	if outputFile != "" {
		args = append(args, "-o", outputFile)
	}
	if config.Model != "" {
		args = append(args, "-m", config.Model)
	}
	if config.BatchSize > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", config.BatchSize))
	}
	if config.Temperature >= 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", config.Temperature))
	}
	if config.ProgressLog {
		args = append(args, "--progress-log")
	}
	if config.ThoughtsLog {
		args = append(args, "--thoughts-log")
	}
	if config.ThinkingBudget > 0 {
		args = append(args, "--thinking-budget", fmt.Sprintf("%d", config.ThinkingBudget))
	}
	if config.ThinkingLevel != "" {
		args = append(args, "--thinking-level", config.ThinkingLevel)
	}
	if config.ResumeMode {
		args = append(args, "--resume")
	} else {
		args = append(args, "--no-resume")
	}
	if desc := strings.TrimSpace(config.Description); desc != "" {
		args = append(args, "-d", desc)
	}

	// Build full command string with proper escaping
	cmdStr := gstPath
	for _, arg := range args {
		// Escape single quotes and wrap args with spaces in quotes
		if strings.Contains(arg, " ") || strings.Contains(arg, "'") {
			cmdStr += " '" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
		} else {
			cmdStr += " " + arg
		}
	}

	// Build enhanced terminal script with info display and error handling
	fileName := filepath.Base(inputFile)
	outputFileName := filepath.Base(outputFile)

	terminalScript := fmt.Sprintf(`
echo -ne "\\033]0;GST Translation: %s\\007"
clear
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║                    🌐 GST Subtitle Translation                     ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
echo "📁 Input:  %s"
echo "📝 Output: %s"
echo "🌍 Target: %s"
echo ""
if [ -f '%s' ]; then
    line_count=$(grep -c "^[0-9]\\+$" '%s' 2>/dev/null || echo "?")
    echo "📊 Subtitle entries: $line_count"
fi
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "⏳ Starting translation..."
echo ""
START_TIME=$(date +%%s)
%s
EXIT_CODE=$?
END_TIME=$(date +%%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION %% 60))
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Translation completed successfully!"
    echo "⏱️  Duration: ${MINUTES}m ${SECONDS}s"
    if [ -f '%s' ]; then
        SIZE=$(ls -lh '%s' | awk '{print $5}')
        echo "📦 Output size: $SIZE"
    fi
else
    echo "❌ Translation failed with exit code: $EXIT_CODE"
fi
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Options:"
echo "  [Enter]  Close this window"
echo "  [o]      Open output file location"
echo "  [l]      View log file (if created)"
echo ""
read -n 1 -p "Your choice: " choice
echo ""
case "$choice" in
    o|O)
        open -R '%s'
        ;;
    l|L)
        LOG_FILE='%s'
        if [ -f "$LOG_FILE" ]; then
            less "$LOG_FILE"
        else
            echo "No log file found at: $LOG_FILE"
            sleep 2
        fi
        ;;
esac
`,
		fileName,
		fileName,
		outputFileName,
		targetLang,
		inputFile,
		inputFile,
		cmdStr,
		outputFile,
		outputFile,
		outputFile,
		strings.TrimSuffix(inputFile, filepath.Ext(inputFile))+"_progress.log",
	)

	// Use osascript to open Terminal.app with the enhanced script
	// Write script to temp file to avoid escaping issues
	tmpScript, err := os.CreateTemp("", "gst-translate-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temp script: %v", err)
	}
	tmpScriptPath := tmpScript.Name()

	// Add self-deletion at the end of the script
	terminalScriptWithCleanup := terminalScript + fmt.Sprintf("\nrm -f '%s'", tmpScriptPath)

	if _, err := tmpScript.WriteString(terminalScriptWithCleanup); err != nil {
		return fmt.Errorf("failed to write temp script: %v", err)
	}
	if err := tmpScript.Close(); err != nil {
		return fmt.Errorf("failed to close temp script: %v", err)
	}

	// Make script executable
	if err := os.Chmod(tmpScriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %v", err)
	}

	// Open Terminal.app and execute the script
	// Note: Script will delete itself when done
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "bash %s"
end tell`, tmpScriptPath)

	cmd := exec.Command("osascript", "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript failed: %v (output: %s)", err, string(output))
	}
	return nil
}

// postProcessSRT fixes common spacing and line break issues in translated SRT files.
// It only applies punctuation fixes to subtitle text lines, leaving index and timestamp lines untouched.
func postProcessSRT(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	timestampRe := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}[,\.]\d{3}\s*-->`)
	indexRe := regexp.MustCompile(`^\d+\s*$`)

	commaRe := regexp.MustCompile(`,(\S)`)
	dotRe := regexp.MustCompile(`\.(\S)`)
	questionRe := regexp.MustCompile(`\?(\S)`)
	exclamationRe := regexp.MustCompile(`!(\S)`)

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || timestampRe.MatchString(trimmed) || indexRe.MatchString(trimmed) {
			continue
		}
		// Fix spacing after punctuation only in subtitle text lines
		line = commaRe.ReplaceAllString(line, ", $1")
		line = dotRe.ReplaceAllString(line, ". $1")
		line = questionRe.ReplaceAllString(line, "? $1")
		line = exclamationRe.ReplaceAllString(line, "! $1")
		lines[i] = line
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}

// translateWithGST shells out to gst translate and streams output
func translateWithGST(inputFile, outputFile, targetLang string, config AITranslationConfig, progressCallback func(string)) (bool, error) {
	gstPath := config.GSTPath
	if strings.TrimSpace(gstPath) == "" {
		gstPath = findGSTPath()
	}
	args := []string{"translate"}
	if inputFile != "" {
		args = append(args, "-i", inputFile)
	}
	if targetLang != "" {
		args = append(args, "-l", targetLang)
	}
	if config.APIKey != "" {
		args = append(args, "-k", config.APIKey)
	}
	if config.SecondaryAPIKey != "" {
		args = append(args, "-k2", config.SecondaryAPIKey)
	}
	if outputFile != "" {
		args = append(args, "-o", outputFile)
	}
	if config.Model != "" {
		args = append(args, "-m", config.Model)
	}
	if config.BatchSize > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", config.BatchSize))
	}
	// Temperature is float, pass if set
	if config.Temperature >= 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", config.Temperature))
	}
	if config.ProgressLog {
		args = append(args, "--progress-log")
	}
	if config.ThoughtsLog {
		args = append(args, "--thoughts-log")
	}
	if config.ThinkingBudget > 0 {
		args = append(args, "--thinking-budget", fmt.Sprintf("%d", config.ThinkingBudget))
	}
	if config.ThinkingLevel != "" {
		args = append(args, "--thinking-level", config.ThinkingLevel)
	}
	if config.ResumeMode {
		args = append(args, "--resume")
	} else {
		args = append(args, "--no-resume")
	}
	if desc := strings.TrimSpace(config.Description); desc != "" {
		args = append(args, "-d", desc)
	}

	AppLog("TRANSLATE", "Starting GST translation: %s -> %s (lang: %s, model: %s)", filepath.Base(inputFile), filepath.Base(outputFile), targetLang, config.Model)
	cmd := exec.Command(gstPath, args...)
	AppLog("CMD", "Execute: %s %s", gstPath, strings.Join(args, " "))
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		AppLog("ERROR", "AI Translation: Failed to start gst: %v", err)
		return false, fmt.Errorf("failed to start gst: %v", err)
	}

	// Stream gst output lines to progress callback
	stream := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 256*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			raw := scanner.Text()
			// Remove carriage returns used for in-place progress and ANSI color codes
			line := strings.ReplaceAll(raw, "\r", "")
			line = stripANSI(line)
			if strings.TrimSpace(line) == "" {
				continue
			}
			if progressCallback != nil {
				progressCallback(line)
			}
		}
	}
	go stream(stdout)
	go stream(stderr)

	err := cmd.Wait()
	if err != nil {
		AppLog("ERROR", "AI Translation: gst failed: %v", err)
		return false, fmt.Errorf("gst failed: %v", err)
	}
	// gst writes the output file itself; treat as success if it completed
	AppLog("SUCCESS", "AI Translation completed: %s", filepath.Base(outputFile))

	// Post-process the translated SRT file to fix spacing issues
	if err := postProcessSRT(outputFile); err != nil {
		AppLog("WARNING", "Failed to post-process SRT file: %v", err)
		// Don't fail the translation, just log the warning
	}

	return true, nil
}

// stripANSI removes ANSI escape sequences used for terminal colors and cursor moves
func stripANSI(s string) string {
	// Matches CSI sequences like \x1b[31m, \x1b[?25l, and OSC etc.
	ansi := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	s = ansi.ReplaceAllString(s, "")
	// Also remove other ESC sequences (OSC terminated by BEL or ST) conservatively
	osc := regexp.MustCompile(`\x1b\].*?(\x07|\x1b\\)`)
	s = osc.ReplaceAllString(s, "")
	return s
}

// TranslationJob represents a translation task
type TranslationJob struct {
	ID           string
	InputFile    string
	OutputFile   string
	SourceLang   string
	TargetLang   string
	Status       string // "pending", "running", "completed", "failed", "paused"
	Progress     float64
	StartLine    int
	TotalLines   int
	Config       AITranslationConfig
	VideoFile    string // For audio context
	CreatedAt    time.Time
	CompletedAt  time.Time
	ErrorMessage string
}

// Removed Gemini-specific types - using GST only

// Removed callGeminiAPI - using GST only

// Global variables for AI translation
var activeTranslationJobs = make(map[string]*TranslationJob)
var aiTranslationAddFile func(string) // Function to add files from drag & drop
var cancelTranslation bool            // Flag to cancel ongoing translation
var resultsWindow fyne.Window         // Separate results window
var resultsArea *widget.Entry         // Results text area
var resultsScroll *container.Scroll   // Scroll container for results
var aiResultsActions *fyne.Container  // Output actions for results window
var aiResultsLines []string           // Capped line buffer for results area

const maxAIResultLines = 500

// appendToResults appends a line to the results area using a capped slice (avoids O(n²) string growth).
func appendToResults(line string) {
	aiResultsLines = append(aiResultsLines, line)
	if len(aiResultsLines) > maxAIResultLines {
		aiResultsLines = aiResultsLines[len(aiResultsLines)-maxAIResultLines:]
	}
	if resultsArea != nil {
		resultsArea.SetText(strings.Join(aiResultsLines, "\n"))
	}
}

func updateAIResultsActions(w fyne.Window, outputDir string) {
	if aiResultsActions == nil {
		return
	}
	if strings.TrimSpace(outputDir) == "" {
		aiResultsActions.Hide()
		aiResultsActions.Objects = nil
		aiResultsActions.Refresh()
		return
	}

	aiResultsActions.Objects = []fyne.CanvasObject{
		widget.NewButton(T("common.open_output_folder"), func() {
			openFolderPath(w, outputDir)
		}),
		widget.NewButton(T("common.copy_output_path"), func() {
			copyTextToClipboard(w, outputDir)
		}),
	}
	aiResultsActions.Show()
	aiResultsActions.Refresh()
}

// createResultsWindow creates a separate window for translation results
func createResultsWindow(a fyne.App) {
	if resultsWindow != nil {
		return // Already created
	}

	resultsWindow = a.NewWindow("Translation Results")
	resultsWindow.Resize(fyne.NewSize(800, 600))

	// Results area with scrolling
	aiResultsLines = nil
	resultsArea = widget.NewMultiLineEntry()
	resultsArea.SetText(T("ai.results_placeholder"))
	resultsArea.Wrapping = fyne.TextWrapWord
	resultsScroll = container.NewScroll(resultsArea)

	// Clear button
	clearResultsBtn := widget.NewButton(T("ai.clear_results"), func() {
		resultsArea.SetText(T("ai.results_cleared"))
	})

	// Copy to clipboard button
	copyBtn := widget.NewButton(T("ai.copy_all"), func() {
		resultsWindow.Clipboard().SetContent(resultsArea.Text)
		dialog.ShowInformation(T("common.copied"), T("ai.results_copied"), resultsWindow)
	})

	aiResultsActions = container.NewHBox()
	aiResultsActions.Hide()

	// Layout
	content := container.NewBorder(
		nil,
		container.NewVBox(
			container.NewHBox(clearResultsBtn, copyBtn),
			aiResultsActions,
		),
		nil,
		nil,
		resultsScroll,
	)

	resultsWindow.SetContent(content)

	// Don't close the app when results window closes
	resultsWindow.SetCloseIntercept(func() {
		resultsWindow.Hide()
	})
}

// createAITranslationTab creates the AI translation interface
func createAITranslationTab(w fyne.Window, a fyne.App) *fyne.Container {
	// Title
	title := widget.NewLabel(T("ai.title"))
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// API Key configuration
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder(T("ai.api_key_hint"))

	secondaryKeyEntry := widget.NewPasswordEntry()
	secondaryKeyEntry.SetPlaceHolder(T("ai.secondary_key_hint"))

	// Remember API Key checkbox
	rememberAPIKeyCheck := widget.NewCheck(T("ai.remember_keys"), nil)
	rememberAPIKeyCheck.Checked = a.Preferences().BoolWithFallback("ai_remember_api_key", false)

	// Load saved API keys if "Remember" is enabled
	if rememberAPIKeyCheck.Checked {
		savedAPIKey := a.Preferences().StringWithFallback("ai_api_key", "")
		savedSecondaryKey := a.Preferences().StringWithFallback("ai_secondary_api_key", "")
		if savedAPIKey != "" {
			apiKeyEntry.SetText(savedAPIKey)
		}
		if savedSecondaryKey != "" {
			secondaryKeyEntry.SetText(savedSecondaryKey)
		}
	}

	// Handler for Remember checkbox
	rememberAPIKeyCheck.OnChanged = func(checked bool) {
		a.Preferences().SetBool("ai_remember_api_key", checked)
		if !checked {
			// Clear saved keys if user unchecks "Remember"
			a.Preferences().SetString("ai_api_key", "")
			a.Preferences().SetString("ai_secondary_api_key", "")
		} else {
			// Save current keys if user checks "Remember"
			if apiKeyEntry.Text != "" {
				a.Preferences().SetString("ai_api_key", apiKeyEntry.Text)
			}
			if secondaryKeyEntry.Text != "" {
				a.Preferences().SetString("ai_secondary_api_key", secondaryKeyEntry.Text)
			}
		}
	}

	// gst path configuration
	gstPathLabel := widget.NewLabel(T("ai.gst_path"))
	gstPathEntry := widget.NewEntry()
	gstPathEntry.SetPlaceHolder(T("ai.gst_path_hint"))

	// Load saved GST path from preferences, fallback to findGSTPath()
	savedGSTPath := a.Preferences().StringWithFallback("ai_gst_path", "")
	if savedGSTPath != "" {
		gstPathEntry.SetText(savedGSTPath)
	} else {
		gstPathEntry.SetText(findGSTPath())
	}

	// Save GST path when it changes
	gstPathEntry.OnChanged = func(newPath string) {
		// Save the path immediately when user changes it
		a.Preferences().SetString("ai_gst_path", newPath)
	}

	var installGSTBtn *widget.Button
	installGSTBtn = widget.NewButton("Install/Update GST", func() {
		pythonCmds := []string{"python3", "python", "py"}
		var python string
		for _, candidate := range pythonCmds {
			if p, err := exec.LookPath(candidate); err == nil {
				python = p
				break
			}
		}
		if python == "" {
			dialog.ShowError(fmt.Errorf("could not find python (tried: python3, python, py)"), w)
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not resolve home directory: %v", err), w)
			return
		}

		venvDir := filepath.Join(homeDir, ".subtitle-forge", "gst-venv")
		venvBin := filepath.Join(venvDir, "bin")
		venvPython := filepath.Join(venvBin, "python3")
		venvGST := filepath.Join(venvBin, "gst")

		installGSTBtn.Disable()
		progress := dialog.NewProgressInfinite("GST", "Installing/updating GST in a local virtual environment...", w)
		progress.Show()

		go func() {
			_ = os.MkdirAll(filepath.Dir(venvDir), 0755)

			if _, err := os.Stat(venvPython); err != nil {
				cmd := exec.Command(python, "-m", "venv", venvDir)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fyne.Do(func() {
						progress.Hide()
						installGSTBtn.Enable()
						dialog.ShowError(fmt.Errorf("GST venv creation failed: %v\n\n%s", err, string(output)), w)
					})
					return
				}
			}

			cmdUpgradePip := exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "pip")
			upgradePipOut, upgradePipErr := cmdUpgradePip.CombinedOutput()
			if upgradePipErr != nil {
				fyne.Do(func() {
					progress.Hide()
					installGSTBtn.Enable()
					dialog.ShowError(fmt.Errorf("pip upgrade failed: %v\n\n%s", upgradePipErr, string(upgradePipOut)), w)
				})
				return
			}

			cmdInstall := exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "gemini-srt-translator")
			installOut, installErr := cmdInstall.CombinedOutput()

			fyne.Do(func() {
				progress.Hide()
				installGSTBtn.Enable()

				if installErr != nil {
					dialog.ShowError(fmt.Errorf("GST install failed: %v\n\n%s", installErr, string(installOut)), w)
					return
				}

				if _, err := os.Stat(venvGST); err == nil {
					gstPathEntry.SetText(venvGST)
					a.Preferences().SetString("ai_gst_path", venvGST)
					dialog.ShowInformation("GST", "GST installed/updated successfully.", w)
					return
				}

				newPath := findGSTPath()
				gstPathEntry.SetText(newPath)
				a.Preferences().SetString("ai_gst_path", newPath)
				dialog.ShowInformation("GST", "GST installed/updated successfully.", w)
			})
		}()
	})
	installGSTBtn.Importance = widget.MediumImportance

	gstPathRow := container.NewBorder(nil, nil, nil, installGSTBtn, gstPathEntry)

	// Model selection for GST
	modelSelect := widget.NewSelect([]string{
		"gemini-2.5-flash (Recommended)",
		"gemini-2.5-pro (Higher Quality)",
		"gemini-3-flash-preview (Latest 3.0)",
		"gemini-3-pro-image-preview (3.0 Image)",
		"gemini-1.5-flash (Fast & Cheap)",
		"gemini-1.5-pro (Balanced)",
	}, nil)
	modelSelect.SetSelected("gemini-2.5-flash (Recommended)")

	// Language selection
	sourceLanguageSelect := widget.NewSelect([]string{
		"Auto-detect",
		"English", "Spanish", "French", "German", "Italian", "Portuguese",
		"Russian", "Japanese", "Korean", "Chinese (Simplified)", "Chinese (Traditional)",
		"Arabic", "Hindi", "Dutch", "Swedish", "Norwegian", "Danish",
	}, nil)
	sourceLanguageSelect.SetSelected("Auto-detect")

	// Create language map for system language detection
	languageMap := map[string]string{
		"English":               "eng",
		"Spanish":               "spa",
		"French":                "fre",
		"German":                "ger",
		"Italian":               "ita",
		"Portuguese":            "por",
		"Russian":               "rus",
		"Japanese":              "jpn",
		"Korean":                "kor",
		"Chinese (Simplified)":  "chi",
		"Chinese (Traditional)": "chi",
		"Arabic":                "ara",
		"Hindi":                 "hin",
		"Dutch":                 "dut",
		"Swedish":               "swe",
		"Norwegian":             "nor",
		"Danish":                "dan",
	}

	// Get system language and set as default target
	systemLang := getSystemLanguage(languageMap)
	targetLanguageSelect := widget.NewSelect([]string{
		"English", "Spanish", "French", "German", "Italian", "Portuguese",
		"Russian", "Japanese", "Korean", "Chinese (Simplified)", "Chinese (Traditional)",
		"Arabic", "Hindi", "Dutch", "Swedish", "Norwegian", "Danish",
		"Brazilian Portuguese", "Mexican Spanish", "Canadian French",
	}, nil)
	// Set default target language to system language if available, otherwise fall back to English
	targetLanguageSelect.SetSelected(systemLang)

	// File selection
	var inputFiles []string

	inputLabel := widget.NewLabel(T("ai.no_files"))
	inputLabel.Wrapping = fyne.TextWrapWord

	// File list for batch mode
	fileList := container.NewVBox()
	fileListScroll := container.NewScroll(fileList)
	fileListScroll.SetMinSize(fyne.NewSize(600, 150))

	var updateFileList func()
	updateFileList = func() {
		fileList.Objects = nil
		for i, file := range inputFiles {
			fileName := filepath.Base(file)
			fileRow := container.NewHBox(
				widget.NewLabel(fmt.Sprintf("%d.", i+1)),
				widget.NewLabel(fileName),
				widget.NewButton(T("common.remove"), func(index int) func() {
					return func() {
						inputFiles = append(inputFiles[:index], inputFiles[index+1:]...)
						updateFileList()
						if len(inputFiles) == 0 {
							inputLabel.SetText(T("ai.no_files"))
						} else {
							inputLabel.SetText(Tf("ai.files_selected", len(inputFiles)))
						}
					}
				}(i)),
			)
			fileList.Add(fileRow)
		}
		fileList.Refresh()
	}

	// Function to add file from drag & drop
	addFileFromDragDrop := func(filePath string) {
		// Check if file already exists
		for _, existing := range inputFiles {
			if existing == filePath {
				return // File already added
			}
		}

		inputFiles = append(inputFiles, filePath)
		if len(inputFiles) == 1 {
			inputLabel.SetText(T("whisper.selected") + filepath.Base(filePath))
		} else {
			inputLabel.SetText(Tf("ai.files_selected", len(inputFiles)))
		}
		updateFileList()
	}

	// Assign to global variable so main.go can access it
	aiTranslationAddFile = addFileFromDragDrop

	// Single file selection
	selectSingleBtn := widget.NewButton(T("ai.select_file"), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			inputFiles = []string{reader.URI().Path()}
			inputLabel.SetText("Selected: " + filepath.Base(reader.URI().Path()))
			updateFileList()
		}, w)
	})

	// Batch file selection
	selectBatchBtn := widget.NewButton(T("ai.select_batch"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			// Find all subtitle files in the selected folder
			inputFiles = []string{}
			supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub"}

			filepath.Walk(uri.Path(), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() {
					ext := strings.ToLower(filepath.Ext(path))
					for _, supportedExt := range supportedExts {
						if ext == supportedExt {
							inputFiles = append(inputFiles, path)
							break
						}
					}
				}
				return nil
			})

			updateFileList()
			inputLabel.SetText(Tf("ai.files_selected", len(inputFiles)))
		}, w)
	})

	// Advanced options
	advancedExpander := widget.NewAccordion()

	// Translation settings
	temperatureSlider := widget.NewSlider(0.0, 2.0)
	temperatureSlider.SetValue(0.3)
	temperatureEntry := widget.NewEntry()
	temperatureEntry.SetText("0.3")
	temperatureEntry.SetPlaceHolder("0.0-2.0")
	temperatureLabel := widget.NewLabel(T("ai.temperature"))

	// Sync slider -> entry
	temperatureSlider.OnChanged = func(value float64) {
		temperatureEntry.SetText(fmt.Sprintf("%.2f", value))
	}

	// Sync entry -> slider
	temperatureEntry.OnChanged = func(text string) {
		var value float64
		if _, err := fmt.Sscanf(text, "%f", &value); err == nil {
			if value >= 0.0 && value <= 2.0 {
				temperatureSlider.SetValue(value)
			}
		}
	}

	batchSizeEntry := widget.NewEntry()
	batchSizeEntry.SetText("500")
	batchSizeEntry.SetPlaceHolder(T("ai.batch_size_hint"))

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetPlaceHolder(T("ai.description_hint"))
	descriptionEntry.Resize(fyne.NewSize(400, 60))

	// Thinking controls
	thinkingBudgetEntry := widget.NewEntry()
	thinkingBudgetEntry.SetText("")
	thinkingBudgetEntry.SetPlaceHolder(T("ai.thinking_budget_hint"))

	thinkingLevelSelect := widget.NewSelect([]string{
		"", "minimal", "low", "medium", "high",
	}, nil)
	thinkingLevelSelect.SetSelected("")

	// Logging options
	progressLogCheck := widget.NewCheck(T("ai.save_progress"), nil)
	progressLogCheck.SetChecked(true)

	thoughtsLogCheck := widget.NewCheck(T("ai.save_thoughts"), nil)

	// Resume options
	resumeModeCheck := widget.NewCheck(T("ai.auto_resume"), nil)
	resumeModeCheck.SetChecked(true)

	startLineEntry := widget.NewEntry()
	startLineEntry.SetPlaceHolder(T("ai.start_line_hint"))

	advancedSettings := container.NewVBox(
		widget.NewLabel(T("ai.translation_quality")),
		container.NewHBox(temperatureLabel, temperatureEntry, temperatureSlider),
		widget.NewLabel(T("ai.temp_tip")),
		container.NewHBox(widget.NewLabel(T("ai.batch_size")), batchSizeEntry),
		widget.NewSeparator(),

		widget.NewLabel(T("ai.content_context")),
		widget.NewLabel(T("ai.description")),
		descriptionEntry,
		widget.NewSeparator(),

		widget.NewLabel(T("ai.thinking_settings")),
		container.NewHBox(widget.NewLabel(T("ai.thinking_budget")), thinkingBudgetEntry),
		container.NewHBox(widget.NewLabel(T("ai.thinking_level")), thinkingLevelSelect),
		widget.NewLabel(T("ai.thinking_budget_tip")),
		widget.NewLabel(T("ai.thinking_level_tip")),
		widget.NewSeparator(),

		widget.NewLabel(T("ai.logging_resume")),
		progressLogCheck,
		thoughtsLogCheck,
		resumeModeCheck,
		container.NewHBox(widget.NewLabel(T("ai.start_line")), startLineEntry),
	)

	advancedExpander.Append(widget.NewAccordionItem(T("ai.advanced_settings"), advancedSettings))

	// Output directory selection
	var outputDir string
	outputLabel := widget.NewLabel(T("ai.output_same_dir"))

	selectOutputBtn := widget.NewButton(T("ai.select_output"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText(T("ai.output_prefix") + outputDir)
		}, w)
	})

	// Translation progress
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel(T("ai.ready"))

	// Action buttons - declare first to avoid forward reference issues
	var translateBtn, stopBtn *widget.Button

	translateBtn = widget.NewButton(T("ai.start_translation"), nil)
	stopBtn = widget.NewButton(T("ai.stop_translation"), nil)

	// Set button callbacks after declaration
	translateBtn.OnTapped = func() {
		if len(inputFiles) == 0 {
			dialog.ShowError(fmt.Errorf("please select subtitle files to translate"), w)
			return
		}

		// API key is optional for GST (uses GST's own key management)

		// Reset cancellation flag and enable stop button
		cancelTranslation = false
		stopBtn.Enable()
		translateBtn.Disable()

		// Save API keys if "Remember" is enabled
		if rememberAPIKeyCheck.Checked {
			a.Preferences().SetString("ai_api_key", apiKeyEntry.Text)
			if secondaryKeyEntry.Text != "" {
				a.Preferences().SetString("ai_secondary_api_key", secondaryKeyEntry.Text)
			}
		}

		// Always save GST path when starting translation
		a.Preferences().SetString("ai_gst_path", gstPathEntry.Text)

		// Get model name and apply smart thinking defaults
		modelName := strings.Split(modelSelect.Selected, " ")[0]
		thinkingBudget := parseInt(thinkingBudgetEntry.Text, 0)
		thinkingLevel := thinkingLevelSelect.Selected

		// Apply smart defaults based on model
		if strings.Contains(modelName, "3-") {
			// Gemini 3.0 models: use thinking_level, ignore thinking_budget
			if thinkingLevel == "" {
				thinkingLevel = "low" // Default for Gemini 3.0
			}
			thinkingBudget = 0 // Don't send thinking_budget for 3.0 models
		} else if strings.Contains(modelName, "2.5-") {
			// Gemini 2.5 models: use thinking_budget (if set), ignore thinking_level
			thinkingLevel = "" // Don't send thinking_level for 2.5 models
		}

		// Create translation config
		config := AITranslationConfig{
			GSTPath:         gstPathEntry.Text,
			APIKey:          apiKeyEntry.Text,
			SecondaryAPIKey: secondaryKeyEntry.Text,
			Model:           modelName,
			Temperature:     parseFloat(temperatureEntry.Text, -1),
			BatchSize:       parseInt(batchSizeEntry.Text, 500),
			Description:     descriptionEntry.Text,
			ResumeMode:      resumeModeCheck.Checked,
			ProgressLog:     progressLogCheck.Checked,
			ThoughtsLog:     thoughtsLogCheck.Checked,
			ThinkingBudget:  thinkingBudget,
			ThinkingLevel:   thinkingLevel,
		}

		// Create/show results window
		if resultsWindow == nil {
			createResultsWindow(a)
		}
		resultsWindow.Show()
		resultsWindow.RequestFocus()

		// Start translation
		startAITranslation(inputFiles, sourceLanguageSelect.Selected, targetLanguageSelect.Selected,
			outputDir, config, progressBar, progressLabel, translateBtn, stopBtn, w)
	}
	translateBtn.Importance = widget.HighImportance

	stopBtn.OnTapped = func() {
		// Set cancellation flag
		cancelTranslation = true
		progressLabel.SetText("⛔ Stopping translation...")
		stopBtn.Disable()
	}
	stopBtn.Disable() // Disabled until translation starts
	stopBtn.Importance = widget.MediumImportance

	clearBtn := widget.NewButton("Clear All", func() {
		inputFiles = []string{}
		outputDir = ""
		inputLabel.SetText("No subtitle files selected")
		outputLabel.SetText("Output: Same directory as input files")
		updateFileList()
		progressBar.SetValue(0)
		progressLabel.SetText("Ready to translate")
		if resultsArea != nil {
			aiResultsLines = nil
			resultsArea.SetText("Translation results will appear here...")
		}
	})

	// Layout
	configSection := container.NewVBox(
		widget.NewLabel("GST Configuration"),
		container.New(layout.NewFormLayout(),
			gstPathLabel, gstPathRow,
			widget.NewLabel("API Key:"), apiKeyEntry,
			widget.NewLabel("Secondary Key:"), secondaryKeyEntry,
			widget.NewLabel(""), rememberAPIKeyCheck,
			widget.NewLabel("Model:"), modelSelect,
			widget.NewLabel("Source:"), sourceLanguageSelect,
			widget.NewLabel("Target:"), targetLanguageSelect,
		),
		widget.NewSeparator(),
	)

	fileSection := container.NewVBox(
		widget.NewLabel("Input Files"),
		inputLabel,
		container.NewHBox(selectSingleBtn, selectBatchBtn),
		fileListScroll,
		widget.NewSeparator(),

		widget.NewLabel("Output Directory"),
		outputLabel,
		selectOutputBtn,
		widget.NewSeparator(),
	)

	// Show Results button
	showResultsBtn := widget.NewButton("Show Results Window", func() {
		if resultsWindow != nil {
			resultsWindow.Show()
			resultsWindow.RequestFocus()
		} else {
			createResultsWindow(a)
			resultsWindow.Show()
		}
	})

	progressSection := container.NewVBox(
		widget.NewLabel("Translation Progress"),
		progressBar,
		progressLabel,
		container.NewHBox(translateBtn, stopBtn, clearBtn, showResultsBtn),
		widget.NewSeparator(),
		widget.NewLabel("💡 Results will appear in a separate window when translation starts"),
	)

	// Main container with tabs or accordion for better organization
	mainContent := container.NewVBox(
		title,
		widget.NewSeparator(),
		configSection,
		advancedExpander,
		widget.NewSeparator(),
		fileSection,
		progressSection,
	)

	return mainContent
}

// startAITranslation begins the translation process
func startAITranslation(inputFiles []string, sourceLang, targetLang, outputDir string,
	config AITranslationConfig, progressBar *widget.ProgressBar, progressLabel *widget.Label,
	translateBtn, stopBtn *widget.Button, w fyne.Window) {

	progressLabel.SetText("Starting AI translation...")
	updateAIResultsActions(w, "")
	if resultsArea != nil {
		aiResultsLines = []string{"🤖 Initializing AI translation..."}
		resultsArea.SetText(strings.Join(aiResultsLines, "\n"))
		// Always scroll to bottom at start
		if resultsScroll != nil {
			resultsScroll.ScrollToBottom()
		}
	}

	// Helper function to check if user is near bottom (smart auto-scroll)
	isNearBottom := func() bool {
		if resultsScroll == nil {
			return false
		}
		offset := resultsScroll.Offset
		contentSize := resultsArea.MinSize()
		visibleSize := resultsScroll.Size()
		// Check if within 50 pixels of bottom
		return (contentSize.Height - offset.Y - visibleSize.Height) < 50
	}

	// Process files in a goroutine
	go func() {
		totalFiles := len(inputFiles)
		successCount := 0

		for i, inputFile := range inputFiles {
			// Check for cancellation
			if cancelTranslation {
				fyne.Do(func() {
					progressLabel.SetText(fmt.Sprintf("⛔ Translation cancelled by user (%d/%d files completed)", successCount, totalFiles))
					if resultsArea != nil {
						appendToResults(fmt.Sprintf("\n⛔ Translation cancelled by user\n✅ Completed: %d files\n⏭️ Skipped: %d files", successCount, totalFiles-i))
						// Only auto-scroll if user is near bottom
						if resultsScroll != nil && isNearBottom() {
							resultsScroll.ScrollToBottom()
						}
					}
					translateBtn.Enable()
					stopBtn.Disable()
				})
				return
			}

			fileName := filepath.Base(inputFile)

			// Update progress
			fileProgress := float64(i) / float64(totalFiles)
			fyne.Do(func() {
				progressBar.SetValue(fileProgress)
				progressLabel.SetText(fmt.Sprintf("Translating %s (%d/%d)", fileName, i+1, totalFiles))
				if resultsArea != nil {
					appendToResults(fmt.Sprintf("🔄 Processing: %s", fileName))
					// Only auto-scroll if user is near bottom
					if resultsScroll != nil && isNearBottom() {
						resultsScroll.ScrollToBottom()
					}
				}
			})

			// Determine output file
			var outputFile string
			if outputDir != "" {
				baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
				outputFile = filepath.Join(outputDir, fmt.Sprintf("%s_%s.srt", baseName, targetLang))
			} else {
				baseName := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
				outputFile = fmt.Sprintf("%s_%s.srt", baseName, targetLang)
			}

			// Show file processing start
			if resultsArea != nil {
				fyne.Do(func() {
					appendToResults("   📖 Reading subtitle file...")
					// Only auto-scroll if user is near bottom
					if resultsScroll != nil && isNearBottom() {
						resultsScroll.ScrollToBottom()
					}
				})
			}

			// Perform translation with progress updates
			success, errorMsg := translateSubtitleFileWithProgress(inputFile, outputFile, sourceLang, targetLang, config, func(status string) {
				if resultsArea != nil {
					fyne.Do(func() {
						appendToResults(fmt.Sprintf("   %s", status))
						// Only auto-scroll if user is near bottom
						if resultsScroll != nil && isNearBottom() {
							resultsScroll.ScrollToBottom()
						}
					})
				}
			})

			if success {
				successCount++
				if resultsArea != nil {
					fyne.Do(func() {
						appendToResults(fmt.Sprintf("✅ Completed: %s → %s", fileName, filepath.Base(outputFile)))
						// Only auto-scroll if user is near bottom
						if resultsScroll != nil && isNearBottom() {
							resultsScroll.ScrollToBottom()
						}
					})
				}
			} else {
				if resultsArea != nil {
					fyne.Do(func() {
						appendToResults(fmt.Sprintf("❌ Failed: %s\n   Error: %s", fileName, errorMsg))
						// Only auto-scroll if user is near bottom
						if resultsScroll != nil && isNearBottom() {
							resultsScroll.ScrollToBottom()
						}
					})
				}
			}
		}

		// Final update
		fyne.Do(func() {
			progressBar.SetValue(1.0)
			progressLabel.SetText(fmt.Sprintf(T("ai.translation_completed_status"), successCount, totalFiles))
			if resultsArea != nil {
				finalOutputDir := outputDir
				if strings.TrimSpace(finalOutputDir) == "" && len(inputFiles) > 0 {
					finalOutputDir = filepath.Dir(inputFiles[0])
				}
				appendToResults(fmt.Sprintf(T("ai.batch_complete_summary"), successCount, totalFiles-successCount, finalOutputDir))
				updateAIResultsActions(w, finalOutputDir)
				// Only auto-scroll if user is near bottom
				if resultsScroll != nil && isNearBottom() {
					resultsScroll.ScrollToBottom()
				}
			}
			translateBtn.Enable()
			stopBtn.Disable()
		})
	}()
}

// translateSubtitleFileWithError translates a single subtitle file using AI and returns error details
func translateSubtitleFileWithError(inputFile, outputFile, sourceLang, targetLang string, config AITranslationConfig) (bool, string) {
	success, err := translateSubtitleFileInternal(inputFile, outputFile, sourceLang, targetLang, config, nil)
	if err != nil {
		return false, err.Error()
	}
	return success, ""
}

// translateSubtitleFileWithProgress translates a single subtitle file with progress updates
func translateSubtitleFileWithProgress(inputFile, outputFile, sourceLang, targetLang string, config AITranslationConfig, progressCallback func(string)) (bool, string) {
	success, err := translateSubtitleFileInternal(inputFile, outputFile, sourceLang, targetLang, config, progressCallback)
	if err != nil {
		return false, err.Error()
	}
	return success, ""
}

// translateSubtitleFile translates a single subtitle file using AI
func translateSubtitleFile(inputFile, outputFile, sourceLang, targetLang string, config AITranslationConfig) bool {
	success, _ := translateSubtitleFileInternal(inputFile, outputFile, sourceLang, targetLang, config, nil)
	return success
}

// translateSubtitleFileInternal is the internal implementation
func translateSubtitleFileInternal(inputFile, outputFile, sourceLang, targetLang string, config AITranslationConfig, progressCallback func(string)) (bool, error) {
	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", inputFile, err)
		return false, err
	}

	// Remove BOM if present (UTF-8 BOM can cause first entry to be skipped)
	contentStr := string(content)
	if len(contentStr) >= 3 && contentStr[0] == '\xef' && contentStr[1] == '\xbb' && contentStr[2] == '\xbf' {
		contentStr = contentStr[3:]
	}

	// Using GST (Python) for translation - open terminal window
	if progressCallback != nil {
		progressCallback("🐍 Opening terminal window for GST translation...")
		progressCallback(fmt.Sprintf("📝 Input: %s", filepath.Base(inputFile)))
		progressCallback(fmt.Sprintf("📝 Output: %s", filepath.Base(outputFile)))
		progressCallback("💡 Watch the terminal window for live progress!")
	}
	err = translateWithGSTInTerminal(inputFile, outputFile, targetLang, config)
	if err != nil {
		return false, fmt.Errorf("failed to open terminal: %v", err)
	}
	// Return success - actual translation happens in terminal
	return true, nil
}

// Removed getSafetySettings - using GST only

// Removed getResponseSchema - using GST only

// Removed getInstruction - using GST only

// Removed cleanAIResponse - using GST only

// Removed translateBatch - using GST only

// Removed getModelOutputLimit - using GST only

// Removed countTokensAPI - using GST only

// Removed fixCommonJSONIssues - using GST only
