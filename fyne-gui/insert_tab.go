package main

import (
	"image/color"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// InsertTabWidgets holds widgets from the insert tab that are needed by drag-drop handlers
type InsertTabWidgets struct {
	MkvFileLabel      *widget.Label
	SubtitleFileLabel *widget.Label
	MkvDropLabel      *widget.Label
	SubtitleDropLabel *widget.Label
	MkvDropArea       *canvas.Rectangle
	SubtitleDropArea  *canvas.Rectangle
}

// createInsertSubtitlesTab creates the Insert Subtitles tab content and returns
// the container along with widgets needed by the drag-drop handlers.
func createInsertSubtitlesTab(w fyne.Window) (*fyne.Container, *InsertTabWidgets) {
	// Create file selection widgets for subtitle insertion
	insertMkvFileLabel := widget.NewLabel(T("insert.no_video"))
	insertSubtitleFileLabel := widget.NewLabel(T("insert.no_subtitle"))

	selectInsertMkvBtn := widget.NewButton(T("insert.select_video"), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !IsVideoFile(filePath) {
				dialog.ShowInformation(T("common.error"), T("insert.invalid_video"), w)
				return
			}

			insertMkvFileLabel.SetText(filePath)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter(VideoFileExtensions()))
		fd.Show()
	})

	selectInsertSubtitleBtn := widget.NewButton(T("insert.select_subtitle"), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			// Check if it's a supported subtitle format
			if !IsSubtitleFile(filePath) {
				dialog.ShowInformation(T("common.error"), T("insert.invalid_subtitle"), w)
				return
			}

			insertSubtitleFileLabel.SetText(filePath)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt", ".smi", ".mpl", ".tmp"}))
		fd.Show()
	})

	// Create language selection for subtitle insertion
	// Define common languages with their 3-letter ISO codes
	languages := map[string]string{
		"English":    "eng",
		"Spanish":    "spa",
		"French":     "fre",
		"German":     "ger",
		"Italian":    "ita",
		"Japanese":   "jpn",
		"Korean":     "kor",
		"Chinese":    "chi",
		"Russian":    "rus",
		"Portuguese": "por",
		"Arabic":     "ara",
		"Hindi":      "hin",
		"Dutch":      "dut",
		"Swedish":    "swe",
		"Polish":     "pol",
		"Turkish":    "tur",
		"Czech":      "cze",
		"Greek":      "gre",
		"Hungarian":  "hun",
		"Finnish":    "fin",
		"Danish":     "dan",
		"Norwegian":  "nor",
		"Romanian":   "rum",
		"Thai":       "tha",
		"Vietnamese": "vie",
		"Bulgarian":  "bul",
		"Croatian":   "hrv",
		"Slovak":     "slo",
		"Slovenian":  "slv",
		"Ukrainian":  "ukr",
	}

	// Define common language codes for dropdown
	langCodes := []string{
		"eng", "spa", "fre", "ger", "ita", "jpn", "kor", "chi", "rus", "por",
		"ara", "hin", "dut", "swe", "pol", "tur", "cze", "gre", "hun", "fin",
		"dan", "nor", "rum", "tha", "vie", "bul", "hrv", "slo", "slv", "ukr",
		"alb", "amh", "aze", "ben", "bos", "cat", "est", "fil", "glg", "geo",
		"heb", "ice", "ind", "kan", "kaz", "khm", "lao", "lat", "lit",
		"mac", "mal", "mar", "mon", "nep", "per", "srp", "swa", "tam", "tel",
		"tgl", "urd", "uzb", "wel", "yid", "zul",
	}

	// Create sorted list of language names for dropdown
	langNames := make([]string, 0, len(languages))
	for name := range languages {
		langNames = append(langNames, name)
	}
	sort.Strings(langNames)

	// Add "Custom" option at the end
	langNames = append(langNames, "Custom")

	// Detect system language
	systemLang := getSystemLanguage(languages)

	// Create language dropdown
	selectedLang := systemLang
	langDropdown := widget.NewSelect(langNames, func(selected string) {
		selectedLang = selected
	})
	langDropdown.SetSelected(systemLang)

	// Create custom language code dropdown with improved readability
	selectedLangCode := "eng"

	// Create the dropdown with explicit text color
	customLangDropdown := widget.NewSelect(langCodes, func(selected string) {
		selectedLangCode = selected
	})
	customLangDropdown.SetSelected("eng")

	// Create a high-contrast container for the dropdown
	padded := container.NewPadded(customLangDropdown)
	langCodeContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the dropdown directly
		padded,
	)

	// Create a card with the high-contrast container
	langCodeCard := widget.NewCard("", "", langCodeContainer)

	// Initially hide both elements
	customLangDropdown.Hide()
	langCodeCard.Hide()

	// Create track name entry
	trackNameEntry := widget.NewEntry()
	trackNameEntry.SetPlaceHolder(systemLang)
	trackNameEntry.SetText(systemLang)

	// Create result label for subtitle insertion
	insertResultLabel := widget.NewLabel("")
	insertResultScroll := container.NewScroll(insertResultLabel)
	insertResultScroll.SetMinSize(fyne.NewSize(800, 150))

	// Progress bar for insertion operations (infinite/indeterminate)
	insertProgress := widget.NewProgressBarInfinite()
	insertProgress.Hide()
	insertProgressLabel := widget.NewLabel("")
	insertProgressLabel.Hide()

	// Create default track options
	defaultTrack := widget.NewCheck(T("insert.default_track"), nil)
	defaultTrack.SetChecked(true)

	// Create forced track option
	forcedTrack := widget.NewCheck(T("insert.forced_track"), nil)

	// Create option to remove other subtitle tracks
	removeOtherTracks := widget.NewCheck(T("insert.remove_others"), nil)

	// Create output file name options
	outputNameEntry := widget.NewEntry()
	outputNameEntry.SetPlaceHolder(T("insert.auto_naming"))

	// Show language dropdown change handler
	langDropdown.OnChanged = func(selected string) {
		selectedLang = selected
		if selected == "Custom" {
			customLangDropdown.Show()
			langCodeCard.Show()
			// Don't auto-update track name for custom selection
		} else {
			customLangDropdown.Hide()
			langCodeCard.Hide()
			// Automatically select the corresponding language code
			if code, ok := languages[selected]; ok {
				// Find the matching code in langCodes
				for _, langCode := range langCodes {
					if langCode == code {
						customLangDropdown.SetSelected(langCode)
						selectedLangCode = langCode
						break
					}
				}

				// Auto-update track name to match selected language
				if trackNameEntry.Text == "" || trackNameEntry.Text == "English" ||
					containsLanguageName(trackNameEntry.Text, languages) {
					trackNameEntry.SetText(selected)
				}
			}
		}
	}

	// Create insert button (declared first so closures can reference it for Disable/Enable)
	insertSubtitleBtn := widget.NewButton(T("insert.insert_btn"), nil)
	insertSubtitleBtn.OnTapped = func() {
		// Check if files are selected
		mkvPath := insertMkvFileLabel.Text
		subtitlePath := insertSubtitleFileLabel.Text

		if mkvPath == T("insert.no_video") || subtitlePath == T("insert.no_subtitle") {
			dialog.ShowInformation(T("insert.missing_files"), T("insert.select_both"), w)
			return
		}

		// Get language code based on selection
		var lang string
		if selectedLang == "Custom" {
			lang = selectedLangCode // Use the selected language code from dropdown
		} else {
			lang = languages[selectedLang]
		}

		// Get track name
		trackName := trackNameEntry.Text
		if trackName == "" {
			trackName = selectedLang // Use selected language name as default
		}

		// Create output file path
		dir := filepath.Dir(mkvPath)
		baseName := filepath.Base(mkvPath)
		inputExt := filepath.Ext(baseName)
		baseName = strings.TrimSuffix(baseName, inputExt)

		// Use custom output name if provided
		outputName := outputNameEntry.Text
		if outputName == "" {
			outputName = baseName + "_with_subtitles" + inputExt
		} else if !strings.HasSuffix(strings.ToLower(outputName), strings.ToLower(inputExt)) {
			outputName = outputName + inputExt
		}

		outputPath := filepath.Join(dir, outputName)

		insertResultLabel.SetText(T("insert.adding"))
		insertProgress.Show()
		insertProgressLabel.SetText(T("insert.inserting"))
		insertProgressLabel.Show()
		insertSubtitleBtn.Disable()

		if IsMP4File(mkvPath) {
			// Use ffmpeg for MP4/M4V files
			if removeOtherTracks.Checked {
				insertResultLabel.SetText(insertResultLabel.Text + "\n" + T("insert.removing_tracks"))
			}

			go func() {
				output, err := InsertMP4Subtitle(mkvPath, subtitlePath, outputPath, lang, trackName, removeOtherTracks.Checked)

				fyne.Do(func() {
					insertProgress.Hide()
					insertProgressLabel.Hide()
					insertSubtitleBtn.Enable()

					if err != nil {
						AppLog("ERROR", "Insert subtitle into MP4 failed: %v", err)
						insertResultLabel.SetText(insertResultLabel.Text + "\n" + T("common.error") + ": " + err.Error() + "\n" + string(output))
						return
					}

					AppLog("SUCCESS", "Subtitle inserted into MP4 successfully: %s", filepath.Base(outputPath))
					insertResultLabel.SetText(insertResultLabel.Text + "\n✅ " + T("insert.success") + "\n" + T("insert.output_file") + outputPath)
				})
			}()
		} else {
			// Use mkvmerge for MKV files
			mkvmergeArgs := []string{
				"-o", outputPath,
			}

			if removeOtherTracks.Checked {
				mkvmergeArgs = append(mkvmergeArgs, "--no-subtitles", mkvPath)
				insertResultLabel.SetText(insertResultLabel.Text + "\n" + T("insert.removing_tracks"))
			} else {
				mkvmergeArgs = append(mkvmergeArgs, mkvPath)
			}

			mkvmergeArgs = append(mkvmergeArgs,
				"--language", "0:"+lang,
				"--track-name", "0:"+trackName,
			)

			if defaultTrack.Checked {
				mkvmergeArgs = append(mkvmergeArgs, "--default-track", "0:yes")
			}

			if forcedTrack.Checked {
				mkvmergeArgs = append(mkvmergeArgs, "--forced-track", "0:yes")
			}

			mkvmergeArgs = append(mkvmergeArgs, subtitlePath)

			go func() {
				cmd := NewMkvmergeCmd(mkvmergeArgs...)
				AppLog("DEBUG", "Subtitle insertion using mkvmerge")

				AppLog("INSERT", "Adding subtitle to MKV: %s -> %s", filepath.Base(subtitlePath), filepath.Base(outputPath))
				output, err := cmd.CombinedOutput()
				AppLogCmd(cmd, output, err)

				fyne.Do(func() {
					insertProgress.Hide()
					insertProgressLabel.Hide()
					insertSubtitleBtn.Enable()

					if err != nil {
						AppLog("ERROR", "Insert subtitle failed: %v", err)
						insertResultLabel.SetText(insertResultLabel.Text + "\n" + T("common.error") + ": " + err.Error() + "\n" + string(output))
						return
					}

					AppLog("SUCCESS", "Subtitle inserted successfully: %s", filepath.Base(outputPath))
					insertResultLabel.SetText(insertResultLabel.Text + "\n✅ " + T("insert.success") + "\n" + T("insert.output_file") + outputPath + "\n" + string(output))
				})
			}()
		}
	}

	// Create layout for subtitle insertion tab
	insertTitleLabel := widget.NewLabelWithStyle(T("insert.title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create visual drop areas (these are just for visual indication, actual drop handling is at window level)
	mkvDropArea := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	mkvDropLabel := widget.NewLabelWithStyle(T("insert.drop_video"), fyne.TextAlignCenter, fyne.TextStyle{})
	mkvDropContainer := container.NewStack(
		mkvDropArea,
		mkvDropLabel,
	)
	mkvDropContainer.Resize(fyne.NewSize(300, 60))

	subtitleDropArea := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	subtitleDropLabel := widget.NewLabelWithStyle(T("insert.drop_subtitle"), fyne.TextAlignCenter, fyne.TextStyle{})
	subtitleDropContainer := container.NewStack(
		subtitleDropArea,
		subtitleDropLabel,
	)
	subtitleDropContainer.Resize(fyne.NewSize(300, 60))

	// Group file selection
	fileSelectionGroup := widget.NewCard(T("insert.file_selection"), "", container.NewVBox(
		container.NewHBox(selectInsertMkvBtn, insertMkvFileLabel),
		mkvDropContainer,
		container.NewHBox(selectInsertSubtitleBtn, insertSubtitleFileLabel),
		subtitleDropContainer,
	))

	// Group subtitle options
	// Set placeholders with extra spaces to make input fields wider
	trackNameEntry.SetPlaceHolder("Enter track name...                                                ")

	// Create section titles with explicit styling for guaranteed readability
	languageTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText(T("insert.language_settings"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	languageTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	languageTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	trackOptionsTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText(T("insert.track_options"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	trackOptionsTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	trackOptionsTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	// Create separator for visual distinction
	languageSeparator := widget.NewSeparator()
	trackOptionsSeparator := widget.NewSeparator()

	// Create form layout for better alignment of labels and inputs with enhanced readability
	// Create labels with explicit styling for guaranteed readability
	languageLabelContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText(T("insert.language"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	languageLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	langCodeLabelContainer := container.NewMax(
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		canvas.NewText(T("insert.language_code"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	langCodeLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	trackNameLabelContainer := container.NewMax(
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		canvas.NewText(T("insert.track_name"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	trackNameLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	// Create a high-contrast container for the track name entry
	trackNamePadded := container.NewPadded(trackNameEntry)
	trackNameContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the entry directly
		trackNamePadded,
	)

	// Create a card with the high-contrast container
	trackNameCard := widget.NewCard("", "", trackNameContainer)

	// Create form layout with high-contrast labels
	languageForm := container.New(layout.NewFormLayout(),
		languageLabelContainer,
		langDropdown,
		langCodeLabelContainer,
		langCodeCard, // Using the card we created earlier
		trackNameLabelContainer,
		trackNameCard,
	)

	// Create a container for the language section with title, separator, and form
	languageSection := container.NewVBox(
		languageTitleContainer,
		languageSeparator,
		languageForm,
	)

	// Group track options with separator for visual distinction
	trackOptionsContainer := container.NewVBox(
		trackOptionsTitleContainer,
		trackOptionsSeparator,
		defaultTrack,
		forcedTrack,
		removeOtherTracks,
	)

	// Group subtitle options with improved organization and readability
	subtitleOptionsGroup := widget.NewCard(T("insert.subtitle_options"), "", container.NewVBox(
		container.NewPadded(languageSection),       // Using our new language section with title and separator
		container.NewPadded(trackOptionsContainer), // Track options already include title and separator
	))

	// Group output options
	// Make output filename entry wider with placeholder
	outputNameEntry.SetPlaceHolder("Enter output filename (leave empty to use original filename)...                                         ")

	// Create output title with explicit styling for guaranteed readability
	outputTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText(T("insert.output_config"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	outputTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	outputTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	// Create a separator for visual distinction
	outputSeparator := widget.NewSeparator()

	// Create output filename label with explicit styling for guaranteed readability
	outputFilenameLabelContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText(T("insert.output_filename"), color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	outputFilenameLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true

	// Create a high-contrast container for the output filename entry
	outputNamePadded := container.NewPadded(outputNameEntry)
	outputNameContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the entry directly
		outputNamePadded,
	)

	// Create a card with the high-contrast container
	outputNameCard := widget.NewCard("", "", outputNameContainer)

	// Create form layout for better alignment with high-contrast labels
	outputForm := container.New(layout.NewFormLayout(),
		outputFilenameLabelContainer,
		outputNameCard,
	)

	// Create a container for the output section with title and separator
	outputSection := container.NewVBox(
		outputTitleContainer,
		outputSeparator,
		outputForm,
	)

	// Add helpful text
	helpText := widget.NewRichText(
		&widget.TextSegment{Text: T("insert.output_note"), Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
		&widget.TextSegment{Text: T("insert.output_note_text")},
	)
	helpText.Wrapping = fyne.TextWrapWord

	// Style the insert button
	insertSubtitleBtn.Importance = widget.HighImportance

	// Group output options with improved organization and readability
	outputOptionsGroup := widget.NewCard(T("insert.output_options"), "", container.NewVBox(
		container.NewPadded(outputSection), // Using our new output section with title and separator
		container.NewPadded(helpText),
		container.NewHBox(layout.NewSpacer(), insertSubtitleBtn, layout.NewSpacer()),
	))

	// Results group with progress bar
	resultsGroup := widget.NewCard(T("common.results"), "", container.NewVBox(
		insertProgressLabel,
		insertProgress,
		insertResultScroll,
	))

	// Create layout for subtitle insertion tab
	insertTabContent := container.NewVBox(
		insertTitleLabel,
		fileSelectionGroup,
		subtitleOptionsGroup,
		outputOptionsGroup,
		resultsGroup,
	)

	widgets := &InsertTabWidgets{
		MkvFileLabel:      insertMkvFileLabel,
		SubtitleFileLabel: insertSubtitleFileLabel,
		MkvDropLabel:      mkvDropLabel,
		SubtitleDropLabel: subtitleDropLabel,
		MkvDropArea:       mkvDropArea,
		SubtitleDropArea:  subtitleDropArea,
	}

	return insertTabContent, widgets
}
