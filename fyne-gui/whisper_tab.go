package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
	title := widget.NewLabel("Whisper Transcription")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("Ready")
	statusLabel.Wrapping = fyne.TextWrapWord
	busyBar := widget.NewProgressBarInfinite()
	busyBar.Hide()

	var inputFile string
	inputLabel := widget.NewLabel("No media/audio file selected")
	inputLabel.Wrapping = fyne.TextWrapWord

	whisperTranscribeSetInputFile = func(path string) {
		inputFile = path
		inputLabel.SetText("Selected: " + filepath.Base(inputFile))
	}

	selectInputBtn := widget.NewButton("Select Media/Audio File", func() {
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
	outputLabel := widget.NewLabel("Output: Same directory as input")
	outputLabel.Wrapping = fyne.TextWrapWord

	selectOutputBtn := widget.NewButton("Select Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText("Output: " + outputDir)
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

	remoteKeyEntry := widget.NewPasswordEntry()
	remoteKeyEntry.SetPlaceHolder("Optional x-api-key")
	remoteKeyEntry.SetText(a.Preferences().StringWithFallback("whisper_remote_api_key", ""))

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
		homeDir, _ := os.UserHomeDir()
		defaultWhisperBin = filepath.Join(homeDir, ".whispercpp-coreml", "whisper.cpp", "build", "bin", "whisper-cli")
	}

	whisperBinEntry := widget.NewEntry()
	whisperBinEntry.SetPlaceHolder("Path to whisper-cli")
	whisperBinEntry.SetText(defaultWhisperBin)
	whisperBinEntry.OnChanged = func(s string) {
		a.Preferences().SetString("whisper_cli_path", s)
	}

	defaultModel := a.Preferences().StringWithFallback("whisper_model_path", "")
	if defaultModel == "" {
		homeDir, _ := os.UserHomeDir()
		defaultModel = filepath.Join(homeDir, ".whispercpp-coreml", "whisper.cpp", "models", "ggml-base.en.bin")
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

	logGrid := widget.NewTextGrid()
	logScroll := container.NewScroll(logGrid)
	logScroll.SetMinSize(fyne.NewSize(0, 120))
	var logText string
	appendLog := func(msg string) {
		ts := time.Now().Format("15:04:05")
		fyne.Do(func() {
			logText += fmt.Sprintf("[%s] %s\n", ts, msg)
			logGrid.SetText(logText)
			logScroll.ScrollToBottom()
		})
	}

	applyModeUI := func() {
		isRemote := modeSelect.Selected == "Remote API"
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
			return
		}

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
			if strings.TrimSpace(modelEntry.Text) == "" {
				dialog.ShowError(fmt.Errorf("please set a GGML model path"), w)
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

			go func() {
				defer fyne.Do(func() {
					busyBar.Hide()
					runBtn.Enable()
				})

				callOne := func(format, outFile string) error {
					trimmed := strings.TrimSpace(baseURL)
					if trimmed == "" {
						return fmt.Errorf("remote URL is empty")
					}
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
					appendLog("Waiting for server response…")

					respBody, err := io.ReadAll(resp.Body)
					if err != nil {
						return err
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
						return
					}
					statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
					resultLabel.SetText("✅ Outputs:\n\n" + strings.Join(done, "\n"))
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
				resultLabel.SetText("Terminal started. Outputs will be written to:\n\n" + strings.Join(outputs, "\n"))
			})

			// Best-effort completion detection: watch for output file(s) appearing.
			// We can't reliably get Terminal process exit status, so we observe filesystem outputs.
			go func(expected []string, started time.Time) {
				// Prefer the main SRT for completion if selected. Note that *.timestamps.srt also ends in .srt,
				// so we must avoid accidentally watching the timestamps file (it doesn't exist until after).
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

				// Poll for up to 2 hours.
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
							resultLabel.SetText("✅ Outputs:\n\n" + strings.Join(expected, "\n"))
						})
						return
					}
				}
			}(outputs, startTime)
			return
		}

		// Non-macOS fallback: run without live output, capture output on completion.
		go func() {
			defer fyne.Do(func() {
				busyBar.Hide()
				runBtn.Enable()
			})

			args := []string{"-m", modelEntry.Text, "-f", inputFile, "-of", outputPrefix}
			if lang != "" {
				args = append(args, "-l", lang)
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

			cmd := exec.Command(whisperBinEntry.Text, args...)
			cmd.Dir = outDir
			out, err := cmd.CombinedOutput()
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("❌ Transcription failed")
					resultLabel.SetText(fmt.Sprintf("❌ Error: %v\n\n%s", err, string(out)))
					return
				}
				if wantSRT.Checked && wantSRTTimestampsOnly.Checked {
					srtOut := outputPrefix + ".srt"
					timestampsOut := outputPrefix + ".timestamps.srt"
					if err := writeTimestampsOnlySRT(srtOut, timestampsOut); err != nil {
						resultLabel.SetText("⚠️ Transcription finished, but timestamps-only SRT generation failed:\n\n" + err.Error() + "\n\nOutputs:\n\n" + strings.Join(outputs, "\n"))
						statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
						return
					}
				}
				statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
				resultLabel.SetText("✅ Outputs:\n\n" + strings.Join(outputs, "\n"))
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
		),
		widget.NewLabel("Remote API: POST {url}/transcribe/{srt|vtt|txt} (or paste a full /transcribe/... URL). Upload field: file. Header: x-api-key (optional). Response can be raw SRT/VTT/TXT or JSON with keys srt/vtt/text (auto-detected)."),
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

	script := "#!/bin/bash\n" +
		"set -e\n" +
		"echo \"" + strings.ReplaceAll(title, "\"", "\\\"") + "\"\n" +
		"echo \"Working dir: " + workingDir + "\"\n" +
		"echo \"\"\n" +
		"cd " + shellQuote(workingDir) + "\n" +
		"set +e\n" +
		strings.Join(quoted, " ") + "\n" +
		"EXITCODE=$?\n" +
		"set -e\n" +
		"echo \"\"\n" +
		"if [ $EXITCODE -eq 0 ]; then\n" +
		"  echo \"✅ Done.\"\n" +
		"else\n" +
		"  echo \"❌ Failed with exit code $EXITCODE\"\n" +
		"fi\n" +
		"echo \"\"\n" +
		"echo \"Press Enter to close this window…\"\n" +
		"read\n"

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

	// Add threads option
	if threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", threads))
	}

	if wantTXT {
		args = append(args, "-otxt")
	}
	if wantSRT {
		args = append(args, "-osrt")
	}
	if wantVTT {
		args = append(args, "-ovtt")
	}

	script := "#!/bin/bash\n" +
		"set -e\n" +
		"echo \"Subtitle Forge - Whisper Transcription\"\n" +
		"cd " + shellQuote(outDir) + "\n" +
		"WHISPER=" + shellQuote(whisperBin) + "\n" +
		"MODEL=" + shellQuote(modelPath) + "\n" +
		"INPUT=" + shellQuote(inputFile) + "\n" +
		"OUTPREFIX=" + shellQuote(outputPrefix) + "\n" +
		"echo \"Input: $INPUT\"\n" +
		"echo \"Output dir: " + shellQuote(outDir) + "\"\n" +
		"echo \"\"\n" +
		"AUDIO=\"$INPUT\"\n" +
		"TMPWAV=\"\"\n" +
		"EXT=\"${INPUT##*.}\"\n" +
		"EXT_LOWER=\"$(echo \"$EXT\" | tr '[:upper:]' '[:lower:]')\"\n" +
		"if [ \"$EXT_LOWER\" != \"wav\" ]; then\n" +
		"  TMPWAV=\"$(mktemp -t subtitle-forge-whisper-XXXXXX).wav\"\n" +
		"  echo \"Converting to WAV (16kHz mono): $TMPWAV\"\n" +
		"  ffmpeg -y -i \"$INPUT\" -ar 16000 -ac 1 -c:a pcm_s16le \"$TMPWAV\"\n" +
		"  AUDIO=\"$TMPWAV\"\n" +
		"fi\n" +
		"echo \"\"\n" +
		"echo \"Running whisper-cli…\"\n" +
		"set +e\n" +
		"\"$WHISPER\" " + strings.Join(args, " ") + "\n" +
		"EXITCODE=$?\n" +
		"set -e\n" +
		"if [ -n \"$TMPWAV\" ]; then rm -f \"$TMPWAV\"; fi\n" +
		"echo \"\"\n" +
		"if [ $EXITCODE -eq 0 ]; then\n" +
		"  echo \"✅ Done.\"\n" +
		"else\n" +
		"  echo \"❌ whisper-cli failed with exit code $EXITCODE\"\n" +
		"fi\n" +
		"echo \"\"\n" +
		"echo \"Press Enter to close this window…\"\n" +
		"read\n"

	if _, err := tmp.WriteString(script); err != nil {
		return "", err
	}
	if err := tmp.Chmod(0755); err != nil {
		return "", err
	}

	return tmp.Name(), nil
}
