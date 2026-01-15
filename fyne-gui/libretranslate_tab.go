package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	title := widget.NewLabel("LibreTranslate")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	defaultURL := a.Preferences().StringWithFallback("libretranslate_url", "http://127.0.0.1:5000")
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("http://127.0.0.1:5000")
	urlEntry.SetText(defaultURL)
	urlEntry.OnChanged = func(s string) { a.Preferences().SetString("libretranslate_url", s) }

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Optional API key")
	apiKeyEntry.SetText(a.Preferences().StringWithFallback("libretranslate_api_key", ""))

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
	inputLabel := widget.NewLabel("No SRT files selected")
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
				widget.NewButton("Remove", func(index int) func() {
					return func() {
						inputFiles = append(inputFiles[:index], inputFiles[index+1:]...)
						updateFileList()
						if len(inputFiles) == 0 {
							inputLabel.SetText("No SRT files selected")
						} else if len(inputFiles) == 1 {
							inputLabel.SetText("Selected: " + filepath.Base(inputFiles[0]))
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
			inputLabel.SetText("Selected: " + filepath.Base(inputFiles[0]))
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

	clearBtn := widget.NewButton("Clear", func() {
		inputFiles = nil
		inputLabel.SetText("No SRT files selected")
		updateFileList()
	})

	selectInputBtn := widget.NewButton("Select SRT File", func() {
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
			langs, err := fetchLibreTranslateLanguages(base)
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

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	logGrid := widget.NewTextGrid()
	logScroll := container.NewScroll(logGrid)
	logScroll.SetMinSize(fyne.NewSize(0, 140))
	var logText string

	appendLog := func(msg string) {
		ts := time.Now().Format("15:04:05")
		fyne.Do(func() {
			logText += fmt.Sprintf("[%s] %s\n", ts, msg)
			logGrid.SetText(logText)
			logScroll.ScrollToBottom()
		})
	}

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
						resultLabel.SetText("✅ Translation completed\n\n" + lastOutFile)
					} else {
						resultLabel.SetText(fmt.Sprintf("✅ Translation completed: %d file(s)", okCount))
					}
					return
				}
				if okCount == 0 {
					resultLabel.SetText(fmt.Sprintf("❌ Translation failed: %d file(s)", failCount))
					return
				}
				resultLabel.SetText(fmt.Sprintf("⚠️ Translation completed with errors: %d ok, %d failed", okCount, failCount))
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
		widget.NewLabel("Log"),
		logScroll,
		resultScroll,
	)

	return content
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

func fetchLibreTranslateLanguages(baseURL string) ([]struct{ Code, Name string }, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/languages")
	if err != nil {
		return nil, err
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
