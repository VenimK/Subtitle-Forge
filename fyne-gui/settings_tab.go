package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createSettingsTab creates the Settings tab content.
// It requires the app, window, and the dependencyButtons container from the extract tab.
func createSettingsTab(a fyne.App, w fyne.Window, dependencyButtons *fyne.Container) *fyne.Container {
	// Create a title with bold styling
	settingsTitle := canvas.NewText(T("settings.title"), color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	settingsTitle.TextSize = 18
	settingsTitle.TextStyle.Bold = true

	// Create a header for dependencies section
	dependencyTitle := widget.NewLabelWithStyle(T("settings.dependencies"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Create a placeholder for the dynamic dependency status updates
	settingsLabel := widget.NewLabel(T("settings.checking"))
	settingsLabel.Wrapping = fyne.TextWrapWord

	// Create a card for theme settings
	themeTitle := canvas.NewText(T("settings.theme_settings"), color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	themeTitle.TextSize = 16
	themeTitle.TextStyle.Bold = true

	// Theme selector with styled label
	themeOptions := []string{"System Default", "Light Theme", "Dark Theme", "Blue Theme", "Warm Theme", "Green Theme", "Spring Theme", "Summer Theme", "Autumn Theme", "Winter Theme"}
	themeSelector := widget.NewSelect(themeOptions, func(selected string) {
		// Save the theme preference
		a.Preferences().SetString("theme", selected)
		ApplyThemeByName(a, selected)
	})

	// Load saved theme preference or default to Dark Theme
	selectedTheme := a.Preferences().StringWithFallback("theme", "Dark Theme")
	themeSelector.SetSelected(selectedTheme)

	// Create a styled theme label with custom color
	themeLabel := widget.NewLabelWithStyle(T("settings.theme_label"), fyne.TextAlignLeading, fyne.TextStyle{
		Bold:      true,
		Italic:    false,
		Monospace: false,
	})

	// Create a colored rectangle background for the label
	labelRect := canvas.NewRectangle(color.NRGBA{R: 40, G: 40, B: 80, A: 255})
	labelContainer := container.NewStack(labelRect, container.NewPadded(themeLabel))

	// Note: Standard labels don't support direct color setting
	// Instead, we're using a colored background with the default text color

	// Create a button to apply theme changes with custom styling and color
	applyThemeBtn := widget.NewButtonWithIcon(T("settings.apply_theme"), theme.ConfirmIcon(), func() {
		selected := themeSelector.Selected
		a.Preferences().SetString("theme", selected)
		ApplyThemeByName(a, selected)
		dialog.ShowInformation(T("settings.theme_applied"), T("settings.theme_applied_msg"), w)
	})
	applyThemeBtn.Importance = widget.HighImportance

	// Create a custom colored apply button
	applyBtnBackground := canvas.NewRectangle(color.NRGBA{R: 0, G: 120, B: 80, A: 255})
	applyBtnContainer := container.NewStack(applyBtnBackground, container.NewPadded(applyThemeBtn))

	// No theme customization option - using predefined themes only

	// Help section
	helpTitle := canvas.NewText(T("settings.help_info"), color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	helpTitle.TextSize = 16
	helpTitle.TextStyle.Bold = true

	// App information
	versionInfo := widget.NewRichText(
		&widget.TextSegment{Text: "Subtitle Forge " + AppVersion + "\n", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
		&widget.TextSegment{Text: T("settings.app_desc") + "\n\n"},
		&widget.TextSegment{Text: " 2025 VenimK@David Software\n", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Italic: true}}},
	)
	versionInfo.Wrapping = fyne.TextWrapWord

	// Add a helpful description of dependencies
	dependencyDescription := widget.NewLabel(T("settings.dep_desc"))
	dependencyDescription.Wrapping = fyne.TextWrapWord

	// Create a list of dependencies with descriptions
	dependencyList := widget.NewLabel(T("settings.dep_list"))
	dependencyList.Wrapping = fyne.TextWrapWord

	// Instructions for missing dependencies
	dependencyInstructions := widget.NewLabel(T("settings.dep_instructions"))
	dependencyInstructions.Wrapping = fyne.TextWrapWord
	dependencyInstructions.TextStyle = fyne.TextStyle{Italic: true}

	// UI Language selector
	langTitle := canvas.NewText(T("settings.ui_language"), color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	langTitle.TextSize = 16
	langTitle.TextStyle.Bold = true

	langNames := SupportedLanguageNames()
	langCodes := SupportedLanguageCodes()
	langSelector := widget.NewSelect(langNames, func(selected string) {
		// Extract code from "EN - English" format
		for i, name := range langNames {
			if name == selected {
				code := langCodes[i]
				SetLanguage(code)
				a.Preferences().SetString("ui_language", code)
				dialog.ShowInformation(T("settings.language_restart"), T("settings.language_restart_msg"), w)
				break
			}
		}
	})
	// Set current language in dropdown
	for i, code := range langCodes {
		if code == GetLanguage() {
			langSelector.SetSelected(langNames[i])
			break
		}
	}

	langSection := container.NewVBox(
		container.NewPadded(langTitle),
		container.NewPadded(container.New(layout.NewFormLayout(),
			widget.NewLabelWithStyle(T("settings.ui_language"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			langSelector,
		)),
	)

	// Create a section for PGS to SRT script configuration
	pgsToSrtTitle := canvas.NewText(T("settings.pgs_config"), color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	pgsToSrtTitle.TextSize = 16
	pgsToSrtTitle.TextStyle.Bold = true

	// Create a label to display the current script path
	pgsToSrtPathLabel := widget.NewLabel(pgsToSrtScriptPath)
	pgsToSrtPathLabel.Wrapping = fyne.TextWrapWord

	// Create a button to browse for the script file
	pgsToSrtBrowseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		// Create a file open dialog
		dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}

			// Update the script path
			pgsToSrtScriptPath = reader.URI().Path()
			pgsToSrtPathLabel.SetText(pgsToSrtScriptPath)
			reader.Close()

			// Run the dependency check again to update status
			updateDependencyStatus(w)
		}, w)

		// Set filters for JavaScript files
		dlg.SetFilter(storage.NewExtensionFileFilter([]string{".js"}))
		dlg.Show()
	})

	// Create a form layout for the PGS to SRT script configuration
	pgsToSrtForm := container.New(layout.NewFormLayout(),
		widget.NewLabel(T("settings.script_path")),
		container.NewBorder(nil, nil, nil, pgsToSrtBrowseBtn, pgsToSrtPathLabel),
	)

	// Add a description for the PGS to SRT script
	pgsToSrtDescription := widget.NewLabel(T("settings.pgs_desc"))
	pgsToSrtDescription.Wrapping = fyne.TextWrapWord

	// Combine all dependency components
	dependencySection := container.NewVBox(
		dependencyTitle,
		container.NewPadded(dependencyDescription),
		container.NewPadded(dependencyList),
		container.NewPadded(settingsLabel),
		container.NewPadded(dependencyInstructions),
		container.NewPadded(dependencyButtons),
		canvas.NewLine(color.NRGBA{R: 200, G: 200, B: 200, A: 128}),
		container.NewPadded(pgsToSrtTitle),
		container.NewPadded(pgsToSrtDescription),
		container.NewPadded(pgsToSrtForm),
	)

	// Custom themed button for resetting to default settings
	resetSettingsBtn := widget.NewButtonWithIcon(T("settings.reset_defaults"), theme.ViewRefreshIcon(), func() {
		// Reset theme to dark theme
		a.Settings().SetTheme(theme.DarkTheme())
		themeSelector.SetSelected("Dark Theme")
		dialog.ShowInformation(T("settings.settings_reset"), T("settings.settings_reset_msg"), w)
	})

	// Style the reset button
	resetSettingsBtn.Importance = widget.MediumImportance

	// Create a custom colored reset button
	resetBtnBackground := canvas.NewRectangle(color.NRGBA{R: 120, G: 60, B: 0, A: 255})
	resetBtnContainer := container.NewStack(resetBtnBackground, container.NewPadded(resetSettingsBtn))

	// Create a styled container for theme buttons
	themeButtonsContainer := container.NewHBox(
		applyBtnContainer,
		layout.NewSpacer(),
		resetBtnContainer,
	)

	// Create info label with custom color styling
	themeInfoLabel := widget.NewRichTextWithText(T("settings.theme_info"))
	themeInfoLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		TextStyle: fyne.TextStyle{Italic: true},
		ColorName: theme.ColorNameForeground,
	}

	// Create a colored background for the info text
	infoBackground := canvas.NewRectangle(color.NRGBA{R: 40, G: 40, B: 60, A: 255})
	infoContainer := container.NewStack(infoBackground, container.NewPadded(themeInfoLabel))

	// Set a custom color for the info text - using ColorName from theme
	// Note: RichTextStyle doesn't have a direct Color field
	themeInfoLabel.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNamePrimary

	// Assemble theme section with styled and colored components
	themeSection := container.NewVBox(
		container.NewPadded(themeTitle),
		container.NewPadded(container.New(layout.NewFormLayout(),
			labelContainer,
			themeSelector,
		)),
		container.NewPadded(infoContainer),
		container.NewPadded(themeButtonsContainer),
	)

	helpSection := container.NewVBox(
		container.NewPadded(helpTitle),
		container.NewPadded(versionInfo),
	)

	// Create cards for each section
	dependencyCard := widget.NewCard("", "", dependencySection)
	themeCard := widget.NewCard("", "", themeSection)
	langCard := widget.NewCard("", "", langSection)
	helpCard := widget.NewCard("", "", helpSection)

	// Assemble settings tab content
	settingsTabContent := container.NewVBox(
		container.NewPadded(settingsTitle),
		dependencyCard,
		themeCard,
		langCard,
		helpCard,
	)

	return settingsTabContent
}
