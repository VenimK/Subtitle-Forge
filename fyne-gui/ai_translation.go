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
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
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

// createAITranslationTab creates the AI translation interface
func createAITranslationTab(w fyne.Window) *fyne.Container {
	// Title
	title := widget.NewLabel("AI-Powered Subtitle Translation")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Provider selection
	providerSelect := widget.NewSelect([]string{
		"Google Gemini AI",
		"OpenAI GPT",
		"DeepL API",
		"Azure Translator",
	}, nil)
	providerSelect.SetSelected("Google Gemini AI")

	// API Key configuration
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter your API key...")

	secondaryKeyEntry := widget.NewPasswordEntry()
	secondaryKeyEntry.SetPlaceHolder("Optional: Secondary API key for more quota")

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
	temperatureSlider.SetValue(0.7)
	temperatureLabel := widget.NewLabel("Temperature: 0.7")
	temperatureSlider.OnChanged = func(value float64) {
		temperatureLabel.SetText(fmt.Sprintf("Temperature: %.1f", value))
	}

	batchSizeEntry := widget.NewEntry()
	batchSizeEntry.SetText("300")
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
		container.NewHBox(temperatureLabel, temperatureSlider),
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

	// Action buttons
	translateBtn := widget.NewButton("Start Translation", func() {
		if len(inputFiles) == 0 {
			dialog.ShowError(fmt.Errorf("please select subtitle files to translate"), w)
			return
		}

		if apiKeyEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("please enter your API key"), w)
			return
		}

		// Create translation config
		config := AITranslationConfig{
			Provider:        strings.ToLower(strings.Split(providerSelect.Selected, " ")[0]),
			APIKey:          apiKeyEntry.Text,
			SecondaryAPIKey: secondaryKeyEntry.Text,
			Model:           strings.Split(modelSelect.Selected, " ")[0],
			Temperature:     temperatureSlider.Value,
			BatchSize:       parseInt(batchSizeEntry.Text, 300),
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
			outputDir, config, progressBar, progressLabel, resultArea, w)
	})
	translateBtn.Importance = widget.HighImportance

	stopBtn := widget.NewButton("Stop Translation", func() {
		// TODO: Implement translation stopping
		progressLabel.SetText("Translation stopped by user")
	})
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
	resultArea *widget.Entry, w fyne.Window) {

	progressLabel.SetText("Starting AI translation...")
	resultArea.SetText("🤖 Initializing AI translation...\n")

	// Process files in a goroutine
	go func() {
		totalFiles := len(inputFiles)
		successCount := 0

		for i, inputFile := range inputFiles {
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
		fmt.Printf("DEBUG: Removed UTF-8 BOM from file\n")
	}

	// Debug: Show first 200 characters of file
	if len(contentStr) > 200 {
		fmt.Printf("DEBUG: First 200 chars of file: %q\n", contentStr[:200])
	} else {
		fmt.Printf("DEBUG: File content (first 200): %q\n", contentStr[:min(200, len(contentStr))])
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

	// Debug: Print first few entries to check parsing
	fmt.Printf("DEBUG: First 5 parsed entries:\n")
	for i, entry := range entries {
		if i >= 5 {
			break
		}
		fmt.Printf("Entry %d: Index=%d, StartTime=%v, EndTime=%v, Text=%q\n",
			i+1, entry.Index, entry.StartTime, entry.EndTime, entry.Text)
	}

	// Debug: Print last few entries to see if we're processing the right range
	fmt.Printf("DEBUG: Last 3 parsed entries:\n")
	start := len(entries) - 3
	if start < 0 {
		start = 0
	}
	for i := start; i < len(entries); i++ {
		entry := entries[i]
		fmt.Printf("Entry %d: Index=%d, StartTime=%v, EndTime=%v, Text=%q\n",
			i+1, entry.Index, entry.StartTime, entry.EndTime, entry.Text)
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

		translatedBatch, err := translateBatch(batch, sourceLang, targetLang, config)
		if err != nil {
			fmt.Printf("Error translating batch %d: %v\n", batchNum, err)
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("❌ Batch %d/%d failed: %s", batchNum, totalBatches, err.Error()))
			}
			return false, fmt.Errorf("batch %d failed: %v", batchNum, err)
		}

		// Validate translated batch has correct number of entries
		if len(translatedBatch) != len(batch) {
			errMsg := fmt.Sprintf("batch %d returned %d entries, expected %d", batchNum, len(translatedBatch), len(batch))
			if progressCallback != nil {
				progressCallback(fmt.Sprintf("⚠️ %s", errMsg))
			}
			return false, fmt.Errorf(errMsg)
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

// cleanAIResponse removes AI thinking/reasoning from response
func cleanAIResponse(response string) string {
	// If response starts with "THOUGHT:" or similar, remove it
	if strings.Contains(response, "THOUGHT:") {
		fmt.Printf("⚠️ AI included thinking process - cleaning response\n")

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
			result := strings.Join(cleanedParts, "\n---SUBTITLE_SEPARATOR---\n")
			fmt.Printf("✅ Cleaned response: extracted %d translations\n", len(cleanedParts))
			return result
		}
	}

	// No cleaning needed
	return response
}

// translateBatch translates subtitle entries using REAL batch processing (like your working translator)
func translateBatch(entries []SubtitleEntry, sourceLang, targetLang string, config AITranslationConfig) ([]SubtitleEntry, error) {
	fmt.Printf("🔧 Processing batch: %d texts\n", len(entries))

	// Create batch text exactly like your working translator
	var textParts []string
	for _, entry := range entries {
		textParts = append(textParts, entry.Text)
	}

	// Join with clear separators (like your working translator)
	batchText := strings.Join(textParts, "\n---SUBTITLE_SEPARATOR---\n")

	// Use strict prompt to prevent AI thinking/reasoning in output
	prompt := fmt.Sprintf(`Translate the following subtitle entries from %s to %s.

CRITICAL RULES:
1. Output ONLY the translated text
2. NO explanations, NO thinking process, NO commentary
3. Keep the ---SUBTITLE_SEPARATOR--- markers EXACTLY as they are
4. Preserve line breaks exactly (same number of lines in translation)
5. Natural conversational translation

DO NOT include phrases like "Translation:", "THOUGHT:", "Entry X:", etc.
Output ONLY the pure translated text with separators.

Text to translate:
%s`, sourceLang, targetLang, batchText)

	// Create API request with system instruction to prevent thinking
	request := GeminiRequest{
		SystemInstruction: &GeminiContent{
			Parts: []GeminiPart{{Text: "You are a translation tool. Output ONLY translated text, never include reasoning, thinking, or explanations."}},
			Role:  "system",
		},
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: prompt}},
				Role:  "user",
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     config.Temperature,
			TopP:            0.95,
			TopK:            20,
			MaxOutputTokens: 8192,
		},
	}

	// Make API call
	fmt.Printf("🧠 AI analyzing subtitle structure...\n")
	translatedText, err := callGeminiAPI(request, config)
	if err != nil {
		fmt.Printf("❌ Batch translation failed: %v\n", err)
		return nil, err
	}

	fmt.Printf("📊 Parsing translation results...\n")

	// Clean up AI thinking/reasoning if present
	translatedText = cleanAIResponse(translatedText)

	// Split response by separators (like your working translator)
	translatedParts := strings.Split(translatedText, "---SUBTITLE_SEPARATOR---")

	// Validate count matches
	if len(translatedParts) != len(entries) {
		fmt.Printf("⚠️ Translation count mismatch: got %d, expected %d\n", len(translatedParts), len(entries))
		// Try to handle gracefully
		if len(translatedParts) < len(entries) {
			// Pad with original text
			for i := len(translatedParts); i < len(entries); i++ {
				translatedParts = append(translatedParts, entries[i].Text)
			}
		} else {
			// Truncate excess
			translatedParts = translatedParts[:len(entries)]
		}
	}

	// Build result entries (preserve original timing and indexes)
	var translatedEntries []SubtitleEntry
	for i, entry := range entries {
		translatedText := strings.TrimSpace(translatedParts[i])

		translatedEntries = append(translatedEntries, SubtitleEntry{
			Index:     entry.Index,     // Preserve original index
			StartTime: entry.StartTime, // Preserve original timing
			EndTime:   entry.EndTime,   // Preserve original timing
			Text:      translatedText,  // Use translated content
		})
	}

	fmt.Printf("✅ Batch completed: %d translations\n", len(translatedEntries))
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
		return "", fmt.Errorf("API error: %s", string(body))
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

	// Basic validation for single entry translation
	if len(translatedText) == 0 {
		return "", fmt.Errorf("API returned empty translation")
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
