package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// AITranslationConfig holds all AI translation settings
type AITranslationConfig struct {
	Provider        string // "gemini", "gst"
	APIKey          string
	SecondaryAPIKey string // For quota management
	Model           string // "gemini-2.5-flash", etc.
	Temperature     float64
	BatchSize       int
	UseThinking     bool
	ThinkingBudget  int
	UseAudioContext bool
	VideoFile       string // Path to video file for audio context
	Description     string
	ResumeMode      bool
	ProgressLog     bool
	ThoughtsLog     bool
	GSTPath         string // Path to gst binary when using gst provider
}

// findGSTPath tries to find the gst executable. Prefers the user's venv path, then PATH.
func findGSTPath() string {
    preferred := "/Users/venimk/subsvenv/bin/gst"
    if _, err := os.Stat(preferred); err == nil {
        return preferred
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
	if config.UseThinking == false {
		args = append(args, "--no-thinking")
	} else if config.ThinkingBudget > 0 {
		args = append(args, "--thinking-budget", fmt.Sprintf("%d", config.ThinkingBudget))
	}
	if config.ProgressLog {
		args = append(args, "--progress-log")
	}
	if config.ThoughtsLog {
		args = append(args, "--thoughts-log")
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
		strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_progress.log",
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
    if config.UseThinking == false {
        args = append(args, "--no-thinking")
    } else if config.ThinkingBudget > 0 {
        args = append(args, "--thinking-budget", fmt.Sprintf("%d", config.ThinkingBudget))
    }
    if config.ProgressLog {
        args = append(args, "--progress-log")
    }
    if config.ThoughtsLog {
        args = append(args, "--thoughts-log")
    }
    if config.ResumeMode {
        args = append(args, "--resume")
    } else {
        args = append(args, "--no-resume")
    }
    if desc := strings.TrimSpace(config.Description); desc != "" {
        args = append(args, "-d", desc)
    }

    cmd := exec.Command(gstPath, args...)
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    if err := cmd.Start(); err != nil {
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
        return false, fmt.Errorf("gst failed: %v", err)
    }
    // gst writes the output file itself; treat as success if it completed
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

// GeminiRequest represents the API request structure
type GeminiRequest struct {
	Contents          []GeminiContent        `json:"contents"`
	GenerationConfig  GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *GeminiContent         `json:"systemInstruction,omitempty"`
	SafetySettings    []GeminiSafetySetting  `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenerationConfig struct {
	Temperature      float64              `json:"temperature,omitempty"`
	TopP             float64              `json:"topP,omitempty"`
	TopK             int                  `json:"topK,omitempty"`
	MaxOutputTokens  int                  `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string               `json:"responseMimeType,omitempty"`
	ResponseSchema   *GeminiResponseSchema `json:"responseSchema,omitempty"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiResponseSchema struct {
	Type     string                          `json:"type"`
	Items    *GeminiResponseSchema           `json:"items,omitempty"`
	Properties map[string]*GeminiResponseSchema `json:"properties,omitempty"`
	Required []string                         `json:"required,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type geminiModelInfo struct {
	OutputTokenLimit int `json:"output_token_limit"`
}

type countTokensRequest struct {
	Model    string           `json:"model"`
	Contents []GeminiContent  `json:"contents"`
}

type countTokensResponse struct {
	TotalTokens int `json:"total_tokens"`
}

// callGeminiAPI sends a generateContent request and returns the raw text response
func callGeminiAPI(request GeminiRequest, config AITranslationConfig) (string, error) {
    // Prepare request body
    jsonData, err := json.Marshal(request)
    if err != nil {
        return "", err
    }

    // Use the model name from config, normalize older aliases
    modelName := config.Model
    if modelName == "gemini-1.5-flash" {
        modelName = "gemini-2.5-flash"
    }

    // Create HTTP request
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, config.APIKey)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")

    // Make request
    client := &http.Client{Timeout: 60 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    if resp.StatusCode != 200 {
        if resp.StatusCode == 429 {
            return "", fmt.Errorf("rate limit exceeded - please wait and try again")
        } else if resp.StatusCode == 401 || resp.StatusCode == 403 {
            return "", fmt.Errorf("invalid API key - check your key at https://aistudio.google.com/app/apikey")
        } else if resp.StatusCode == 400 {
            return "", fmt.Errorf("bad request - check your input format")
        } else if resp.StatusCode >= 500 {
            return "", fmt.Errorf("Google API server error (%d) - please try again later", resp.StatusCode)
        }
        return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
    }

    // Parse response
    var geminiResp GeminiResponse
    if err := json.Unmarshal(body, &geminiResp); err != nil {
        return "", err
    }
    if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
        return "", fmt.Errorf("no translation received from API")
    }

    translatedText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
    if translatedText == "" {
        return "", fmt.Errorf("API returned empty translation")
    }
    if strings.HasPrefix(translatedText, "```") {
        lines := strings.Split(translatedText, "\n")
        if len(lines) > 2 {
            translatedText = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
        }
    }
    return translatedText, nil
}

// Global variables for AI translation
var activeTranslationJobs = make(map[string]*TranslationJob)
var aiTranslationAddFile func(string) // Function to add files from drag & drop
var cancelTranslation bool            // Flag to cancel ongoing translation
var resultsWindow fyne.Window       // Separate results window
var resultsArea *widget.Entry       // Results text area
var resultsScroll *container.Scroll // Scroll container for results

// createResultsWindow creates a separate window for translation results
func createResultsWindow(a fyne.App) {
	if resultsWindow != nil {
		return // Already created
	}

	resultsWindow = a.NewWindow("Translation Results")
	resultsWindow.Resize(fyne.NewSize(800, 600))

	// Results area with scrolling
	resultsArea = widget.NewMultiLineEntry()
	resultsArea.SetText("🤖 Translation results will appear here...\n")
	resultsArea.Wrapping = fyne.TextWrapWord
	resultsScroll = container.NewScroll(resultsArea)

	// Clear button
	clearResultsBtn := widget.NewButton("Clear Results", func() {
		resultsArea.SetText("🤖 Translation results cleared...\n")
	})

	// Copy to clipboard button
	copyBtn := widget.NewButton("Copy All", func() {
		resultsWindow.Clipboard().SetContent(resultsArea.Text)
		dialog.ShowInformation("Copied", "Results copied to clipboard", resultsWindow)
	})

	// Layout
	content := container.NewBorder(
		nil,
		container.NewHBox(clearResultsBtn, copyBtn),
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
	title := widget.NewLabel("AI-Powered Subtitle Translation")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Provider selection
	providerSelect := widget.NewSelect([]string{
		"Google Gemini AI",
		"gst (Python)",
	}, nil)
	providerSelect.SetSelected("Google Gemini AI")

	// API Key configuration
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter your API key...")

	secondaryKeyEntry := widget.NewPasswordEntry()
	secondaryKeyEntry.SetPlaceHolder("Optional: Secondary API key for more quota")

	// Remember API Key checkbox
	rememberAPIKeyCheck := widget.NewCheck("Remember API Keys (saved locally)", nil)
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

	// gst path configuration (visible only for gst provider)
	gstPathLabel := widget.NewLabel("gst Path:")
	gstPathEntry := widget.NewEntry()
	gstPathEntry.SetPlaceHolder("/path/to/gst or 'gst' if on PATH")
	gstPathEntry.SetText(findGSTPath())
	// Hidden by default (Gemini selected)
	gstPathLabel.Hide()
	gstPathEntry.Hide()

	// Model selection (dynamic based on provider)
	modelSelect := widget.NewSelect([]string{
		"gemini-2.5-flash (Recommended)",
		"gemini-2.5-pro (Higher Quality)",
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

	targetLanguageSelect := widget.NewSelect([]string{
		"English", "Spanish", "French", "German", "Italian", "Portuguese",
		"Russian", "Japanese", "Korean", "Chinese (Simplified)", "Chinese (Traditional)",
		"Arabic", "Hindi", "Dutch", "Swedish", "Norwegian", "Danish",
		"Brazilian Portuguese", "Mexican Spanish", "Canadian French",
	}, nil)
	targetLanguageSelect.SetSelected("English")

	// File selection
	var inputFiles []string

	inputLabel := widget.NewLabel("No subtitle files selected")
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
				widget.NewButton("Remove", func(index int) func() {
					return func() {
						inputFiles = append(inputFiles[:index], inputFiles[index+1:]...)
						updateFileList()
						if len(inputFiles) == 0 {
							inputLabel.SetText("No subtitle files selected")
						} else {
							inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch translation", len(inputFiles)))
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
			inputLabel.SetText("Selected: " + filepath.Base(filePath))
		} else {
			inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch translation", len(inputFiles)))
		}
		updateFileList()
	}

	// Assign to global variable so main.go can access it
	aiTranslationAddFile = addFileFromDragDrop

	// Single file selection
	selectSingleBtn := widget.NewButton("Select Subtitle File", func() {
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
	selectBatchBtn := widget.NewButton("Select Multiple Files (Batch)", func() {
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
			inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch translation", len(inputFiles)))
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
	temperatureLabel := widget.NewLabel("Temperature:")
	
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
	batchSizeEntry.SetText("100")
	batchSizeEntry.SetPlaceHolder("Batch size (lines per request)")

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetPlaceHolder("Optional: Describe the content context (e.g., 'Medical TV series, use medical terminology')")
	descriptionEntry.Resize(fyne.NewSize(400, 60))

	// AI-specific options
	useThinkingCheck := widget.NewCheck("Enable AI Thinking (Gemini 2.5 only)", nil)
	useThinkingCheck.SetChecked(true)

	thinkingBudgetSlider := widget.NewSlider(0, 24576)
	thinkingBudgetSlider.SetValue(2048)
	thinkingBudgetLabel := widget.NewLabel("Thinking Budget: 2048 tokens")
	thinkingBudgetSlider.OnChanged = func(value float64) {
		thinkingBudgetLabel.SetText(fmt.Sprintf("Thinking Budget: %.0f tokens", value))
	}

	useAudioContextCheck := widget.NewCheck("Extract audio context from video (requires FFmpeg)", nil)

	// Logging options
	progressLogCheck := widget.NewCheck("Save progress log", nil)
	progressLogCheck.SetChecked(true)

	thoughtsLogCheck := widget.NewCheck("Save AI thinking process log", nil)

	// Resume options
	resumeModeCheck := widget.NewCheck("Auto-resume interrupted translations", nil)
	resumeModeCheck.SetChecked(true)

	startLineEntry := widget.NewEntry()
	startLineEntry.SetPlaceHolder("Start from line (optional)")

	advancedSettings := container.NewVBox(
		widget.NewLabel("Translation Quality"),
		container.NewHBox(temperatureLabel, temperatureEntry, temperatureSlider),
		widget.NewLabel("💡 Tip: Lower values (0.1-0.3) = more consistent, Higher (0.6-1.0) = more creative"),
		container.NewHBox(widget.NewLabel("Batch Size:"), batchSizeEntry),
		widget.NewSeparator(),

		widget.NewLabel("Content Context"),
		widget.NewLabel("Description:"),
		descriptionEntry,
		widget.NewSeparator(),

		widget.NewLabel("AI Features (Gemini)"),
		useThinkingCheck,
		container.NewHBox(thinkingBudgetLabel, thinkingBudgetSlider),
		useAudioContextCheck,
		widget.NewSeparator(),

		widget.NewLabel("Logging & Resume"),
		progressLogCheck,
		thoughtsLogCheck,
		resumeModeCheck,
		container.NewHBox(widget.NewLabel("Start Line:"), startLineEntry),
	)

	advancedExpander.Append(widget.NewAccordionItem("Advanced Settings", advancedSettings))

	// Output directory selection
	var outputDir string
	outputLabel := widget.NewLabel("Output: Same directory as input files")

	selectOutputBtn := widget.NewButton("Select Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText("Output: " + outputDir)
		}, w)
	})

	// Translation progress
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("Ready to translate")

	// Action buttons - declare first to avoid forward reference issues
	var translateBtn, stopBtn *widget.Button
	
	translateBtn = widget.NewButton("Start Translation", nil)
	stopBtn = widget.NewButton("Stop Translation", nil)
	
	// Set button callbacks after declaration
	translateBtn.OnTapped = func() {
		if len(inputFiles) == 0 {
			dialog.ShowError(fmt.Errorf("please select subtitle files to translate"), w)
			return
		}

		// Skip API key validation for gst provider
		if providerSelect.Selected != "gst (Python)" {
			if apiKeyEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("Please enter your Google Gemini API key.\n\nGet your free API key at:\nhttps://aistudio.google.com/app/apikey"), w)
				return
			}

			// Validate API key format (basic check)
			if len(apiKeyEntry.Text) < 20 || !strings.HasPrefix(apiKeyEntry.Text, "AI") {
				dialog.ShowError(fmt.Errorf("Invalid API key format.\n\nGemini API keys should start with 'AI' and be at least 20 characters long."), w)
				return
			}
		}

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

		// Create translation config
		config := AITranslationConfig{
			Provider:        strings.ToLower(strings.Split(providerSelect.Selected, " ")[0]),
			APIKey:          apiKeyEntry.Text,
			SecondaryAPIKey: secondaryKeyEntry.Text,
			Model:           strings.Split(modelSelect.Selected, " ")[0],
			Temperature:     parseFloat(temperatureEntry.Text, 0.3),
			BatchSize:       parseInt(batchSizeEntry.Text, 100),
			UseThinking:     useThinkingCheck.Checked,
			ThinkingBudget:  int(thinkingBudgetSlider.Value),
			UseAudioContext: useAudioContextCheck.Checked,
			Description:     descriptionEntry.Text,
			ResumeMode:      resumeModeCheck.Checked,
			ProgressLog:     progressLogCheck.Checked,
			ThoughtsLog:     thoughtsLogCheck.Checked,
			GSTPath:         gstPathEntry.Text,
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
			resultsArea.SetText("Translation results will appear here...")
		}
	})

	// Layout
	configSection := container.NewVBox(
		widget.NewLabel("AI Provider & Authentication"),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Provider:"), providerSelect,
			widget.NewLabel("API Key:"), apiKeyEntry,
			widget.NewLabel("Secondary Key:"), secondaryKeyEntry,
			widget.NewLabel(""), rememberAPIKeyCheck,
			widget.NewLabel("Model:"), modelSelect,
			gstPathLabel, gstPathEntry,
		),
		widget.NewSeparator(),

		widget.NewLabel("Languages"),
		container.New(layout.NewFormLayout(),
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

	// Provider onChange handler - after all widgets are created
	providerSelect.OnChanged = func(selected string) {
		// Allow only Gemini and gst
		if selected != "Google Gemini AI" && selected != "gst (Python)" {
			dialog.ShowInformation("Provider Not Available",
				"Only Google Gemini AI and gst (Python) are currently supported.", w)
			providerSelect.SetSelected("Google Gemini AI")
			selected = "Google Gemini AI"
		}

		if selected == "gst (Python)" {
			// Show gst path, hide Gemini-only controls
			gstPathLabel.Show()
			gstPathEntry.Show()
			useThinkingCheck.Hide()
			thinkingBudgetLabel.Hide()
			thinkingBudgetSlider.Hide()
			useAudioContextCheck.Hide()
		} else {
			gstPathLabel.Hide()
			gstPathEntry.Hide()
			useThinkingCheck.Show()
			thinkingBudgetLabel.Show()
			thinkingBudgetSlider.Show()
			useAudioContextCheck.Show()
		}
		configSection.Refresh()
	}

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

	// Initialize visibility for the default selection
	providerSelect.OnChanged(providerSelect.Selected)

	return mainContent
}

// startAITranslation begins the translation process
func startAITranslation(inputFiles []string, sourceLang, targetLang, outputDir string,
	config AITranslationConfig, progressBar *widget.ProgressBar, progressLabel *widget.Label,
	translateBtn, stopBtn *widget.Button, w fyne.Window) {

	progressLabel.SetText("Starting AI translation...")
	if resultsArea != nil {
		resultsArea.SetText("🤖 Initializing AI translation...\n")
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
						resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n\n⛔ Translation cancelled by user\n✅ Completed: %d files\n⏭️ Skipped: %d files", successCount, totalFiles-i))
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
					resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n🔄 Processing: %s", fileName))
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
					resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n🔄 Processing: %s", fileName))
					resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n   📖 Reading subtitle file..."))
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
						resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n   %s", status))
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
						resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n✅ Completed: %s → %s", fileName, filepath.Base(outputFile)))
						// Only auto-scroll if user is near bottom
						if resultsScroll != nil && isNearBottom() {
							resultsScroll.ScrollToBottom()
						}
					})
				}
			} else {
				if resultsArea != nil {
					fyne.Do(func() {
						resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n❌ Failed: %s\n   Error: %s", fileName, errorMsg))
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
			progressLabel.SetText(fmt.Sprintf("Translation completed: %d/%d files successful", successCount, totalFiles))
			if resultsArea != nil {
				resultsArea.SetText(resultsArea.Text + fmt.Sprintf("\n\n🎉 Batch translation completed!\n✅ Success: %d files\n❌ Failed: %d files", successCount, totalFiles-successCount))
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

	// Parse SRT content
	if progressCallback != nil {
		progressCallback("🔍 Parsing subtitle entries...")
	}
	entries, err := parseSRT(contentStr)
	if err != nil {
		fmt.Printf("Error parsing SRT file %s: %v\n", inputFile, err)
		return false, err
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("📊 Found %d subtitle entries", len(entries)))
	}

    // If provider is gst (Python), open terminal window
    if strings.ToLower(config.Provider) == "gst" {
        if progressCallback != nil {
            progressCallback("🐍 Opening terminal window for gst translation...")
            progressCallback(fmt.Sprintf("📝 Input: %s", filepath.Base(inputFile)))
            progressCallback(fmt.Sprintf("📝 Output: %s", filepath.Base(outputFile)))
            progressCallback("💡 Watch the terminal window for live progress!")
        }
        err := translateWithGSTInTerminal(inputFile, outputFile, targetLang, config)
        if err != nil {
            return false, fmt.Errorf("failed to open terminal: %v", err)
        }
        // Return success - actual translation happens in terminal
        return true, nil
    }

    // Translate in batches
	translatedEntries := make([]SubtitleEntry, len(entries))
	batchSize := config.BatchSize
	totalBatches := (len(entries) + batchSize - 1) / batchSize
	// Previous message context like gst
	var previousMsgs []GeminiContent
	// Token limit cache
	modelLimit := 0
	if lim, err := getModelOutputLimit(config.Model, config.APIKey); err == nil {
		modelLimit = lim
	}

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("🚀 Starting translation in %d batches (batch size: %d)", totalBatches, batchSize))
	}

	for i := 0; i < len(entries); i += batchSize {
		// propose end
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}

		// Token preflight: shrink until under 90% of model limit
		if modelLimit > 0 {
			for {
				// Build temporary user content for [i:end]
				type tmpIn struct{ Index string `json:"index"`; Content string `json:"content"` }
				tmpArr := make([]tmpIn, 0, end-i)
				for j := i; j < end; j++ {
					tmpArr = append(tmpArr, tmpIn{Index: fmt.Sprintf("%d", j-i), Content: entries[j].Text})
				}
				b, _ := json.Marshal(tmpArr)
				contents := append([]GeminiContent{}, previousMsgs...)
				contents = append(contents, GeminiContent{Role: "user", Parts: []GeminiPart{{Text: string(b)}}})

				total, err := countTokensAPI(config.Model, config.APIKey, contents)
				if err != nil {
					break // fallback: proceed without shrinking
				}
				if total <= int(float64(modelLimit)*0.9) {
					break
				}
				// shrink by half until at least 1
				newLen := (end - i) / 2
				if newLen < 1 { newLen = 1 }
				end = i + newLen
				if progressCallback != nil {
					progressCallback(fmt.Sprintf("🔧 Reducing batch size due to token limit: now %d", end-i))
				}
			}
		}

		batch := entries[i:end]
		batchNum := (i / batchSize) + 1

		if progressCallback != nil {
			progressCallback(fmt.Sprintf("🔧 Processing batch %d: %d texts", batchNum, len(batch)))
		}

		// Check for cancellation
		if cancelTranslation {
			return false, fmt.Errorf("translation cancelled by user")
		}

		translatedBatch, nextPrev, err := translateBatch(batch, sourceLang, targetLang, config, previousMsgs)
		if err != nil {
			fmt.Printf("Error translating batch %d: %v\n", batchNum, err)
			if progressCallback != nil {
				// Provide more helpful error messages
				errorMsg := err.Error()
				if strings.Contains(errorMsg, "API error") {
					if strings.Contains(errorMsg, "429") {
						progressCallback(fmt.Sprintf("❌ Rate limit exceeded. Please wait a moment and try again."))
					} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "403") {
						progressCallback(fmt.Sprintf("❌ Invalid API key. Please check your key at https://aistudio.google.com/app/apikey"))
					} else if strings.Contains(errorMsg, "quota") {
						progressCallback(fmt.Sprintf("❌ API quota exceeded. Check your usage at https://aistudio.google.com/"))
					} else {
						progressCallback(fmt.Sprintf("❌ Batch %d/%d failed: %s", batchNum, totalBatches, errorMsg))
					}
				} else {
					progressCallback(fmt.Sprintf("❌ Batch %d/%d failed: %s", batchNum, totalBatches, errorMsg))
				}
			}
			return false, fmt.Errorf("batch %d failed: %v", batchNum, err)
		}

		// Validate translated batch has correct number of entries
		if len(translatedBatch) != len(batch) {
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("⚠️ Translation count mismatch - attempting to recover"))
			}
			// Attempt to recover by padding with original text
			if len(translatedBatch) < len(batch) {
				for i := len(translatedBatch); i < len(batch); i++ {
					translatedBatch = append(translatedBatch, batch[i])
				}
			} else {
				// Truncate excess
				translatedBatch = translatedBatch[:len(batch)]
			}
		}

		if progressCallback != nil {
			progressCallback(fmt.Sprintf("✅ Batch %d completed: %d translations", batchNum, len(translatedBatch)))
		}

		copy(translatedEntries[i:end], translatedBatch)
		previousMsgs = nextPrev
	}

	// Generate output SRT
	if progressCallback != nil {
		progressCallback("📝 Generating output SRT file...")
	}
	outputContent := generateSRT(translatedEntries)

	// Write output file
	if progressCallback != nil {
		progressCallback("💾 Saving translated file...")
	}
	err = os.WriteFile(outputFile, []byte(outputContent), 0644)
	if err != nil {
		fmt.Printf("Error writing output file %s: %v\n", outputFile, err)
		return false, err
	}

	return true, nil
}

// getSafetySettings returns safety settings to disable all content filtering
func getSafetySettings() []GeminiSafetySetting {
	return []GeminiSafetySetting{
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
	}
}

// getResponseSchema returns the structured JSON schema for translation response
func getResponseSchema() *GeminiResponseSchema {
	return &GeminiResponseSchema{
		Type: "array",
		Items: &GeminiResponseSchema{
			Type: "object",
			Properties: map[string]*GeminiResponseSchema{
				"index": {Type: "string"},
				"content": {Type: "string"},
			},
			Required: []string{"index", "content"},
		},
	}
}

// getInstruction generates detailed translation instruction based on target language
func getInstruction(targetLang string, useAudioContext bool, audioFile string, description string) string {
	fields := "- index: a string identifier\n- content: the text to translate\n"
	
	if useAudioContext && audioFile != "" {
		fields += "- time_start: the start time of the segment\n- time_end: the end time of the segment\n"
	}

	instruction := fmt.Sprintf(`You are an assistant that translates subtitles from any language to %s.
You will receive a list of objects, each with these fields:

%s
Translate the 'content' field of each object.
If the 'content' field is empty, leave it as is.
Preserve line breaks, formatting, and special characters.
Do NOT move or merge 'content' between objects.
Do NOT add or remove any objects.
Do NOT alter the 'index' field.`, targetLang, fields)

	if useAudioContext && audioFile != "" {
		instruction += fmt.Sprintf(`

You will also receive an audio file.
Use the time_start and time_end of each object to analyze the audio.
Analyze the speaker's voice in the audio to determine gender, then apply grammatical gender rules for %s:
1. Listen for voice characteristics to identify if speaker is male/female:
   - Use masculine verb forms/adjectives if speaker sounds male
   - Use feminine verb forms/adjectives if speaker sounds female
   - Apply gender agreement to: verbs, adjectives, past participles, pronouns
   - Example: French 'I am tired' -> 'Je suis fatigué' (male) vs 'Je suis fatiguée' (female)
2. In some cases you also need to identify who the current speaker is talking to:
   - If the speaker is talking to a male, use masculine forms.
   - If the speaker is talking to a female, use feminine forms.
   - If the speaker is talking to a group, use plural forms.
   - Example: Portuguese 'You are tired' -> 'Você está cansado' (male) vs 'Você está cansada' (female)
   - Example: Spanish 'You are tired' (group) -> 'Ustedes están cansados' (male/general group) vs 'Ustedes están cansadas' (female group)`, targetLang)
	}

	if description != "" {
		instruction += fmt.Sprintf("\n\nAdditional user instruction:\n\n%s", description)
	}

	return instruction
}

// cleanAIResponse removes AI thinking/reasoning from response
func cleanAIResponse(response string) string {
	// If response starts with "THOUGHT:" or similar, remove it
	if strings.Contains(response, "THOUGHT:") {

		// Try to extract just the translations by looking for patterns like "*Translation:*"
		lines := strings.Split(response, "\n")
		var cleanedParts []string
		var currentTranslation strings.Builder
		inTranslation := false

		for _, line := range lines {
			// Look for translation markers like "*Translation:*"
			if strings.Contains(line, "*Translation:*") {
				// Extract the translation part after the marker
				parts := strings.Split(line, "*Translation:*")
				if len(parts) > 1 {
					translation := strings.TrimSpace(parts[1])
					if currentTranslation.Len() > 0 {
						cleanedParts = append(cleanedParts, currentTranslation.String())
						currentTranslation.Reset()
					}
					currentTranslation.WriteString(translation)
					inTranslation = true
				}
				continue
			}

			// Skip lines with entry markers like "**Entry"
			if strings.HasPrefix(strings.TrimSpace(line), "**Entry") {
				if inTranslation && currentTranslation.Len() > 0 {
					cleanedParts = append(cleanedParts, currentTranslation.String())
					currentTranslation.Reset()
				}
				inTranslation = false
				continue
			}

			// If we're in a translation and line doesn't look like metadata, include it
			if inTranslation && !strings.Contains(line, "THOUGHT:") && !strings.Contains(line, "I need to") {
				if currentTranslation.Len() > 0 {
					currentTranslation.WriteString("\n")
				}
				currentTranslation.WriteString(line)
			}
		}

		// Add the last translation
		if currentTranslation.Len() > 0 {
			cleanedParts = append(cleanedParts, currentTranslation.String())
		}

		if len(cleanedParts) > 0 {
			return strings.Join(cleanedParts, "\n---SUBTITLE_SEPARATOR---\n")
		}
	}

	// No cleaning needed
	return response
}

// translateBatch translates subtitle entries using structured JSON with schema, and returns next previous-message context
func translateBatch(entries []SubtitleEntry, sourceLang, targetLang string, config AITranslationConfig, previous []GeminiContent) ([]SubtitleEntry, []GeminiContent, error) {

    // Build input JSON with index and content fields
    type InputEntry struct {
        Index   string `json:"index"`
        Content string `json:"content"`
    }
    var inputEntries []InputEntry
    for i, entry := range entries {
        inputEntries = append(inputEntries, InputEntry{Index: fmt.Sprintf("%d", i), Content: entry.Text})
    }

    // Marshal to JSON for the user part
    inputJSON, _ := json.Marshal(inputEntries)
    userPart := GeminiContent{Role: "user", Parts: []GeminiPart{{Text: string(inputJSON)}}}

    // Create API request with structured JSON response
    request := GeminiRequest{
        SystemInstruction: &GeminiContent{
            Parts: []GeminiPart{{Text: getInstruction(targetLang, config.UseAudioContext, config.VideoFile, config.Description)}},
            Role:  "system",
        },
        Contents: append(append([]GeminiContent{}, previous...), userPart),
        GenerationConfig: GeminiGenerationConfig{
            Temperature:      config.Temperature,
            TopP:             0.95,
            TopK:             40,
            MaxOutputTokens:  8192,
            ResponseMimeType: "application/json",
            ResponseSchema:   getResponseSchema(),
        },
        SafetySettings: getSafetySettings(),
    }

    // Make API call
    translatedJSON, err := callGeminiAPI(request, config)
    if err != nil {
        return nil, nil, err
    }

    // Parse JSON response
    type OutputEntry struct {
        Index   string `json:"index"`
        Content string `json:"content"`
    }
    
    var outputEntries []OutputEntry
    err = json.Unmarshal([]byte(translatedJSON), &outputEntries)
    if err != nil {
        // Try to fix common JSON issues
        fixedJSON := fixCommonJSONIssues(translatedJSON)
        err = json.Unmarshal([]byte(fixedJSON), &outputEntries)
        if err != nil {
            return nil, nil, fmt.Errorf("failed to parse translation response: %v\nResponse: %s", err, translatedJSON)
        }
    }

    // Validate count matches
    if len(outputEntries) != len(entries) {
        return nil, nil, fmt.Errorf("translation count mismatch: expected %d, got %d", len(entries), len(outputEntries))
    }

    // Build result entries (preserve original timing and indexes)
    var translatedEntries []SubtitleEntry
    for i, entry := range entries {
        translatedText := strings.TrimSpace(outputEntries[i].Content)

        translatedEntries = append(translatedEntries, SubtitleEntry{
            Index:     entry.Index,     // Preserve original index
            StartTime: entry.StartTime, // Preserve original timing
            EndTime:   entry.EndTime,   // Preserve original timing
            Text:      translatedText,  // Use translated content
        })
    }

    // Build previous messages for next batch: user original subset and model translated subset
    modelArr, _ := json.Marshal(outputEntries)
    nextPrevious := []GeminiContent{
        {Role: "user", Parts: []GeminiPart{{Text: string(inputJSON)}}},
        {Role: "model", Parts: []GeminiPart{{Text: string(modelArr)}}},
    }

    return translatedEntries, nextPrevious, nil
}

// getModelOutputLimit fetches model metadata to get output token limit
func getModelOutputLimit(modelName, apiKey string) (int, error) {
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s?key=%s", modelName, apiKey)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return 0, err
    }
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return 0, fmt.Errorf("model info error (%d): %s", resp.StatusCode, string(body))
    }
    var m map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
        return 0, err
    }
    if v, ok := m["output_token_limit"].(float64); ok {
        return int(v), nil
    }
    return 0, fmt.Errorf("output_token_limit not found")
}

// countTokensAPI calls the token counting endpoint for given contents
func countTokensAPI(modelName, apiKey string, contents []GeminiContent) (int, error) {
    reqBody := map[string]any{
        "model":    modelName,
        "contents": contents,
    }
    b, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models:countTokens?key=%s", apiKey)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
    if err != nil { return 0, err }
    req.Header.Set("Content-Type", "application/json")
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil { return 0, err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return 0, fmt.Errorf("countTokens error (%d): %s", resp.StatusCode, string(body))
    }
    var res struct { TotalTokens int `json:"total_tokens"` }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil { return 0, err }
    return res.TotalTokens, nil
}

 

// min returns the minimum of two integers
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// fixCommonJSONIssues attempts to fix common JSON formatting problems
func fixCommonJSONIssues(jsonStr string) string {
    fixed := strings.TrimSpace(jsonStr)

    // Strip markdown code fences
    if strings.HasPrefix(fixed, "```") {
        lines := strings.Split(fixed, "\n")
        if len(lines) > 2 {
            fixed = strings.Join(lines[1:len(lines)-1], "\n")
            fixed = strings.TrimSpace(fixed)
        }
    }

    // Keep from first '[' onward if any prelude text exists
    if idx := strings.Index(fixed, "["); idx > 0 {
        fixed = fixed[idx:]
    }

    // Fix trailing commas before closing brackets/braces
    fixed = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(fixed, "$1")

    // Fix missing commas between objects (common AI mistake)
    fixed = regexp.MustCompile(`}(\s*){`).ReplaceAllString(fixed, "},$1{")

    // Attempt to cut at last complete ']' if response is truncated
    if last := strings.LastIndex(fixed, "]"); last != -1 {
        fixed = fixed[:last+1]
    }

    // Ensure starts with '['
    if !strings.HasPrefix(fixed, "[") {
        fixed = "[" + fixed
    }
    // Balance brackets: if more '[' than ']', append missing ']'s
    open := strings.Count(fixed, "[")
    close := strings.Count(fixed, "]")
    if close < open {
        fixed = fixed + strings.Repeat("]", open-close)
    }

    return strings.TrimSpace(fixed)
}

// generateSRT generates SRT content from subtitle entries
func generateSRT(entries []SubtitleEntry) string {
    var result strings.Builder
    for _, entry := range entries {
        result.WriteString(fmt.Sprintf("%d\n", entry.Index))
        result.WriteString(fmt.Sprintf("%s --> %s\n",
            formatSRTTime(entry.StartTime),
            formatSRTTime(entry.EndTime)))
        result.WriteString(entry.Text)
        result.WriteString("\n\n")
    }
    return result.String()
}
