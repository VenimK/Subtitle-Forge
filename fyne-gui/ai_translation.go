package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Provider        string // "gemini", "openai", "deepl", "azure"
	APIKey          string
	SecondaryAPIKey string // For quota management
	Model           string // "gemini-2.5-flash", "gpt-4", etc.
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

// Global variables for AI translation
var activeTranslationJobs = make(map[string]*TranslationJob)
var aiTranslationAddFile func(string) // Function to add files from drag & drop
var cancelTranslation bool            // Flag to cancel ongoing translation

// createAITranslationTab creates the AI translation interface
func createAITranslationTab(w fyne.Window, a fyne.App) *fyne.Container {
	// Title
	title := widget.NewLabel("AI-Powered Subtitle Translation")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Provider selection
	providerSelect := widget.NewSelect([]string{
		"Google Gemini AI",
		"OpenAI GPT (Coming Soon)",
		"DeepL API (Coming Soon)",
		"Azure Translator (Coming Soon)",
	}, nil)
	providerSelect.SetSelected("Google Gemini AI")

	// Show info message when non-Gemini provider is selected
	providerSelect.OnChanged = func(selected string) {
		if selected != "Google Gemini AI" {
			dialog.ShowInformation("Provider Not Available",
				"Only Google Gemini AI is currently supported.\nOther providers are planned for future releases.", w)
			providerSelect.SetSelected("Google Gemini AI")
		}
	}

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

	// Results area
	resultArea := widget.NewMultiLineEntry()
	resultArea.SetText("Translation results will appear here...")
	resultArea.Wrapping = fyne.TextWrapWord
	resultScroll := container.NewScroll(resultArea)
	resultScroll.SetMinSize(fyne.NewSize(600, 200))

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

		if apiKeyEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Please enter your Google Gemini API key.\n\nGet your free API key at:\nhttps://aistudio.google.com/app/apikey"), w)
			return
		}

		// Validate API key format (basic check)
		if len(apiKeyEntry.Text) < 20 || !strings.HasPrefix(apiKeyEntry.Text, "AI") {
			dialog.ShowError(fmt.Errorf("Invalid API key format.\n\nGemini API keys should start with 'AI' and be at least 20 characters long."), w)
			return
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
		}

		// Start translation
		startAITranslation(inputFiles, sourceLanguageSelect.Selected, targetLanguageSelect.Selected,
			outputDir, config, progressBar, progressLabel, resultArea, translateBtn, stopBtn, w)
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
		resultArea.SetText("Translation results will appear here...")
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

	progressSection := container.NewVBox(
		widget.NewLabel("Translation Progress"),
		progressBar,
		progressLabel,
		container.NewHBox(translateBtn, stopBtn, clearBtn),
		widget.NewSeparator(),

		widget.NewLabel("Results"),
		resultScroll,
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
	resultArea *widget.Entry, translateBtn, stopBtn *widget.Button, w fyne.Window) {

	progressLabel.SetText("Starting AI translation...")
	resultArea.SetText("🤖 Initializing AI translation...\n")

	// Process files in a goroutine
	go func() {
		totalFiles := len(inputFiles)
		successCount := 0

		for i, inputFile := range inputFiles {
			// Check for cancellation
			if cancelTranslation {
				fyne.Do(func() {
					progressLabel.SetText(fmt.Sprintf("⛔ Translation cancelled by user (%d/%d files completed)", successCount, totalFiles))
					resultArea.SetText(resultArea.Text + fmt.Sprintf("\n\n⛔ Translation cancelled by user\n✅ Completed: %d files\n⏭️ Skipped: %d files", successCount, totalFiles-i))
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
				resultArea.SetText(resultArea.Text + fmt.Sprintf("\n🔄 Processing: %s", fileName))
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
			fyne.Do(func() {
				resultArea.SetText(resultArea.Text + fmt.Sprintf("\n🔄 Processing: %s", fileName))
				resultArea.SetText(resultArea.Text + fmt.Sprintf("\n   📖 Reading subtitle file..."))
			})

			// Perform translation with progress updates
			success, errorMsg := translateSubtitleFileWithProgress(inputFile, outputFile, sourceLang, targetLang, config, func(status string) {
				fyne.Do(func() {
					resultArea.SetText(resultArea.Text + fmt.Sprintf("\n   %s", status))
				})
			})

			if success {
				successCount++
				fyne.Do(func() {
					resultArea.SetText(resultArea.Text + fmt.Sprintf("\n✅ Completed: %s → %s", fileName, filepath.Base(outputFile)))
				})
			} else {
				fyne.Do(func() {
					resultArea.SetText(resultArea.Text + fmt.Sprintf("\n❌ Failed: %s\n   Error: %s", fileName, errorMsg))
				})
			}
		}

		// Final update
		fyne.Do(func() {
			progressBar.SetValue(1.0)
			progressLabel.SetText(fmt.Sprintf("Translation completed: %d/%d files successful", successCount, totalFiles))
			resultArea.SetText(resultArea.Text + fmt.Sprintf("\n\n🎉 Batch translation completed!\n✅ Success: %d files\n❌ Failed: %d files", successCount, totalFiles-successCount))
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

	// Translate in batches
	translatedEntries := make([]SubtitleEntry, len(entries))
	batchSize := config.BatchSize
	totalBatches := (len(entries) + batchSize - 1) / batchSize

	if progressCallback != nil {
		progressCallback(fmt.Sprintf("🚀 Starting translation in %d batches (batch size: %d)", totalBatches, batchSize))
	}

	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
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

		translatedBatch, err := translateBatch(batch, sourceLang, targetLang, config)
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

// translateBatch translates subtitle entries using structured JSON with schema
func translateBatch(entries []SubtitleEntry, sourceLang, targetLang string, config AITranslationConfig) ([]SubtitleEntry, error) {

	// Build input JSON with index and content fields
	type InputEntry struct {
		Index   string `json:"index"`
		Content string `json:"content"`
	}
	
	var inputEntries []InputEntry
	for i, entry := range entries {
		inputEntries = append(inputEntries, InputEntry{
			Index:   fmt.Sprintf("%d", i),
			Content: entry.Text,
		})
	}
	
	inputJSON, err := json.MarshalIndent(inputEntries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to create input JSON: %v", err)
	}

	// Get detailed instruction using the helper function
	instruction := getInstruction(targetLang, config.UseAudioContext, config.VideoFile, config.Description)
	
	// Create user prompt with input data
	userPrompt := fmt.Sprintf("%s\n\nInput data:\n%s", instruction, string(inputJSON))

	// Create API request with structured JSON response
	request := GeminiRequest{
		SystemInstruction: &GeminiContent{
			Parts: []GeminiPart{{Text: "You are a professional subtitle translator. Always follow the instructions precisely and return valid JSON."}},
			Role:  "system",
		},
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: userPrompt}},
				Role:  "user",
			},
		},
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
		return nil, err
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
			return nil, fmt.Errorf("failed to parse translation response: %v\nResponse: %s", err, translatedJSON)
		}
	}

	// Validate count matches
	if len(outputEntries) != len(entries) {
		return nil, fmt.Errorf("translation count mismatch: expected %d, got %d", len(entries), len(outputEntries))
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

	return translatedEntries, nil
}

func callGeminiAPI(request GeminiRequest, config AITranslationConfig) (string, error) {
	// Prepare request body
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Use the correct model name from your working translator
	modelName := config.Model
	if modelName == "gemini-1.5-flash" {
		modelName = "gemini-2.5-flash" // Use the working model from your translator
	}

	// Create HTTP request
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		modelName, config.APIKey)

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

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		// Provide user-friendly error messages
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
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no translation received from API")
	}

	translatedText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Validate the response is not empty
	if strings.TrimSpace(translatedText) == "" {
		return "", fmt.Errorf("API returned empty translation")
	}

	// Clean up markdown formatting if present (for text responses)
	translatedText = strings.TrimSpace(translatedText)
	if strings.HasPrefix(translatedText, "```") {
		// Remove code block wrapper
		lines := strings.Split(translatedText, "\n")
		if len(lines) > 2 {
			// Remove first and last lines (the ``` markers)
			translatedText = strings.Join(lines[1:len(lines)-1], "\n")
			translatedText = strings.TrimSpace(translatedText)
		}
	}

	return translatedText, nil
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
	fixed := jsonStr

	// Fix trailing commas before closing brackets/braces
	fixed = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(fixed, "$1")

	// Fix missing commas between objects (common AI mistake)
	fixed = regexp.MustCompile(`}(\s*){`).ReplaceAllString(fixed, "},$1{")

	// Fix unescaped quotes in content
	fixed = regexp.MustCompile(`"content":\s*"([^"]*)"([^",}\]]*)"([^",}\]]*)"([^",}\]]*)`).ReplaceAllStringFunc(fixed, func(match string) string {
		// This is a complex case - for now, just return the original
		return match
	})

	// Remove any non-JSON text at the end
	if lastBracket := strings.LastIndex(fixed, "]"); lastBracket != -1 {
		fixed = fixed[:lastBracket+1]
	}

	return fixed
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
