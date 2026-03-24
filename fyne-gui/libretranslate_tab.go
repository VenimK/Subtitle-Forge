package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var libreTranslateSetInputFile func(string)

var libreTranslateAddInputFile func(string)

type libreTranslateLanguagesResponse []struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func createLibreTranslateTab(w fyne.Window, a fyne.App) *fyne.Container {
	title := widget.NewLabel(T("libre.title"))
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	defaultURL := a.Preferences().StringWithFallback("libretranslate_url", "http://127.0.0.1:5100")
	if defaultURL == "http://127.0.0.1:5000" {
		defaultURL = "http://127.0.0.1:5100"
		a.Preferences().SetString("libretranslate_url", defaultURL)
	}
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("http://127.0.0.1:5100")
	urlEntry.SetText(defaultURL)
	urlEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_url", s) }

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Optional API key")
	apiKeyEntry.SetText(a.Preferences().StringWithFallback("libretranslate_api_key", ""))

	threadsEntry := widget.NewEntry()
	threadsEntry.SetPlaceHolder("8")
	threadsEntry.SetText(a.Preferences().StringWithFallback("libretranslate_threads", "8"))
	threadsEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_threads", s) }

	reqLimitEntry := widget.NewEntry()
	reqLimitEntry.SetPlaceHolder("60")
	reqLimitEntry.SetText(a.Preferences().StringWithFallback("libretranslate_req_limit", "60"))
	reqLimitEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_req_limit", s) }

	rememberKey := widget.NewCheck("Remember API key (saved locally)", nil)
	rememberKey.Checked = a.Preferences().BoolWithFallback("libretranslate_remember_api_key", false)
	rememberKey.OnChanged = func(checked bool) {
		a.Preferences().SetBool("libretranslate_remember_api_key", checked)
		if !checked {
			a.Preferences().SetString("libretranslate_api_key", "")
		}
	}
	apiKeyEntry.OnChanged = func(s string) {
		if rememberKey.Checked {
			a.Preferences().SetString("libretranslate_api_key", s)
		}
	}

	var inputFiles []string
	inputLabel := widget.NewLabel(T("libre.no_files"))
	inputLabel.Wrapping = fyne.TextWrapWord

	fileList := container.NewVBox()
	fileListScroll := container.NewScroll(fileList)
	fileListScroll.SetMinSize(fyne.NewSize(0, 120))

	var updateFileList func()
	updateFileList = func() {
		fileList.Objects = nil
		for i, file := range inputFiles {
			fileName := filepath.Base(file)
			fileRow := container.NewHBox(
				widget.NewLabel(fmt.Sprintf("%d.", i+1)),
				widget.NewLabel(fileName),
				layout.NewSpacer(),
				widget.NewButton(T("common.remove"), func(index int) func() {
					return func() {
						inputFiles = append(inputFiles[:index], inputFiles[index+1:]...)
						updateFileList()
						if len(inputFiles) == 0 {
							inputLabel.SetText(T("libre.no_files"))
						} else if len(inputFiles) == 1 {
							inputLabel.SetText(T("whisper.selected") + filepath.Base(inputFiles[0]))
						} else {
							inputLabel.SetText(fmt.Sprintf("%d SRT files selected", len(inputFiles)))
						}
					}
				}(i)),
			)
			fileList.Add(fileRow)
		}
		fileList.Refresh()
	}

	addInputFile := func(path string) {
		if strings.ToLower(filepath.Ext(path)) != ".srt" {
			return
		}
		for _, existing := range inputFiles {
			if existing == path {
				return
			}
		}
		inputFiles = append(inputFiles, path)
		if len(inputFiles) == 1 {
			inputLabel.SetText(T("whisper.selected") + filepath.Base(inputFiles[0]))
		} else {
			inputLabel.SetText(fmt.Sprintf("%d SRT files selected", len(inputFiles)))
		}
		updateFileList()
	}

	libreTranslateSetInputFile = func(path string) {
		inputFiles = nil
		addInputFile(path)
	}

	libreTranslateAddInputFile = func(path string) {
		addInputFile(path)
	}

	clearBtn := widget.NewButton(T("common.clear"), func() {
		inputFiles = nil
		inputLabel.SetText(T("libre.no_files"))
		updateFileList()
	})

	selectInputBtn := widget.NewButton(T("libre.select_file"), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			libreTranslateAddInputFile(reader.URI().Path())
		}, w)
		fd.Show()
	})
	selectInputBtn.Importance = widget.HighImportance

	outputDir := ""
	outputLabel := widget.NewLabel(T("libre.output_same_dir"))
	outputLabel.Wrapping = fyne.TextWrapWord
	selectOutputBtn := widget.NewButton(T("libre.select_output"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText(T("libre.output_prefix") + outputDir)
		}, w)
	})

	sourceEntry := widget.NewEntry()
	sourceEntry.SetPlaceHolder("en")
	sourceEntry.SetText(a.Preferences().StringWithFallback("libretranslate_source", "en"))
	sourceEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_source", s) }

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("nl")
	targetEntry.SetText(a.Preferences().StringWithFallback("libretranslate_target", "nl"))
	targetEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_target", s) }

	languagesLabel := widget.NewLabel("")
	languagesLabel.Wrapping = fyne.TextWrapWord

	var refreshLanguagesBtn *widget.Button
	refreshLanguagesBtn = widget.NewButton("Fetch Supported Languages", func() {
		base := strings.TrimSpace(urlEntry.Text)
		if base == "" {
			dialog.ShowError(fmt.Errorf("please set LibreTranslate URL"), w)
			return
		}

		progress := dialog.NewProgressInfinite("LibreTranslate", "Fetching languages…", w)
		progress.Show()
		refreshLanguagesBtn.Disable()

		go func() {
			langs, err := fetchLibreTranslateLanguages(base, apiKeyEntry.Text)
			fyne.Do(func() {
				progress.Hide()
				refreshLanguagesBtn.Enable()
				if err != nil {
					languagesLabel.SetText(fmt.Sprintf("❌ Failed to fetch languages: %v", err))
					return
				}

				if len(langs) == 0 {
					languagesLabel.SetText("No languages returned.")
					return
				}

				var parts []string
				for _, l := range langs {
					if l.Code == "" {
						continue
					}
					parts = append(parts, fmt.Sprintf("%s (%s)", l.Name, l.Code))
				}
				languagesLabel.SetText(strings.Join(parts, ", "))
			})
		}()
	})

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord
	resultScroll := container.NewScroll(resultLabel)
	resultScroll.SetMinSize(fyne.NewSize(0, 200))

	libreResultActions := container.NewHBox()
	libreResultActions.Hide()

	showLibreResultActions := func(path string) {
		if strings.TrimSpace(path) == "" {
			libreResultActions.Hide()
			libreResultActions.Objects = nil
			libreResultActions.Refresh()
			return
		}
		libreResultActions.Objects = []fyne.CanvasObject{
			widget.NewButton(T("common.open_output_folder"), func() {
				openFolderPath(w, path)
			}),
			widget.NewButton(T("common.copy_output_path"), func() {
				copyTextToClipboard(w, path)
			}),
		}
		libreResultActions.Show()
		libreResultActions.Refresh()
	}

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	logGrid := widget.NewTextGrid()
	logScroll := container.NewScroll(logGrid)
	logScroll.SetMinSize(fyne.NewSize(0, 140))
	const maxLibreLogLines = 500
	var logLines []string

	appendLog := func(msg string) {
		ts := time.Now().Format("15:04:05")
		fyne.Do(func() {
			logLines = append(logLines, fmt.Sprintf("[%s] %s", ts, msg))
			if len(logLines) > maxLibreLogLines {
				logLines = logLines[len(logLines)-maxLibreLogLines:]
			}
			logGrid.SetText(strings.Join(logLines, "\n"))
			logScroll.ScrollToBottom()
		})
	}

	// Server management: status indicator and smart buttons
	serverStatusLabel := widget.NewLabel("🔴 Stopped")
	serverStatusLabel.TextStyle = fyne.TextStyle{Bold: true}

	scriptAvailable := runtime.GOOS == "darwin" && libreHelperScriptPath() != ""

	startBtn := widget.NewButton("Start Server", nil)
	stopBtn := widget.NewButton("Stop Server", nil)
	installBtn := widget.NewButton("Install LibreTranslate", nil)

	autoFillAPIKey := func() {
		if strings.TrimSpace(apiKeyEntry.Text) != "" {
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		keyFile := filepath.Join(home, ".libretranslate", "api_key.txt")
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return
		}
		key := strings.TrimSpace(string(data))
		if key != "" {
			fyne.Do(func() {
				apiKeyEntry.SetText(key)
				a.Preferences().SetString("libretranslate_api_key", key)
			})
		}
	}

	updateButtonStates := func(running bool) {
		if running {
			serverStatusLabel.SetText("🟢 Running")
			startBtn.Disable()
			if scriptAvailable {
				stopBtn.Enable()
			}
			installBtn.Disable()
		} else {
			serverStatusLabel.SetText("🔴 Stopped")
			if scriptAvailable {
				startBtn.Enable()
			}
			stopBtn.Disable()
			if scriptAvailable && !isLibreLocalInstallPresent() {
				installBtn.Enable()
			} else {
				installBtn.Disable()
			}
		}
		if !scriptAvailable {
			startBtn.Disable()
			stopBtn.Disable()
			installBtn.Disable()
		}
	}

	checkServerStatus := func() {
		baseURL := strings.TrimSpace(urlEntry.Text)
		if baseURL == "" {
			baseURL = "http://127.0.0.1:5100"
		}
		running := isLibreTranslateServerUp(baseURL)
		if running {
			autoFillAPIKey()
		}
		fyne.Do(func() {
			updateButtonStates(running)
		})
	}

	parsePositiveInt := func(label, raw string) (int, error) {
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer", label)
		}
		return n, nil
	}

	runLibreMac := func(subcmd string) {
		if runtime.GOOS != "darwin" {
			appendLog("❌ Local server helper is only available on macOS")
			return
		}
		script := libreHelperScriptPath()
		if script == "" {
			appendLog("❌ Libre-mac.sh not found. Make sure it exists next to the app or in the repo root.")
			return
		}

		threads, err := parsePositiveInt("THREADS", threadsEntry.Text)
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, w) })
			return
		}
		reqLimit, err := parsePositiveInt("REQ_LIMIT", reqLimitEntry.Text)
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, w) })
			return
		}

		appendLog(fmt.Sprintf("Running: %s (THREADS=%d, REQ_LIMIT=%d)", subcmd, threads, reqLimit))

		cmd := exec.Command("bash", script, subcmd)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("THREADS=%d", threads),
			fmt.Sprintf("REQ_LIMIT=%d", reqLimit),
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			appendLog(fmt.Sprintf("❌ Failed to create pipe: %v", err))
			return
		}
		cmd.Stderr = cmd.Stdout // merge stderr into stdout pipe

		if err := cmd.Start(); err != nil {
			appendLog(fmt.Sprintf("❌ Failed to start: %v", err))
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				appendLog(line)
			}
		}

		if err := cmd.Wait(); err != nil {
			appendLog(fmt.Sprintf("❌ Command failed: %v", err))
		} else {
			appendLog("✅ Done")
		}

		// Re-check server status after command completes
		if subcmd == "start-bg" {
			// Server needs time to initialize; poll until it responds or timeout
			appendLog("Waiting for server to become ready...")
			for i := 0; i < 15; i++ {
				time.Sleep(2 * time.Second)
				baseURL := strings.TrimSpace(urlEntry.Text)
				if baseURL == "" {
					baseURL = "http://127.0.0.1:5100"
				}
				if isLibreTranslateServerUp(baseURL) {
					appendLog("✅ Server is ready")
					checkServerStatus()
					return
				}
			}
			appendLog("⚠️ Server did not respond within 30s — it may still be loading")
		} else {
			time.Sleep(1 * time.Second)
		}
		checkServerStatus()
	}

	runLibreInstall := func() {
		if runtime.GOOS != "darwin" {
			appendLog("❌ Install helper is only available on macOS")
			return
		}
		script := libreHelperScriptPath()
		if script == "" {
			appendLog("❌ Libre-mac.sh not found")
			return
		}

		absScript, err := filepath.Abs(script)
		if err != nil {
			absScript = script
		}

		appendLog("Opening terminal for LibreTranslate installation...")
		osascript := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "bash %s install"
end tell`, absScript)
		cmd := exec.Command("osascript", "-e", osascript)
		if out, err := cmd.CombinedOutput(); err != nil {
			appendLog(fmt.Sprintf("❌ Failed to open terminal: %v (%s)", err, strings.TrimSpace(string(out))))
		} else {
			appendLog("✅ Install started in Terminal. Close terminal when done.")
		}
	}

	startBtn.OnTapped = func() {
		startBtn.Disable()
		serverStatusLabel.SetText("🟡 Starting...")
		go runLibreMac("start-bg")
	}
	stopBtn.OnTapped = func() {
		stopBtn.Disable()
		serverStatusLabel.SetText("🟡 Stopping...")
		go func() {
			runLibreMac("stop")

			// If server is still responding after script stop, kill the orphan process by port
			baseURL := strings.TrimSpace(urlEntry.Text)
			if baseURL == "" {
				baseURL = "http://127.0.0.1:5100"
			}
			if isLibreTranslateServerUp(baseURL) {
				appendLog("Server still responding — finding orphan process...")
				// Extract port from URL
				port := "5000"
				if u, err := url.Parse(baseURL); err == nil && u.Port() != "" {
					port = u.Port()
				}
				// Find PID of the process listening on the port
				pidOut, err := exec.Command("bash", "-c",
					fmt.Sprintf("lsof -nP -i TCP:%s -sTCP:LISTEN -t 2>/dev/null | head -1", port),
				).Output()
				pid := strings.TrimSpace(string(pidOut))
				if err != nil || pid == "" {
					appendLog(fmt.Sprintf("⚠️ No process found listening on port %s", port))
				} else {
					appendLog(fmt.Sprintf("Killing PID %s on port %s...", pid, port))
					if out, err := exec.Command("kill", pid).CombinedOutput(); err != nil {
						appendLog(fmt.Sprintf("⚠️ kill failed: %v %s", err, strings.TrimSpace(string(out))))
					} else {
						appendLog("✅ Orphan process killed")
					}
				}
				time.Sleep(1 * time.Second)
				checkServerStatus()
			}
		}()
	}
	installBtn.OnTapped = func() {
		go runLibreInstall()
	}

	// Initial button state
	updateButtonStates(false)

	// Auto-check server status on tab creation
	go checkServerStatus()

	// Periodic status polling (every 10 seconds)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			checkServerStatus()
		}
	}()

	var translateBtn *widget.Button
	translateBtn = widget.NewButton("Translate SRT", func() {
		if len(inputFiles) == 0 {
			dialog.ShowError(fmt.Errorf("please select one or more SRT files"), w)
			return
		}
		baseURL := strings.TrimSpace(urlEntry.Text)
		if baseURL == "" {
			dialog.ShowError(fmt.Errorf("please set LibreTranslate URL"), w)
			return
		}

		src := strings.TrimSpace(sourceEntry.Text)
		tgt := strings.TrimSpace(targetEntry.Text)
		if src == "" || tgt == "" {
			dialog.ShowError(fmt.Errorf("please set source and target language codes"), w)
			return
		}

		translateBtn.Disable()
		selectInputBtn.Disable()
		selectOutputBtn.Disable()
		clearBtn.Disable()
		refreshLanguagesBtn.Disable()
		fyne.Do(func() {
			statusLabel.SetText("Starting…")
			progressBar.SetValue(0)
			progressBar.Show()
			showLibreResultActions("")
		})

		go func() {
			apiKey := apiKeyEntry.Text
			appendLog("Starting translation")

			if err := checkLibreTranslateReachable(baseURL); err != nil {
				fyne.Do(func() {
					translateBtn.Enable()
					selectInputBtn.Enable()
					selectOutputBtn.Enable()
					clearBtn.Enable()
					refreshLanguagesBtn.Enable()
					progressBar.Hide()
					statusLabel.SetText("")
					resultLabel.SetText(fmt.Sprintf("❌ Cannot reach LibreTranslate server: %v", err))
				})
				appendLog(fmt.Sprintf("Cannot reach server: %v", err))
				return
			}
			appendLog("Server reachable")

			var okCount int
			var failCount int
			var lastOutFile string
			var lastOutDir string
			for i, inFile := range inputFiles {
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Translating %d/%d: %s", i+1, len(inputFiles), filepath.Base(inFile)))
					if len(inputFiles) > 0 {
						progressBar.SetValue(float64(i) / float64(len(inputFiles)))
					}
				})
				appendLog(fmt.Sprintf("[%d/%d] Translating %s", i+1, len(inputFiles), filepath.Base(inFile)))
				outDir := outputDir
				if outDir == "" {
					outDir = filepath.Dir(inFile)
				}
				lastOutDir = outDir
				baseName := strings.TrimSuffix(filepath.Base(inFile), filepath.Ext(inFile))
				outFile := filepath.Join(outDir, fmt.Sprintf("%s.%s.srt", baseName, tgt))
				err := libreTranslateTranslateFile(baseURL, apiKey, inFile, src, tgt, outFile)
				if err != nil {
					failCount++
					appendLog(fmt.Sprintf("❌ Failed: %v", err))
					continue
				}
				okCount++
				lastOutFile = outFile
				appendLog("✅ Completed: " + outFile)
			}
			fyne.Do(func() {
				if len(inputFiles) > 0 {
					progressBar.SetValue(1)
				}
			})

			fyne.Do(func() {
				translateBtn.Enable()
				selectInputBtn.Enable()
				selectOutputBtn.Enable()
				clearBtn.Enable()
				refreshLanguagesBtn.Enable()
				progressBar.Hide()
				statusLabel.SetText("")
				if okCount > 0 && failCount == 0 {
					if okCount == 1 {
						resultLabel.SetText(fmt.Sprintf("%s\n\n%s\n\n📁 %s:\n%s", T("libre.translation_completed"), lastOutFile, T("common.output_folder"), lastOutDir))
					} else {
						resultLabel.SetText(fmt.Sprintf(T("libre.translation_completed_count"), okCount, T("common.output_folder"), lastOutDir))
					}
					showLibreResultActions(lastOutDir)
					return
				}
				if okCount == 0 {
					resultLabel.SetText(fmt.Sprintf(T("libre.translation_failed_count"), failCount))
					return
				}
				resultLabel.SetText(fmt.Sprintf(T("libre.translation_completed_with_errors"), okCount, failCount, T("common.output_folder"), lastOutDir))
				showLibreResultActions(lastOutDir)
			})
			appendLog(fmt.Sprintf("Finished: %d ok, %d failed", okCount, failCount))
		}()
	})
	translateBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Server"),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("URL:"), urlEntry,
			widget.NewLabel("API key:"), apiKeyEntry,
			widget.NewLabel(""), rememberKey,
		),
		widget.NewLabel("Advanced (local server)"),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("THREADS:"), threadsEntry,
			widget.NewLabel("REQ_LIMIT:"), reqLimitEntry,
		),
		container.NewHBox(refreshLanguagesBtn, layout.NewSpacer()),
		languagesLabel,
		widget.NewSeparator(),
		widget.NewLabel("Input"),
		inputLabel,
		container.NewHBox(selectInputBtn, clearBtn, layout.NewSpacer()),
		fileListScroll,
		widget.NewSeparator(),
		widget.NewLabel("Languages"),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Source:"), sourceEntry,
			widget.NewLabel("Target:"), targetEntry,
		),
		widget.NewSeparator(),
		widget.NewLabel("Output"),
		outputLabel,
		selectOutputBtn,
		widget.NewSeparator(),
		translateBtn,
		statusLabel,
		progressBar,
		func() *fyne.Container {
			hint := widget.NewLabel("THREADS/REQ_LIMIT apply only to a local LibreTranslate server started via Libre-mac.sh (macOS). For remote servers, change these on the server.")
			hint.Wrapping = fyne.TextWrapWord

			serverLabel := widget.NewLabel("Local Server")
			serverLabel.TextStyle = fyne.TextStyle{Bold: true}

			return container.NewVBox(
				serverLabel,
				container.NewHBox(widget.NewLabel("Status:"), serverStatusLabel, layout.NewSpacer()),
				container.NewHBox(startBtn, stopBtn, installBtn),
				hint,
			)
		}(),
		container.NewHBox(widget.NewLabel("Log"), layout.NewSpacer(), widget.NewButton("Clear", func() {
			logLines = nil
			logGrid.SetText("")
		})),
		logScroll,
		resultScroll,
		libreResultActions,
	)

	return content
}

// isLibreTranslateServerUp checks if a LibreTranslate server is running on the given URL.
// It hits /languages which is specific to LibreTranslate:
//   - 200 → running, no auth required
//   - 403 → running, API key required
//   - anything else (or connection error) → not LibreTranslate
func isLibreTranslateServerUp(baseURL string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/languages")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden
}

func checkLibreTranslateReachable(baseURL string) error {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/languages")
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func fetchLibreTranslateLanguages(baseURL, apiKey string) ([]struct{ Code, Name string }, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/languages")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) != "" {
		q := u.Query()
		q.Set("api_key", apiKey)
		u.RawQuery = q.Encode()
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var decoded libreTranslateLanguagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	out := make([]struct{ Code, Name string }, 0, len(decoded))
	for _, l := range decoded {
		out = append(out, struct{ Code, Name string }{Code: l.Code, Name: l.Name})
	}
	return out, nil
}

func libreTranslateTranslateFile(baseURL, apiKey, inputFile, source, target, outputFile string) error {
	endpoint := strings.TrimRight(baseURL, "/") + "/translate_file"

	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(inputFile))
	if err != nil {
		_ = writer.Close()
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		_ = writer.Close()
		return err
	}

	_ = writer.WriteField("source", source)
	_ = writer.WriteField("target", target)
	if strings.TrimSpace(apiKey) != "" {
		_ = writer.WriteField("api_key", apiKey)
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	// Some LibreTranslate instances return raw SRT file contents; others return JSON with translatedFileUrl
	if bytes.HasPrefix(bytes.TrimSpace(respBody), []byte("{")) {
		var parsed map[string]any
		if err := json.Unmarshal(respBody, &parsed); err == nil {
			if v, ok := parsed["translatedFileUrl"].(string); ok && v != "" {
				dlURL := v
				if baseU, err := url.Parse(baseURL); err == nil {
					// Handle relative URLs if server returns them
					if u2, err := url.Parse(v); err == nil && !u2.IsAbs() {
						dlURL = baseU.ResolveReference(u2).String()
					}
				}

				resp2, err := client.Get(dlURL)
				if err != nil {
					return err
				}
				defer resp2.Body.Close()
				if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
					b, _ := io.ReadAll(io.LimitReader(resp2.Body, 16*1024))
					return fmt.Errorf("download HTTP %d: %s", resp2.StatusCode, strings.TrimSpace(string(b)))
				}
				data, err := io.ReadAll(resp2.Body)
				if err != nil {
					return err
				}
				return os.WriteFile(outputFile, data, 0644)
			}
		}
	}

	return os.WriteFile(outputFile, respBody, 0644)
}
