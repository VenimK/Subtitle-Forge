package main

import (
	"fmt"
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

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord
	resultScroll := container.NewScroll(resultLabel)
	resultScroll.SetMinSize(fyne.NewSize(0, 200))

	var runBtn *widget.Button
	runBtn = widget.NewButton("Start Transcription", func() {
		if inputFile == "" {
			dialog.ShowError(fmt.Errorf("please select an input file"), w)
			return
		}
		if strings.TrimSpace(whisperBinEntry.Text) == "" {
			dialog.ShowError(fmt.Errorf("please set whisper-cli path"), w)
			return
		}
		if strings.TrimSpace(modelEntry.Text) == "" {
			dialog.ShowError(fmt.Errorf("please set a GGML model path"), w)
			return
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
		}
		if wantVTT.Checked {
			outputs = append(outputs, outputPrefix+".vtt")
		}
		if wantTXT.Checked {
			outputs = append(outputs, outputPrefix+".txt")
		}

		threads := int(threadsSlider.Value)

		if runtime.GOOS == "darwin" {
			cmdFile, err := createWhisperCommandFile(inputFile, outDir, whisperBinEntry.Text, modelEntry.Text, outputPrefix, wantSRT.Checked, wantVTT.Checked, wantTXT.Checked, threads)
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
				// Prefer SRT for completion if selected; otherwise first expected file.
				watchList := make([]string, 0, len(expected))
				for _, p := range expected {
					if strings.HasSuffix(strings.ToLower(p), ".srt") {
						watchList = append([]string{p}, watchList...)
					} else {
						watchList = append(watchList, p)
					}
				}
				if len(watchList) == 0 {
					return
				}

				watched := watchList[0]

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
				statusLabel.SetText(fmt.Sprintf("✅ Completed in %s", time.Since(startTime).Round(time.Second)))
				resultLabel.SetText("✅ Outputs:\n\n" + strings.Join(outputs, "\n"))
			})
		}()
	})
	runBtn.Importance = widget.HighImportance

	config := container.NewVBox(
		title,
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
		container.NewHBox(wantSRT, wantVTT, wantTXT, layout.NewSpacer()),
		widget.NewSeparator(),
		runBtn,
		widget.NewLabel("Result"),
		resultScroll,
	)

	refreshModelOptions()

	return config
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

func createWhisperCommandFile(inputFile, outDir, whisperBin, modelPath, outputPrefix string, wantSRT, wantVTT, wantTXT bool, threads int) (string, error) {
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
