package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var whisperTranscribeSetInputFile func(string)

type progressReader struct {
	r        io.Reader
	total    int64
	onUpdate func(sent int64)
	sent     int64
	last     time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.sent += int64(n)
		now := time.Now()
		if p.onUpdate != nil && (p.last.IsZero() || now.Sub(p.last) > 500*time.Millisecond) {
			p.last = now
			p.onUpdate(p.sent)
		}
	}
	return n, err
}

func createWhisperTranscribeTab(w fyne.Window, a fyne.App) *fyne.Container {
	title := widget.NewLabel(T("whisper.title"))
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel(T("whisper.ready"))
	statusLabel.Wrapping = fyne.TextWrapWord
	busyBar := widget.NewProgressBarInfinite()
	busyBar.Hide()

	var inputFile string
	inputLabel := widget.NewLabel(T("whisper.no_file"))
	inputLabel.Wrapping = fyne.TextWrapWord

	whisperTranscribeSetInputFile = func(path string) {
		inputFile = path
		inputLabel.SetText(T("whisper.selected") + filepath.Base(inputFile))
	}

	selectInputBtn := widget.NewButton(T("whisper.select_file"), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			whisperTranscribeSetInputFile(reader.URI().Path())
		}, w)
	})
	selectInputBtn.Importance = widget.HighImportance

	outputDir := ""
	outputLabel := widget.NewLabel(T("whisper.output_same_dir"))
	outputLabel.Wrapping = fyne.TextWrapWord

	selectOutputBtn := widget.NewButton(T("whisper.select_output"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText(T("whisper.output_prefix") + outputDir)
		}, w)
	})

	modeSelect := widget.NewSelect([]string{"Local (whisper-cli)", "Remote API"}, nil)
	modeSelect.PlaceHolder = "Local (whisper-cli)"
	modeSelect.SetSelected(a.Preferences().StringWithFallback("whisper_transcribe_mode", "Local (whisper-cli)"))
	modeSelect.OnChanged = func(s string) {
		if s == "" {
			return
		}
		a.Preferences().SetString("whisper_transcribe_mode", s)
	}

	remoteURLEntry := widget.NewEntry()
	remoteURLEntry.SetPlaceHolder("http://192.168.1.54:8000")
	remoteURLEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_url", "http://127.0.0.1:8000"))
	remoteURLEntry.OnChanged = func(s string) { a.Preferences().SetString("whisper_remote_url", s) }

	appendLog := func(msg string) {}

	remoteKeyEntry := widget.NewPasswordEntry()
	remoteKeyEntry.SetPlaceHolder("Optional x-api-key")
	remoteKeyEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_api_key", ""))

	remoteDebugLog := widget.NewCheck("Show server response (debug)", nil)
	remoteDebugLog.Checked = a.Preferences().BoolWithFallback("whisper_remote_debug", false)
	remoteDebugLog.OnChanged = func(checked bool) {
		a.Preferences().SetBool("whisper_remote_debug", checked)
	}

	remoteStreamLogs := widget.NewCheck("Stream server logs (SSE)", nil)
	remoteStreamLogs.Checked = a.Preferences().BoolWithFallback("whisper_remote_stream", false)
	remoteStreamLogs.OnChanged = func(checked bool) {
		a.Preferences().SetBool("whisper_remote_stream", checked)
	}

	var applyModeUI func()

	remoteTerminalLogs := widget.NewCheck("Stream logs in Terminal (macOS)", nil)
	remoteTerminalLogs.Checked = a.Preferences().BoolWithFallback("whisper_remote_terminal_logs", false)
	remoteTerminalLogs.OnChanged = func(checked bool) {
		a.Preferences().SetBool("whisper_remote_terminal_logs", checked)
		applyModeUI()
	}

	remoteModelEntry := widget.NewEntry()
	remoteModelEntry.SetPlaceHolder("model name (e.g. base.en, small, ggml-base.en.bin)")
	remoteModelEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_model", ""))
	remoteModelEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_remote_model", s)
	}

	remoteModelSelect := widget.NewSelect(nil, func(s string) {
		if s == "" {
			return
		}
		remoteModelEntry.SetText(s)
	})
	remoteModelSelect.PlaceHolder = "Fetch remote models"

	refreshRemoteModels := widget.NewButton("Refresh Remote Models", func() {
		baseURL := strings.TrimSpace(remoteURLEntry.Text)
		if baseURL == "" {
			appendLog("Remote URL is empty; cannot fetch models")
			return
		}
		lowerURL := strings.ToLower(baseURL)
		if idx := strings.Index(lowerURL, "/transcribe"); idx != -1 {
			baseURL = baseURL[:idx]
		}
		modelURL := strings.TrimRight(baseURL, "/") + "/models"
		req, err := http.NewRequest("GET", modelURL, nil)
		if err != nil {
			appendLog("Failed to build model request: " + err.Error())
			return
		}
		key := strings.TrimSpace(remoteKeyEntry.Text)
		if key != "" {
			req.Header.Set("x-api-key", key)
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			appendLog("Failed to fetch models: " + err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			appendLog(fmt.Sprintf("Model fetch failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body))))
			return
		}
		var payload struct {
			Models  []string `json:"models"`
			Default string   `json:"default"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			appendLog("Failed to parse models: " + err.Error())
			return
		}
		if len(payload.Models) == 0 {
			appendLog("Remote models list is empty")
			return
		}
		remoteModelSelect.Options = payload.Models
		remoteModelSelect.Refresh()
		appendLog(fmt.Sprintf("Loaded %d remote models", len(payload.Models)))
		if remoteModelEntry.Text == "" && payload.Default != "" {
			remoteModelEntry.SetText(payload.Default)
		}
	})

	remoteThreadsEntry := widget.NewEntry()
	remoteThreadsEntry.SetPlaceHolder("threads (e.g. 6)")
	remoteThreadsEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_threads", ""))
	remoteThreadsEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_remote_threads", s)
	}

	remoteBeamEntry := widget.NewEntry()
	remoteBeamEntry.SetPlaceHolder("beam size (e.g. 1)")
	remoteBeamEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_beam", ""))
	remoteBeamEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_remote_beam", s)
	}

	remoteBestOfEntry := widget.NewEntry()
	remoteBestOfEntry.SetPlaceHolder("best-of (e.g. 1)")
	remoteBestOfEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_bestof", ""))
	remoteBestOfEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_remote_bestof", s)
	}

	remoteLangEntry := widget.NewEntry()
	remoteLangEntry.SetPlaceHolder("language code (e.g. en)")
	remoteLangEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_language", ""))
	remoteLangEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_remote_language", s)
	}

	rememberRemoteKey := widget.NewCheck("Remember API key (saved locally)", nil)
	rememberRemoteKey.Checked = a.Preferences().BoolWithFallback("whisper_remote_remember_api_key", false)
	rememberRemoteKey.OnChanged = func(checked bool) {
		a.Preferences().SetBool("whisper_remote_remember_api_key", checked)
		if !checked {
			a.Preferences().SetString("whisper_remote_api_key", "")
			relaxed := strings.TrimSpace(remoteKeyEntry.Text)
			if relaxed != "" {
				remoteKeyEntry.SetText("")
			}
		}
	}
	remoteKeyEntry.OnChanged = func(s string) {
		if rememberRemoteKey.Checked {
			a.Preferences().SetString("whisper_remote_api_key", s)
		}
	}

	defaultWhisperBin := a.Preferences().StringWithFallback("whisper_cli_path", "")
	if defaultWhisperBin == "" {
		detectedWhisperBin := whisperDefaultCLIPath()
		if isExecutableFile(detectedWhisperBin) {
			defaultWhisperBin = detectedWhisperBin
			a.Preferences().SetString("whisper_cli_path", detectedWhisperBin)
		} else {
			defaultWhisperBin = detectedWhisperBin
		}
	}

	whisperBinEntry := widget.NewEntry()
	whisperBinEntry.SetPlaceHolder("Path to whisper-cli")
	whisperBinEntry.SetText(defaultWhisperBin)
	whisperBinEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_cli_path", s)
	}

	defaultModel := a.Preferences().StringWithFallback("whisper_model_path", "")
	if defaultModel == "" {
		detectedModel := whisperDefaultModelPath()
		if isReadableFile(detectedModel) {
			defaultModel = detectedModel
			a.Preferences().SetString("whisper_model_path", detectedModel)
		} else {
			defaultModel = detectedModel
		}
	}

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("Path to GGML model (e.g. ggml-base.en.bin)")
	modelEntry.SetText(defaultModel)
	modelEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_model_path", s)
	}

	langLabel := widget.NewLabel("Language: Auto")
	langLabel.Wrapping = fyne.TextWrapWord
	langOptions := []string{
		"Auto",
		"en",
		"nl",
		"de",
		"fr",
		"es",
		"it",
		"pt",
		"sv",
		"da",
		"no",
		"fi",
		"pl",
		"cs",
		"tr",
		"ru",
		"uk",
		"ar",
		"hi",
		"ja",
		"ko",
		"zh",
	}
	langSelect := widget.NewSelect(langOptions, nil)
	langSelect.PlaceHolder = "Auto"
	langSelect.SetSelected(a.Preferences().StringWithFallback("whisper_language", "Auto"))
	langSelect.OnChanged = func(s string) {
		if s == "" {
			return
		}
		a.Preferences().SetString("whisper_language", s)
		if s == "Auto" {
			langLabel.SetText("Language: Auto")
			return
		}
		langLabel.SetText("Language: " + s)
	}

	modelOptions := []string{}
	modelSelect := widget.NewSelect(modelOptions, nil)
	modelSelect.PlaceHolder = "Select detected model"
	refreshModelOptions := func() {
		modelOptions = modelOptions[:0]
		modelDir := filepath.Dir(strings.TrimSpace(modelEntry.Text))
		if modelDir == "" {
			modelSelect.Options = modelOptions
			modelSelect.Refresh()
			return
		}
		entries, err := os.ReadDir(modelDir)
		if err != nil {
			modelSelect.Options = modelOptions
			modelSelect.Refresh()
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			if strings.HasPrefix(lower, "ggml-") && strings.HasSuffix(lower, ".bin") {
				modelOptions = append(modelOptions, name)
			}
		}
		sort.Strings(modelOptions)
		modelSelect.Options = modelOptions
		modelSelect.Refresh()
		// Try to keep selection consistent with the current entry.
		curBase := filepath.Base(strings.TrimSpace(modelEntry.Text))
		for _, opt := range modelOptions {
			if opt == curBase {
				modelSelect.SetSelected(opt)
				return
			}
		}
		modelSelect.ClearSelected()
	}
	modelSelect.OnChanged = func(s string) {
		if s == "" {
			return
		}
		modelDir := filepath.Dir(strings.TrimSpace(modelEntry.Text))
		if modelDir == "" {
			return
		}
		modelEntry.SetText(filepath.Join(modelDir, s))
	}

	refreshModelsBtn := widget.NewButton("Refresh Models", func() {
		refreshModelOptions()
	})

	modelNameOptions := []string{
		"tiny.en",
		"tiny",
		"base.en",
		"base",
		"small.en",
		"small",
		"medium.en",
		"medium",
		"large-v2",
		"large-v3",
	}
	modelNameSelect := widget.NewSelect(modelNameOptions, nil)
	modelNameSelect.PlaceHolder = "Model to download/build (e.g. base.en)"
	modelNameSelect.SetSelected(a.Preferences().StringWithFallback("whisper_model_name", "base.en"))
	modelNameSelect.OnChanged = func(s string) {
		if s == "" {
			return
		}
		a.Preferences().SetString("whisper_model_name", s)
	}

	coremlCondaEnvEntry := widget.NewEntry()
	coremlCondaEnvEntry.SetPlaceHolder("Conda env name (e.g. py311-whisper)")
	coremlCondaEnvEntry.SetText(a.Preferences().StringWithFallback("whisper_coreml_env", "py311-whisper"))
	coremlCondaEnvEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_coreml_env", s)
	}

	coremlVenvActivateEntry := widget.NewEntry()
	coremlVenvActivateEntry.SetPlaceHolder("Venv activate path (e.g. ~/py311-whisper/bin/activate)")
	coremlVenvActivateEntry.SetText(a.Preferences().StringWithFallback("whisper_coreml_venv_activate", ""))
	coremlVenvActivateEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_coreml_venv_activate", s)
	}

	installModelBtn := widget.NewButton("Download Model (Terminal)", func() {
		if runtime.GOOS != "darwin" {
			dialog.ShowError(fmt.Errorf("model download helper is currently only implemented for macOS"), w)
			return
		}
		root, err := findWhisperCppRoot(whisperBinEntry.Text, modelEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		modelName := strings.TrimSpace(modelNameSelect.Selected)
		if modelName == "" {
			dialog.ShowError(fmt.Errorf("please select a model name"), w)
			return
		}
		scriptPath := filepath.Join(root, "models", "download-ggml-model.sh")
		cmdFile, err := createTerminalCommandFile(root, fmt.Sprintf("Download model: %s", modelName), []string{"bash", scriptPath, modelName})
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if err := exec.Command("open", cmdFile).Start(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open Terminal: %v", err), w)
			return
		}
	})

	generateCoreMLBtn := widget.NewButton("Generate CoreML Encoder (Terminal)", func() {
		if runtime.GOOS != "darwin" {
			dialog.ShowError(fmt.Errorf("CoreML model generation helper is currently only implemented for macOS"), w)
			return
		}
		root, err := findWhisperCppRoot(whisperBinEntry.Text, modelEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		modelName := strings.TrimSpace(modelNameSelect.Selected)
		if modelName == "" {
			dialog.ShowError(fmt.Errorf("please select a model name"), w)
			return
		}
		scriptPath := filepath.Join(root, "models", "generate-coreml-model.sh")
		cmd := []string{"bash", scriptPath, modelName}
		venvActivate := strings.TrimSpace(coremlVenvActivateEntry.Text)
		if venvActivate == "" {
			homeDir, _ := os.UserHomeDir()
			fallback := filepath.Join(homeDir, "py311-whisper", "bin", "activate")
			if _, err := os.Stat(fallback); err == nil {
				venvActivate = fallback
			}
		}
		if venvActivate != "" {
			homeDir, _ := os.UserHomeDir()
			if strings.HasPrefix(venvActivate, "~/") {
				venvActivate = filepath.Join(homeDir, strings.TrimPrefix(venvActivate, "~/"))
			}
			line := "source " + shellQuote(venvActivate) + " && bash " + shellQuote(scriptPath) + " " + shellQuote(modelName)
			cmd = []string{"bash", "-lc", line}
		} else if _, err := exec.LookPath("conda"); err == nil {
			envName := strings.TrimSpace(coremlCondaEnvEntry.Text)
			if envName == "" {
				envName = "py311-whisper"
			}
			cmd = []string{"conda", "run", "-n", envName, "bash", scriptPath, modelName}
		}
		cmdFile, err := createTerminalCommandFile(root, fmt.Sprintf("Generate CoreML encoder: %s", modelName), cmd)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if err := exec.Command("open", cmdFile).Start(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open Terminal: %v", err), w)
			return
		}
	})

	browseWhisperBtn := widget.NewButton("Browse", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			whisperBinEntry.SetText(reader.URI().Path())
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{"", ".exe"}))
		fd.Show()
	})

	browseModelBtn := widget.NewButton("Browse", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			modelEntry.SetText(reader.URI().Path())
		}, w)
		fd.Show()
	})

	// Threads slider
	defaultThreads := a.Preferences().IntWithFallback("whisper_threads", 4)
	threadsLabel := widget.NewLabel(fmt.Sprintf("Threads: %d", defaultThreads))
	threadsSlider := widget.NewSlider(1, 16)
	threadsSlider.Step = 1
	threadsSlider.Value = float64(defaultThreads)
	threadsSlider.OnChanged = func(f float64) {
		threadsLabel.SetText(fmt.Sprintf("Threads: %d", int(f)))
		a.Preferences().SetInt("whisper_threads", int(f))
	}

	wantSRT := widget.NewCheck("Generate SRT", nil)
	wantSRT.SetChecked(true)
	wantVTT := widget.NewCheck("Generate VTT", nil)
	wantTXT := widget.NewCheck("Generate TXT", nil)
	wantSRTTimestampsOnly := widget.NewCheck("Timestamps-only SRT (extra file)", nil)
	wantSRTTimestampsOnly.SetChecked(a.Preferences().BoolWithFallback("whisper_srt_timestamps_only", false))
	wantSRTTimestampsOnly.OnChanged = func(b bool) {
		a.Preferences().SetBool("whisper_srt_timestamps_only", b)
	}
	wantSRT.OnChanged = func(b bool) {
		if b {
			wantSRTTimestampsOnly.Enable()
			return
		}
		wantSRTTimestampsOnly.Disable()
		wantSRTTimestampsOnly.SetChecked(false)
	}

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord
	resultScroll := container.NewScroll(resultLabel)
	resultScroll.SetMinSize(fyne.NewSize(0, 200))

	whisperResultActions := container.NewHBox()
	whisperResultActions.Hide()

	showWhisperResultActions := func(path string) {
		if strings.TrimSpace(path) == "" {
			whisperResultActions.Hide()
			whisperResultActions.Objects = nil
			whisperResultActions.Refresh()
			return
		}
		whisperResultActions.Objects = []fyne.CanvasObject{
			widget.NewButton(T("common.open_output_folder"), func() {
				openFolderPath(w, path)
			}),
			widget.NewButton(T("common.copy_output_path"), func() {
				copyTextToClipboard(w, path)
			}),
		}
		whisperResultActions.Show()
		whisperResultActions.Refresh()
	}

	logEntry := widget.NewMultiLineEntry()
	logEntry.Wrapping = fyne.TextWrapWord
	logScroll := container.NewScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(0, 520))
	var logText string
	var logFile *os.File
	var logMu sync.Mutex
	appendLog = func(msg string) {
		ts := time.Now().Format("15:04:05")
		fyne.Do(func() {
			logText += fmt.Sprintf("[%s] %s\n", ts, msg)
			logEntry.SetText(logText)
			logScroll.ScrollToBottom()
		})
		logMu.Lock()
		if logFile != nil {
			_, _ = fmt.Fprintf(logFile, "[%s] %s\n", ts, msg)
			_ = logFile.Sync()
		}
		logMu.Unlock()
	}

	applyModeUI = func() {
		isRemote := modeSelect.Selected == "Remote API"
		showLog := !(runtime.GOOS == "darwin" && remoteTerminalLogs.Checked)
		if isRemote {
			whisperBinEntry.Disable()
			browseWhisperBtn.Disable()
			modelEntry.Disable()
			browseModelBtn.Disable()
			modelSelect.Disable()
			refreshModelsBtn.Disable()
			modelNameSelect.Disable()
			installModelBtn.Disable()
			generateCoreMLBtn.Disable()
			coremlCondaEnvEntry.Disable()
			coremlVenvActivateEntry.Disable()
			threadsSlider.Disable()
			wantSRT.Enable()
			wantVTT.Enable()
			wantTXT.Enable()
			langSelect.Disable()
		} else {
			whisperBinEntry.Enable()
			browseWhisperBtn.Enable()
			modelEntry.Enable()
			browseModelBtn.Enable()
			modelSelect.Enable()
			refreshModelsBtn.Enable()
			modelNameSelect.Enable()
			installModelBtn.Enable()
			generateCoreMLBtn.Enable()
			coremlCondaEnvEntry.Enable()
			coremlVenvActivateEntry.Enable()
			threadsSlider.Enable()
			langSelect.Enable()
		}
		if showLog {
			logScroll.Show()
		} else {
			logScroll.Hide()
		}
	}
	modeSelect.OnChanged = func(s string) {
		if s == "" {
			return
		}
		a.Preferences().SetString("whisper_transcribe_mode", s)
		applyModeUI()
	}

	var runBtn *widget.Button
	runBtn = widget.NewButton("Start Transcription", func() {
		if inputFile == "" {
			dialog.ShowError(fmt.Errorf("please select an input file"), w)
			return
		}

		if modeSelect.Selected != "Remote API" {
			if strings.TrimSpace(whisperBinEntry.Text) == "" {
				dialog.ShowError(fmt.Errorf("please set whisper-cli path"), w)
				return
			}
			if !isExecutableFile(strings.TrimSpace(whisperBinEntry.Text)) {
				dialog.ShowError(fmt.Errorf(T("whisper.err_whisper_cli_missing"), strings.TrimSpace(whisperBinEntry.Text)), w)
				return
			}
			if strings.TrimSpace(modelEntry.Text) == "" {
				dialog.ShowError(fmt.Errorf("please set a GGML model path"), w)
				return
			}
			if !isReadableFile(strings.TrimSpace(modelEntry.Text)) {
				dialog.ShowError(fmt.Errorf(T("whisper.err_model_missing"), strings.TrimSpace(modelEntry.Text)), w)
				return
			}
		}
		if !wantSRT.Checked && !wantVTT.Checked && !wantTXT.Checked {
			dialog.ShowError(fmt.Errorf("please select at least one output format (SRT/VTT/TXT)"), w)
			return
		}

		outDir := outputDir
		if outDir == "" {
			outDir = filepath.Dir(inputFile)
		}

		startTime := time.Now()
		fyne.Do(func() {
			statusLabel.SetText("Starting…")
			busyBar.Show()
			showWhisperResultActions("")
		})
		runBtn.Disable()

		outputPrefix := filepath.Join(outDir, strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))
		var outputs []string
		if wantSRT.Checked {
			outputs = append(outputs, outputPrefix+".srt")
			if wantSRTTimestampsOnly.Checked {
				outputs = append(outputs, outputPrefix+".timestamps.srt")
			}
		}
		if wantVTT.Checked {
			outputs = append(outputs, outputPrefix+".vtt")
		}
		if wantTXT.Checked {
			outputs = append(outputs, outputPrefix+".txt")
		}

		threads := int(threadsSlider.Value)
		lang := strings.TrimSpace(langSelect.Selected)
		if lang == "Auto" {
			lang = ""
		}

		if modeSelect.Selected == "Remote API" {
			baseURL := strings.TrimSpace(remoteURLEntry.Text)
			if baseURL == "" {
				dialog.ShowError(fmt.Errorf("please set Remote API URL"), w)
				busyBar.Hide()
				runBtn.Enable()
				statusLabel.SetText("❌ Missing remote URL")
				return
			}
			apiKey := strings.TrimSpace(remoteKeyEntry.Text)
			if apiKey == "" {
				appendLog("⚠️ No Remote API key set (x-api-key). If your server requires it, the request will fail.")
				fyne.Do(func() {
					statusLabel.SetText("⚠️ Remote mode: no API key")
				})
			}

			appendLog("Remote mode selected")
			appendLog("Input: " + inputFile)
			appendLog("Remote URL: " + baseURL)

			if runtime.GOOS == "darwin" && remoteTerminalLogs.Checked {
				var logPath string
				logMu.Lock()
				if logFile == nil {
					if tmp, err := os.CreateTemp("", "subtitle-forge-remote-log-*.log"); err == nil {
						logFile = tmp
						logPath = tmp.Name()
					}
				}
				logMu.Unlock()
				if logPath != "" {
					appendLog("Remote log file: " + logPath)
					script := "clear; echo 'Subtitle Forge - Remote Logs'; printf 'Log file: %s\\n' " + shellQuote(logPath) + "; echo ''; " +
						"tail -f " + shellQuote(logPath) + " | awk '{ if ($0 ~ /__SF_DONE__/) exit; print }'; " +
						"echo ''; echo '✅ Done.'; echo 'Closing in 3 seconds…'; sleep 3; " +
						"osascript -e 'tell application \"Terminal\" to close front window' &>/dev/null &"
					cmdFile, err := createTerminalCommandFile(filepath.Dir(logPath), "Remote transcription logs", []string{"bash", "-lc", script})
					if err == nil {
						_ = exec.Command("open", cmdFile).Start()
					}
				}
			}

			go func() {
				defer fyne.Do(func() {
					busyBar.Hide()
					runBtn.Enable()
				})
				defer func() {
					logMu.Lock()
					if logFile != nil {
						_, _ = fmt.Fprintln(logFile, "__SF_DONE__")
						_ = logFile.Sync()
						_ = logFile.Close()
						logFile = nil
					}
					logMu.Unlock()
				}()

				callOne := func(format, outFile string) error {
					trimmed := strings.TrimSpace(baseURL)
					if trimmed == "" {
						return fmt.Errorf("remote URL is empty")
					}
					useStream := remoteStreamLogs.Checked || (runtime.GOOS == "darwin" && remoteTerminalLogs.Checked)
					endpoint := strings.TrimRight(trimmed, "/")
					lower := strings.ToLower(endpoint)
					// Allow user to paste either:
					//  - http://host:8000
					//  - http://host:8000/transcribe
					//  - http://host:8000/transcribe/srt
					if strings.Contains(lower, "/transcribe/") {
						// If they already provided /transcribe/{something}, replace the tail with the requested format.
						idx := strings.LastIndex(lower, "/transcribe/")
						endpoint = endpoint[:idx+len("/transcribe/")] + format
					} else if strings.HasSuffix(lower, "/transcribe") {
						endpoint = endpoint + "/" + format
					} else {
						endpoint = endpoint + "/transcribe/" + format
					}
					if useStream {
						endpoint = strings.Replace(endpoint, "/transcribe/", "/transcribe/stream/", 1)
					}
					params := []string{}
					if remoteDebugLog.Checked {
						params = append(params, "debug=1")
					}
					if model := strings.TrimSpace(remoteModelEntry.Text); model != "" {
						params = append(params, "model="+url.QueryEscape(model))
					}
					if threads := strings.TrimSpace(remoteThreadsEntry.Text); threads != "" {
						params = append(params, "threads="+url.QueryEscape(threads))
					}
					if beam := strings.TrimSpace(remoteBeamEntry.Text); beam != "" {
						params = append(params, "beam_size="+url.QueryEscape(beam))
					}
					if bestOf := strings.TrimSpace(remoteBestOfEntry.Text); bestOf != "" {
						params = append(params, "best_of="+url.QueryEscape(bestOf))
					}
					if lang := strings.TrimSpace(remoteLangEntry.Text); lang != "" {
						params = append(params, "language="+url.QueryEscape(lang))
					}
					if len(params) > 0 {
						sep := "?"
						if strings.Contains(endpoint, "?") {
							sep = "&"
						}
						endpoint = endpoint + sep + strings.Join(params, "&")
					}
					f, err := os.Open(inputFile)
					if err != nil {
						return err
					}
					defer f.Close()

					var fileSize int64
					if st, err := f.Stat(); err == nil {
						fileSize = st.Size()
					}

					appendLog("POST " + endpoint)
					appendLog(fmt.Sprintf("Uploading file (%.2f MB)…", float64(fileSize)/1024.0/1024.0))
					fyne.Do(func() {
						statusLabel.SetText("Uploading…")
					})

					pr, pw := io.Pipe()
					writer := multipart.NewWriter(pw)
					contentType := writer.FormDataContentType()

					go func() {
						defer pw.Close()
						part, err := writer.CreateFormFile("file", filepath.Base(inputFile))
						if err != nil {
							_ = writer.Close()
							_ = pw.CloseWithError(err)
							return
						}
						p := &progressReader{
							r:     f,
							total: fileSize,
							onUpdate: func(sent int64) {
								if fileSize > 0 {
									pct := (float64(sent) / float64(fileSize)) * 100
									pct = math.Max(0, math.Min(100, pct))
									appendLog(fmt.Sprintf("Upload progress: %.1f%% (%.2f/%.2f MB)", pct, float64(sent)/1024.0/1024.0, float64(fileSize)/1024.0/1024.0))
								} else {
									appendLog(fmt.Sprintf("Upload progress: %.2f MB", float64(sent)/1024.0/1024.0))
								}
							},
						}
						if _, err := io.Copy(part, p); err != nil {
							_ = writer.Close()
							_ = pw.CloseWithError(err)
							return
						}
						_ = writer.Close()
					}()

					req, err := http.NewRequest("POST", endpoint, pr)
					if err != nil {
						return err
					}
					req.Header.Set("Content-Type", contentType)
					if apiKey != "" {
						req.Header.Set("x-api-key", apiKey)
					}

					transport := http.DefaultTransport.(*http.Transport).Clone()
					transport.ForceAttemptHTTP2 = false
					client := &http.Client{Timeout: 2 * time.Hour, Transport: transport}
					resp, err := client.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					fyne.Do(func() {
						statusLabel.SetText("Transcribing…")
					})
					appendLog("Waiting for server response…")

					if useStream {
						if resp.StatusCode < 200 || resp.StatusCode >= 300 {
							body, _ := io.ReadAll(resp.Body)
							return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
						}
						scanner := bufio.NewScanner(resp.Body)
						scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
						var resultContent string
					sseLoop:
						for scanner.Scan() {
							line := strings.TrimSpace(scanner.Text())
							if !strings.HasPrefix(line, "data: ") {
								continue
							}
							payload := strings.TrimPrefix(line, "data: ")
							var msg map[string]any
							if err := json.Unmarshal([]byte(payload), &msg); err != nil {
								continue
							}
							typeVal, _ := msg["type"].(string)
							switch typeVal {
							case "log":
								if m, ok := msg["message"].(string); ok {
									appendLog(m)
								}
							case "error":
								if m, ok := msg["message"].(string); ok {
									return fmt.Errorf("server error: %s", m)
								}
								return fmt.Errorf("server error")
							case "result":
								if c, ok := msg["content"].(string); ok {
									resultContent = c
									break sseLoop
								}
							}
						}
						if err := scanner.Err(); err != nil {
							return err
						}
						if resultContent == "" {
							return fmt.Errorf("no result content received from stream")
						}
						appendLog("Saving output: " + outFile)
						fyne.Do(func() {
							statusLabel.SetText("Saving output…")
						})
						if err := os.WriteFile(outFile, []byte(resultContent), 0644); err != nil {
							return err
						}
						if format == "srt" && wantSRTTimestampsOnly.Checked {
							timestampsOut := strings.TrimSuffix(outFile, ".srt") + ".timestamps.srt"
							appendLog("Generating timestamps-only SRT: " + timestampsOut)
							if err := writeTimestampsOnlySRT(outFile, timestampsOut); err != nil {
								return err
							}
						}
						return nil
					}

					respBody, err := io.ReadAll(resp.Body)
					if err != nil {
						return err
					}
					if remoteDebugLog.Checked {
						preview := strings.TrimSpace(string(respBody))
						if preview == "" {
							appendLog("Response body (debug): <empty>")
						} else if len(preview) > 4000 {
							appendLog("Response body (debug, truncated): " + preview[:4000])
						} else {
							appendLog("Response body (debug): " + preview)
						}
					}
					if resp.StatusCode < 200 || resp.StatusCode >= 300 {
						return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
					}
					appendLog(fmt.Sprintf("Response: HTTP %d (%d bytes)", resp.StatusCode, len(respBody)))

					payload := strings.TrimSpace(string(respBody))
					// Some servers return raw SRT/VTT/text; others return JSON like {"srt": "..."}.
					if strings.HasPrefix(payload, "{") {
						var m map[string]any
						if err := json.Unmarshal(respBody, &m); err == nil {
							if v, ok := m["_log"]; ok {
								if s, ok := v.(string); ok {
									trimmed := strings.TrimSpace(s)
									if trimmed != "" {
										appendLog("Server log:\n" + trimmed)
									}
								}
							}
							var key string
							switch format {
							case "srt":
								key = "srt"
							case "vtt":
								key = "vtt"
							case "txt":
								key = "text"
							}
							if key != "" {
								if v, ok := m[key]; ok {
									if s, ok := v.(string); ok {
										appendLog("Parsed JSON response key: " + key)
										respBody = []byte(s)
									}
								}
							}
						}
					}

					appendLog("Saving output: " + outFile)
					fyne.Do(func() {
						statusLabel.SetText("Saving output…")
					})
					if err := os.WriteFile(outFile, respBody, 0644); err != nil {
						return err
					}
					if format == "srt" && wantSRTTimestampsOnly.Checked {
						timestampsOut := strings.TrimSuffix(outFile, ".srt") + ".timestamps.srt"
						appendLog("Generating timestamps-only SRT: " + timestampsOut)
						if err := writeTimestampsOnlySRT(outFile, timestampsOut); err != nil {
							return err
						}
					}
					return nil
				}

				fyne.Do(func() {
					statusLabel.SetText("Uploading to remote server…")
				})

				var done []string
				var errs []string

				if wantSRT.Checked {
					outFile := outputPrefix + ".srt"
					if err := callOne("srt", outFile); err != nil {
						errs = append(errs, "SRT: "+err.Error())
					} else {
						done = append(done, outFile)
						if wantSRTTimestampsOnly.Checked {
							done = append(done, strings.TrimSuffix(outFile, ".srt")+".timestamps.srt")
						}
					}
				}
				if wantVTT.Checked {
					outFile := outputPrefix + ".vtt"
					if err := callOne("vtt", outFile); err != nil {
						errs = append(errs, "VTT: "+err.Error())
					} else {
						done = append(done, outFile)
					}
				}
				if wantTXT.Checked {
					outFile := outputPrefix + ".txt"
					if err := callOne("txt", outFile); err != nil {
						errs = append(errs, "TXT: "+err.Error())
					} else {
						done = append(done, outFile)
					}
				}

				fyne.Do(func() {
					if len(errs) > 0 {
						statusLabel.SetText("❌ Remote transcription failed")
						resultLabel.SetText("❌ Errors:\n\n" + strings.Join(errs, "\n"))
						busyBar.Hide()
						runBtn.Enable()
						return
					}
					statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
					resultLabel.SetText(fmt.Sprintf("%s\n\n%s\n\n📁 %s:\n%s", T("whisper.outputs_title"), strings.Join(done, "\n"), T("common.output_folder"), outDir))
					showWhisperResultActions(outDir)
					busyBar.Hide()
					runBtn.Enable()
				})
			}()
			return
		}

		if runtime.GOOS == "darwin" {
			cmdFile, err := createWhisperCommandFile(inputFile, outDir, whisperBinEntry.Text, modelEntry.Text, outputPrefix, wantSRT.Checked, wantVTT.Checked, wantTXT.Checked, threads, lang)
			if err != nil {
				fyne.Do(func() {
					busyBar.Hide()
					runBtn.Enable()
					dialog.ShowError(err, w)
					statusLabel.SetText("❌ Failed to start")
				})
				return
			}

			// Fire and forget; Terminal will run it.
			if err := exec.Command("open", cmdFile).Start(); err != nil {
				fyne.Do(func() {
					busyBar.Hide()
					runBtn.Enable()
					dialog.ShowError(fmt.Errorf("failed to open Terminal: %v", err), w)
					statusLabel.SetText("❌ Failed to open Terminal")
				})
				return
			}

			fyne.Do(func() {
				busyBar.Hide()
				runBtn.Enable()
				statusLabel.SetText("Running in Terminal…")
				resultLabel.SetText(fmt.Sprintf("%s\n\n%s\n\n📁 %s:\n%s", T("whisper.terminal_started_outputs"), strings.Join(outputs, "\n"), T("common.output_folder"), outDir))
				showWhisperResultActions(outDir)
			})

			// Best-effort completion detection: watch for output file(s) appearing.
			go func(expected []string, started time.Time) {
				watched := ""
				mainSRT := outputPrefix + ".srt"
				for _, p := range expected {
					if p == mainSRT {
						watched = p
						break
					}
				}
				if watched == "" {
					if len(expected) == 0 {
						return
					}
					watched = expected[0]
				}

				deadline := time.Now().Add(2 * time.Hour)
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				var lastSize int64 = -1
				stableCount := 0

				for time.Now().Before(deadline) {
					<-ticker.C
					info, err := os.Stat(watched)
					if err != nil {
						continue
					}
					if info.Size() > 0 && info.Size() == lastSize {
						stableCount++
					} else {
						stableCount = 0
					}
					lastSize = info.Size()
					if stableCount >= 2 {
						if wantSRTTimestampsOnly.Checked {
							srtOut := outputPrefix + ".srt"
							timestampsOut := outputPrefix + ".timestamps.srt"
							if _, err := os.Stat(srtOut); err == nil {
								if err := writeTimestampsOnlySRT(srtOut, timestampsOut); err != nil {
									appendLog("❌ Failed to generate timestamps-only SRT: " + err.Error())
								}
							}
						}
						fyne.Do(func() {
							statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(started).Round(time.Second)))
							resultLabel.SetText(fmt.Sprintf("%s\n\n%s\n\n📁 %s:\n%s", T("whisper.outputs_title"), strings.Join(expected, "\n"), T("common.output_folder"), outDir))
							showWhisperResultActions(outDir)
						})
						return
					}
				}
			}(outputs, startTime)
			return
		}

		// Non-macOS fallback: run in-process with live log streaming
		go func() {
			defer fyne.Do(func() {
				busyBar.Hide()
				runBtn.Enable()
			})

			// Step 1: Convert to WAV if input is not already WAV
			audioFile := inputFile
			var tmpWav string
			ext := strings.ToLower(filepath.Ext(inputFile))
			if ext != ".wav" {
				fyne.Do(func() {
					statusLabel.SetText("Converting to WAV (16kHz mono)…")
				})
				appendLog("Converting to WAV (16kHz mono)…")

				tmpWav = filepath.Join(os.TempDir(), fmt.Sprintf("subtitle-forge-whisper-%d.wav", time.Now().UnixNano()))
				ffmpegCmd := exec.Command("ffmpeg", "-y", "-i", inputFile, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", tmpWav)
				ffmpegOut, ffmpegErr := ffmpegCmd.CombinedOutput()
				if ffmpegErr != nil {
					appendLog("❌ FFmpeg conversion failed: " + ffmpegErr.Error())
					if len(ffmpegOut) > 0 {
						appendLog(string(ffmpegOut))
					}
					fyne.Do(func() {
						statusLabel.SetText("❌ FFmpeg conversion failed")
						resultLabel.SetText(fmt.Sprintf("❌ Error converting to WAV: %v\n\n%s", ffmpegErr, string(ffmpegOut)))
					})
					return
				}
				audioFile = tmpWav
				appendLog("WAV conversion complete: " + tmpWav)
			}

			// Clean up temp WAV when done
			if tmpWav != "" {
				defer os.Remove(tmpWav)
			}

			// Step 2: Build whisper-cli arguments
			args := []string{"-m", modelEntry.Text, "-f", audioFile, "-of", outputPrefix}
			if lang != "" {
				args = append(args, "-l", lang)
			}
			if threads > 0 {
				args = append(args, "-t", fmt.Sprintf("%d", threads))
			}
			if wantTXT.Checked {
				args = append(args, "-otxt")
			}
			if wantSRT.Checked {
				args = append(args, "-osrt")
			}
			if wantVTT.Checked {
				args = append(args, "-ovtt")
			}

			// Step 3: Run whisper-cli with live output streaming
			fyne.Do(func() {
				statusLabel.SetText("Running whisper-cli…")
			})
			appendLog("Running: " + whisperBinEntry.Text + " " + strings.Join(args, " "))

			cmd := exec.Command(whisperBinEntry.Text, args...)
			cmd.Dir = outDir

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fyne.Do(func() {
					statusLabel.SetText("❌ Failed to start whisper-cli")
					resultLabel.SetText("❌ Error: " + err.Error())
				})
				return
			}
			cmd.Stderr = cmd.Stdout

			if err := cmd.Start(); err != nil {
				fyne.Do(func() {
					statusLabel.SetText("❌ Failed to start whisper-cli")
					resultLabel.SetText("❌ Error: " + err.Error())
				})
				return
			}

			scanner := bufio.NewScanner(stdout)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				appendLog(line)
			}

			err = cmd.Wait()
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("❌ Transcription failed")
					resultLabel.SetText(fmt.Sprintf("❌ whisper-cli failed: %v", err))
					return
				}
				if wantSRT.Checked && wantSRTTimestampsOnly.Checked {
					srtOut := outputPrefix + ".srt"
					timestampsOut := outputPrefix + ".timestamps.srt"
					if err := writeTimestampsOnlySRT(srtOut, timestampsOut); err != nil {
						resultLabel.SetText("⚠️ Transcription finished, but timestamps-only SRT generation failed:\n\n" + err.Error() + "\n\nOutputs:\n\n" + strings.Join(outputs, "\n"))
						statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
						showWhisperResultActions(outDir)
						return
					}
				}
				statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
				resultLabel.SetText(fmt.Sprintf("%s\n\n%s\n\n📁 %s:\n%s", T("whisper.outputs_title"), strings.Join(outputs, "\n"), T("common.output_folder"), outDir))
				showWhisperResultActions(outDir)
			})
		}()
	})
	runBtn.Importance = widget.HighImportance

	config := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Mode"),
		modeSelect,
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Remote URL:"), remoteURLEntry,
			widget.NewLabel("Remote API key:"), remoteKeyEntry,
			widget.NewLabel(""), rememberRemoteKey,
			widget.NewLabel(""), remoteDebugLog,
			widget.NewLabel(""), remoteStreamLogs,
			widget.NewLabel(""), remoteTerminalLogs,
			widget.NewLabel("Remote model:"), remoteModelEntry,
			widget.NewLabel("Remote model list:"), remoteModelSelect,
			widget.NewLabel(""), refreshRemoteModels,
			widget.NewLabel("Remote threads:"), remoteThreadsEntry,
			widget.NewLabel("Remote beam size:"), remoteBeamEntry,
			widget.NewLabel("Remote best-of:"), remoteBestOfEntry,
			widget.NewLabel("Remote language:"), remoteLangEntry,
		),
		widget.NewLabel("Remote API: POST {url}/transcribe/{srt|vtt|txt} or {url}/transcribe/stream/{srt|vtt|txt} for SSE logs. Upload field: file. Header: x-api-key (optional). Response can be raw SRT/VTT/TXT or JSON with keys srt/vtt/text (auto-detected). Optional query params: model, threads, beam_size, best_of, language."),
		widget.NewSeparator(),
		widget.NewLabel("Status"),
		statusLabel,
		busyBar,
		widget.NewSeparator(),
		widget.NewLabel("Input"),
		inputLabel,
		selectInputBtn,
		widget.NewSeparator(),
		widget.NewLabel("Output"),
		outputLabel,
		selectOutputBtn,
		widget.NewSeparator(),
		widget.NewLabel("Whisper CLI"),
		container.NewBorder(nil, nil, nil, browseWhisperBtn, whisperBinEntry),
		widget.NewLabel("Model"),
		container.NewHBox(modelSelect, refreshModelsBtn),
		container.NewBorder(nil, nil, nil, browseModelBtn, modelEntry),
		widget.NewSeparator(),
		langLabel,
		langSelect,
		widget.NewLabel("Model Tools (macOS)"),
		modelNameSelect,
		widget.NewLabel("CoreML Env (optional)"),
		coremlCondaEnvEntry,
		coremlVenvActivateEntry,
		container.NewHBox(installModelBtn, generateCoreMLBtn, layout.NewSpacer()),
		widget.NewSeparator(),
		threadsLabel,
		threadsSlider,
		widget.NewSeparator(),
		widget.NewLabel("Formats"),
		container.NewVBox(
			container.NewHBox(wantSRT, wantVTT, wantTXT, layout.NewSpacer()),
			wantSRTTimestampsOnly,
		),
		widget.NewSeparator(),
		runBtn,
		widget.NewLabel("Result"),
		resultScroll,
		whisperResultActions,
		widget.NewLabel("Log"),
		logScroll,
	)

	refreshModelOptions()
	applyModeUI()

	return config
}

func writeTimestampsOnlySRT(inPath, outPath string) error {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	s := bufio.NewScanner(in)
	// Default scanner token limit (64K) can be too small for some subtitle lines.
	s.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for {
		if !s.Scan() {
			break
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		isDigits := true
		for _, r := range line {
			if r < '0' || r > '9' {
				isDigits = false
				break
			}
		}
		if !isDigits {
			continue
		}

		idxLine := line
		if !s.Scan() {
			break
		}
		timeLine := strings.TrimSpace(s.Text())
		if !strings.Contains(timeLine, " --> ") {
			continue
		}

		if _, err := out.WriteString(idxLine + "\n" + timeLine + "\n\n"); err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func shellQuote(s string) string {
	// Safe single-quote escaping for POSIX shells.
	// abc'def -> 'abc'"'"'def'
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func findWhisperCppRoot(whisperBin, modelPath string) (string, error) {
	tryRoots := []string{}
	if strings.TrimSpace(whisperBin) != "" {
		p := filepath.Dir(whisperBin)
		for i := 0; i < 6; i++ {
			tryRoots = append(tryRoots, p)
			p = filepath.Dir(p)
		}
	}
	if strings.TrimSpace(modelPath) != "" {
		p := filepath.Dir(modelPath)
		for i := 0; i < 4; i++ {
			tryRoots = append(tryRoots, p)
			p = filepath.Dir(p)
		}
	}
	seen := map[string]bool{}
	for _, root := range tryRoots {
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		dl := filepath.Join(root, "models", "download-ggml-model.sh")
		gen := filepath.Join(root, "models", "generate-coreml-model.sh")
		if _, err := os.Stat(dl); err == nil {
			if _, err2 := os.Stat(gen); err2 == nil {
				return root, nil
			}
			return root, nil
		}
	}
	return "", fmt.Errorf("could not locate whisper.cpp root (expected models/download-ggml-model.sh). Check whisper-cli path / model path.")
}

func createTerminalCommandFile(workingDir, title string, cmd []string) (string, error) {
	if strings.TrimSpace(workingDir) == "" {
		return "", fmt.Errorf("working directory is empty")
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("command is empty")
	}
	tmp, err := os.CreateTemp("", "subtitle-forge-whisper-tool-*.command")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	quoted := make([]string, 0, len(cmd))
	for _, c := range cmd {
		quoted = append(quoted, shellQuote(c))
	}

	escapedTitle := strings.ReplaceAll(title, "\"", "\\\"")

	script := `#!/bin/bash
# ── Subtitle Forge ──────────────────────────────────────────
set -e

# ANSI color codes
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
RED='\033[31m'
YELLOW='\033[33m'
MAGENTA='\033[35m'
RESET='\033[0m'

clear
printf "${BOLD}${CYAN}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                                                              ║"
echo "║              🔧  S U B T I T L E   F O R G E               ║"
echo "║                                                              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
printf "${RESET}\n"
printf "${BOLD}${MAGENTA}  ▸ %s${RESET}\n\n" "` + escapedTitle + `"
printf "${DIM}  Working dir: %s${RESET}\n" ` + shellQuote(workingDir) + `
printf "${DIM}  Started:     $(date '+%H:%M:%S')${RESET}\n"
echo ""
printf "${CYAN}──────────────────────────────────────────────────────────────${RESET}\n\n"

cd ` + shellQuote(workingDir) + `

# Spinner for background task
spin() {
  local pid=$1
  local frames='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0
  local start=$SECONDS
  while kill -0 "$pid" 2>/dev/null; do
    local elapsed=$(( SECONDS - start ))
    local mins=$(( elapsed / 60 ))
    local secs=$(( elapsed % 60 ))
    printf "\r${BOLD}${YELLOW}  ${frames:i%${#frames}:1} Running...${RESET} ${DIM}[%02d:%02d]${RESET}  " "$mins" "$secs"
    i=$(( i + 1 ))
    sleep 0.1
  done
  printf "\r                                                    \r"
}

START_TIME=$SECONDS
set +e
` + strings.Join(quoted, " ") + ` &
CMD_PID=$!
spin $CMD_PID
wait $CMD_PID
EXITCODE=$?
set -e

ELAPSED=$(( SECONDS - START_TIME ))
ELAPSED_MIN=$(( ELAPSED / 60 ))
ELAPSED_SEC=$(( ELAPSED % 60 ))

echo ""
printf "${CYAN}──────────────────────────────────────────────────────────────${RESET}\n\n"
if [ $EXITCODE -eq 0 ]; then
  printf "${BOLD}${GREEN}  ✅ Completed successfully${RESET}\n"
else
  printf "${BOLD}${RED}  ❌ Failed with exit code $EXITCODE${RESET}\n"
fi
printf "${DIM}  Duration: %02d:%02d${RESET}\n" "$ELAPSED_MIN" "$ELAPSED_SEC"
printf "${DIM}  Finished: $(date '+%H:%M:%S')${RESET}\n"
echo ""
printf "${DIM}  Press Enter to close this window…${RESET}"
read
`

	if _, err := tmp.WriteString(script); err != nil {
		return "", err
	}
	if err := tmp.Chmod(0755); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func createWhisperCommandFile(inputFile, outDir, whisperBin, modelPath, outputPrefix string, wantSRT, wantVTT, wantTXT bool, threads int, langCode string) (string, error) {
	if strings.TrimSpace(outDir) == "" {
		return "", fmt.Errorf("output directory is empty")
	}
	if strings.TrimSpace(inputFile) == "" {
		return "", fmt.Errorf("input file is empty")
	}
	if strings.TrimSpace(whisperBin) == "" {
		return "", fmt.Errorf("whisper-cli path is empty")
	}
	if strings.TrimSpace(modelPath) == "" {
		return "", fmt.Errorf("model path is empty")
	}

	tmp, err := os.CreateTemp("", "subtitle-forge-whisper-*.command")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	args := []string{"-m", "\"$MODEL\"", "-f", "\"$AUDIO\"", "-of", "\"$OUTPREFIX\""}

	if strings.TrimSpace(langCode) != "" {
		args = append(args, "-l", shellQuote(strings.TrimSpace(langCode)))
	}

	if threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", threads))
	}

	// Build output format list for display
	var fmtList []string
	if wantSRT {
		args = append(args, "-osrt")
		fmtList = append(fmtList, "SRT")
	}
	if wantVTT {
		args = append(args, "-ovtt")
		fmtList = append(fmtList, "VTT")
	}
	if wantTXT {
		args = append(args, "-otxt")
		fmtList = append(fmtList, "TXT")
	}
	fmtDisplay := strings.Join(fmtList, ", ")

	langDisplay := "Auto-detect"
	if strings.TrimSpace(langCode) != "" {
		langDisplay = strings.TrimSpace(langCode)
	}

	script := `#!/bin/bash
# ── Subtitle Forge ──────────────────────────────────────────
set -e

# ANSI color codes
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
RED='\033[31m'
YELLOW='\033[33m'
MAGENTA='\033[35m'
WHITE='\033[97m'
RESET='\033[0m'

clear
printf "${BOLD}${CYAN}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                                                              ║"
echo "║              🔧  S U B T I T L E   F O R G E               ║"
echo "║                                                              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
printf "${RESET}\n"
printf "${BOLD}${MAGENTA}  ▸ Whisper Transcription${RESET}\n\n"

printf "${BOLD}${WHITE}  ┌─ Configuration ─────────────────────────────────────────┐${RESET}\n"
printf "${WHITE}  │${RESET}  ${DIM}Input:${RESET}    %s\n" "$(basename ` + shellQuote(inputFile) + `)"
printf "${WHITE}  │${RESET}  ${DIM}Model:${RESET}    %s\n" "$(basename ` + shellQuote(modelPath) + `)"
printf "${WHITE}  │${RESET}  ${DIM}Language:${RESET}  ` + langDisplay + `\n"
printf "${WHITE}  │${RESET}  ${DIM}Threads:${RESET}   ` + fmt.Sprintf("%d", threads) + `\n"
printf "${WHITE}  │${RESET}  ${DIM}Formats:${RESET}   ` + fmtDisplay + `\n"
printf "${WHITE}  │${RESET}  ${DIM}Output:${RESET}    %s\n" ` + shellQuote(outDir) + `
printf "${BOLD}${WHITE}  └─────────────────────────────────────────────────────────┘${RESET}\n"
printf "${DIM}  Started: $(date '+%H:%M:%S')${RESET}\n\n"

cd ` + shellQuote(outDir) + `
WHISPER=` + shellQuote(whisperBin) + `
MODEL=` + shellQuote(modelPath) + `
INPUT=` + shellQuote(inputFile) + `
OUTPREFIX=` + shellQuote(outputPrefix) + `

# ── Step 1: Audio conversion ────────────────────────────────
AUDIO="$INPUT"
TMPWAV=""
EXT="${INPUT##*.}"
EXT_LOWER="$(echo "$EXT" | tr '[:upper:]' '[:lower:]')"
if [ "$EXT_LOWER" != "wav" ]; then
  TMPWAV="$(mktemp -t subtitle-forge-whisper-XXXXXX).wav"
  printf "${YELLOW}  ⟳ Converting to WAV (16kHz mono)…${RESET}\n"
  ffmpeg -y -loglevel warning -stats -i "$INPUT" -ar 16000 -ac 1 -c:a pcm_s16le "$TMPWAV" 2>&1 | while IFS= read -r line; do
    printf "    ${DIM}%s${RESET}\n" "$line"
  done
  AUDIO="$TMPWAV"
  printf "${GREEN}  ✓ Audio conversion complete${RESET}\n\n"
else
  printf "${GREEN}  ✓ Input is already WAV${RESET}\n\n"
fi

# ── Step 2: Whisper transcription ───────────────────────────
printf "${CYAN}──────────────────────────────────────────────────────────────${RESET}\n"
printf "${BOLD}${YELLOW}  ⟳ Transcribing…${RESET}\n\n"

START_TIME=$SECONDS
set +e
"$WHISPER" ` + strings.Join(args, " ") + ` 2>&1 | while IFS= read -r line; do
  printf "  ${DIM}│${RESET} %s\n" "$line"
done
EXITCODE=${PIPESTATUS[0]}
set -e

# Clean up temp WAV
if [ -n "$TMPWAV" ]; then rm -f "$TMPWAV"; fi

ELAPSED=$(( SECONDS - START_TIME ))
ELAPSED_MIN=$(( ELAPSED / 60 ))
ELAPSED_SEC=$(( ELAPSED % 60 ))

echo ""
printf "${CYAN}──────────────────────────────────────────────────────────────${RESET}\n\n"
if [ $EXITCODE -eq 0 ]; then
  printf "${BOLD}${GREEN}  ✅ Transcription completed successfully!${RESET}\n\n"
  printf "${WHITE}  Output files:${RESET}\n"
  for ext in srt vtt txt; do
    f="${OUTPREFIX}.${ext}"
    if [ -f "$f" ]; then
      SIZE=$(du -h "$f" | cut -f1 | xargs)
      printf "    ${GREEN}✓${RESET} %s ${DIM}(%s)${RESET}\n" "$(basename "$f")" "$SIZE"
    fi
  done
else
  printf "${BOLD}${RED}  ❌ Transcription failed (exit code $EXITCODE)${RESET}\n"
fi
echo ""
printf "${DIM}  Duration:  %02d:%02d${RESET}\n" "$ELAPSED_MIN" "$ELAPSED_SEC"
printf "${DIM}  Finished:  $(date '+%H:%M:%S')${RESET}\n"
echo ""
printf "${DIM}  Press Enter to close this window…${RESET}"
read
`

	if _, err := tmp.WriteString(script); err != nil {
		return "", err
	}
	if err := tmp.Chmod(0755); err != nil {
		return "", err
	}

	return tmp.Name(), nil
}
