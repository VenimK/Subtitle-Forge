package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ThemeCustomizer provides a UI for customizing theme colors
type ThemeCustomizer struct {
	window         fyne.Window
	app            fyne.App
	currentPrefs   UserThemePreferences
	colorPickers   map[string]*ColorPicker
	previewSamples map[string]*canvas.Rectangle
	useCustom      *widget.Check
}

// NewThemeCustomizer creates a new theme customizer
func NewThemeCustomizer(app fyne.App, window fyne.Window) *ThemeCustomizer {
	// Load current preferences or use defaults
	prefs, err := LoadThemePreferences()
	if err != nil {
		prefs = DefaultUserThemePreferences()
	}

	return &ThemeCustomizer{
		window:       window,
		app:          app,
		currentPrefs: prefs,
		colorPickers: make(map[string]*ColorPicker),
		previewSamples: make(map[string]*canvas.Rectangle),
	}
}

// Show displays the theme customizer dialog
func (tc *ThemeCustomizer) Show() {
	// Create the main container
	content := container.NewVBox()

	// Add title
	title := widget.NewLabelWithStyle("Theme Color Customization", 
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	content.Add(title)

	// Add option to use custom colors or default theme
	tc.useCustom = widget.NewCheck("Use Custom Colors", func(checked bool) {
		// Update the current theme to use or not use custom colors
		if customTheme, ok := tc.app.Settings().Theme().(*CustomTheme); ok {
			customTheme.SetUseCustomColors(checked)
			tc.app.Settings().SetTheme(customTheme)
		}
	})
	tc.useCustom.SetChecked(true)
	content.Add(tc.useCustom)

	// Create tabs for different color categories
	tabs := container.NewAppTabs()

	// Main colors tab
	mainColorsTab := tc.createMainColorsTab()
	tabs.Append(container.NewTabItem("Main Colors", mainColorsTab))

	// Text colors tab
	textColorsTab := tc.createTextColorsTab()
	tabs.Append(container.NewTabItem("Text Colors", textColorsTab))

	// Button colors tab
	buttonColorsTab := tc.createButtonColorsTab()
	tabs.Append(container.NewTabItem("Button Colors", buttonColorsTab))

	// Add tabs to content
	content.Add(tabs)

	// Create preview section
	previewContainer := tc.createPreviewSection()
	content.Add(previewContainer)

	// Create buttons container
	buttonsContainer := tc.createButtonsContainer()
	content.Add(buttonsContainer)

	// Create and show a proper window (not a dialog) to ensure resizability
	customWindow := fyne.CurrentApp().NewWindow("Theme Customization")
	
	// Create a scrollable container to ensure content is accessible even when window is small
	scrollContainer := container.NewScroll(content)
	
	// Set proper content with padding to ensure all elements are visible
	customWindow.SetContent(container.NewPadded(scrollContainer))
	
	// Set a good initial size
	customWindow.Resize(fyne.NewSize(900, 750))
	
	// Explicitly ensure the window is resizable (should be default, but let's be explicit)
	customWindow.SetFixedSize(false)
	
	// Note: Windows in Fyne don't have a SetMinSize method, but the scrollable container
	// will ensure content is accessible regardless of window size
	
	// Center the window on screen for better user experience
	customWindow.CenterOnScreen()
	
	// Show the window - this must be the last call
	customWindow.Show()
}

// createMainColorsTab creates the tab for main theme colors
func (tc *ThemeCustomizer) createMainColorsTab() fyne.CanvasObject {
	container := container.NewVBox()

	// Primary color
	primaryPicker := NewColorPicker(tc.currentPrefs.PrimaryColor, func(c color.NRGBA) {
		tc.currentPrefs.PrimaryColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["primary"] = primaryPicker
	container.Add(widget.NewCard("Primary Color", "Main accent color used throughout the app", primaryPicker))

	// Background color
	backgroundPicker := NewColorPicker(tc.currentPrefs.BackgroundColor, func(c color.NRGBA) {
		tc.currentPrefs.BackgroundColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["background"] = backgroundPicker
	container.Add(widget.NewCard("Background Color", "Main background color for the app", backgroundPicker))

	// Foreground color
	foregroundPicker := NewColorPicker(tc.currentPrefs.ForegroundColor, func(c color.NRGBA) {
		tc.currentPrefs.ForegroundColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["foreground"] = foregroundPicker
	container.Add(widget.NewCard("Foreground Color", "Main text and icon color", foregroundPicker))

	return container
}

// createTextColorsTab creates the tab for text-related colors
func (tc *ThemeCustomizer) createTextColorsTab() fyne.CanvasObject {
	container := container.NewVBox()

	// Title color
	titlePicker := NewColorPicker(tc.currentPrefs.TitleColor, func(c color.NRGBA) {
		tc.currentPrefs.TitleColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["title"] = titlePicker
	container.Add(widget.NewCard("Title Color", "Color for section titles", titlePicker))

	// Placeholder color
	placeholderPicker := NewColorPicker(tc.currentPrefs.PlaceholderColor, func(c color.NRGBA) {
		tc.currentPrefs.PlaceholderColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["placeholder"] = placeholderPicker
	container.Add(widget.NewCard("Placeholder Color", "Color for placeholder text in input fields", placeholderPicker))

	// Input background color
	inputBgPicker := NewColorPicker(tc.currentPrefs.InputBgColor, func(c color.NRGBA) {
		tc.currentPrefs.InputBgColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["inputBg"] = inputBgPicker
	container.Add(widget.NewCard("Input Background", "Background color for input fields", inputBgPicker))

	return container
}

// createButtonColorsTab creates the tab for button-related colors
func (tc *ThemeCustomizer) createButtonColorsTab() fyne.CanvasObject {
	container := container.NewVBox()

	// Button color
	buttonPicker := NewColorPicker(tc.currentPrefs.ButtonColor, func(c color.NRGBA) {
		tc.currentPrefs.ButtonColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["button"] = buttonPicker
	container.Add(widget.NewCard("Button Color", "Default color for buttons", buttonPicker))

	// Hover color
	hoverPicker := NewColorPicker(tc.currentPrefs.HoverColor, func(c color.NRGBA) {
		tc.currentPrefs.HoverColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["hover"] = hoverPicker
	container.Add(widget.NewCard("Hover Color", "Color when hovering over buttons", hoverPicker))

	// Pressed color
	pressedPicker := NewColorPicker(tc.currentPrefs.PressedColor, func(c color.NRGBA) {
		tc.currentPrefs.PressedColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["pressed"] = pressedPicker
	container.Add(widget.NewCard("Pressed Color", "Color when buttons are pressed", pressedPicker))

	// Disabled color
	disabledPicker := NewColorPicker(tc.currentPrefs.DisabledColor, func(c color.NRGBA) {
		tc.currentPrefs.DisabledColor = c
		tc.updatePreviewSamples()
	})
	tc.colorPickers["disabled"] = disabledPicker
	container.Add(widget.NewCard("Disabled Color", "Color for disabled elements", disabledPicker))

	return container
}

// createButtonsContainer creates the container with action buttons
func (tc *ThemeCustomizer) createButtonsContainer() *fyne.Container {
	// Create buttons with clear labels and icons
	resetBtn := widget.NewButtonWithIcon("Reset to Defaults", theme.ViewRefreshIcon(), func() {
		tc.resetToDefaults()
	})

	applyBtn := widget.NewButtonWithIcon("Apply Changes", theme.ConfirmIcon(), func() {
		tc.applyChanges()
	})

	saveBtn := widget.NewButtonWithIcon("Save as Default", theme.DocumentSaveIcon(), func() {
		tc.saveAsDefault()
	})

	// Set button importance for better visibility
	resetBtn.Importance = widget.MediumImportance
	applyBtn.Importance = widget.HighImportance
	saveBtn.Importance = widget.HighImportance
	
	// Wrap buttons in fixed size containers to ensure they have adequate size
	resetBtnContainer := container.NewPadded(resetBtn)
	applyBtnContainer := container.NewPadded(applyBtn)
	saveBtnContainer := container.NewPadded(saveBtn)

	// Create a vertical container with proper spacing
	// First a spacer to push content down, then buttons in a row, then another spacer
	return container.NewVBox(
		layout.NewSpacer(),
		container.NewHBox(
			layout.NewSpacer(),
			resetBtnContainer,
			layout.NewSpacer(),
			applyBtnContainer,
			layout.NewSpacer(),
			saveBtnContainer,
			layout.NewSpacer(),
		),
		widget.NewSeparator(),
		container.NewPadded(widget.NewLabel("Changes will apply immediately")),
	)
}

// createPreviewSection creates a preview of the theme colors
func (tc *ThemeCustomizer) createPreviewSection() fyne.CanvasObject {
	previewCard := widget.NewCard("Theme Preview", "", nil)

	// Create sample UI elements to show the theme colors
	primarySample := canvas.NewRectangle(tc.currentPrefs.PrimaryColor)
	primarySample.SetMinSize(fyne.NewSize(50, 20))
	tc.previewSamples["primary"] = primarySample

	backgroundSample := canvas.NewRectangle(tc.currentPrefs.BackgroundColor)
	backgroundSample.SetMinSize(fyne.NewSize(50, 20))
	tc.previewSamples["background"] = backgroundSample

	buttonSample := canvas.NewRectangle(tc.currentPrefs.ButtonColor)
	buttonSample.SetMinSize(fyne.NewSize(50, 20))
	tc.previewSamples["button"] = buttonSample

	// Create a grid to display the samples
	samplesGrid := container.New(layout.NewGridLayout(3),
		container.NewVBox(widget.NewLabel("Primary"), primarySample),
		container.NewVBox(widget.NewLabel("Background"), backgroundSample),
		container.NewVBox(widget.NewLabel("Button"), buttonSample),
	)

	// Create a sample button with the current theme
	sampleButton := widget.NewButton("Sample Button", func() {})
	sampleDisabledButton := widget.NewButton("Disabled Button", func() {})
	sampleDisabledButton.Disable()

	// Create a sample entry
	sampleEntry := widget.NewEntry()
	sampleEntry.SetPlaceHolder("Sample placeholder text")

	// Assemble the preview
	previewContent := container.NewVBox(
		widget.NewLabel("Color Samples:"),
		samplesGrid,
		widget.NewLabel("UI Element Samples:"),
		sampleButton,
		sampleDisabledButton,
		sampleEntry,
	)

	previewCard.SetContent(previewContent)
	return previewCard
}

// updatePreviewSamples updates the preview samples with current colors
func (tc *ThemeCustomizer) updatePreviewSamples() {
	for name, sample := range tc.previewSamples {
		switch name {
		case "primary":
			sample.FillColor = tc.currentPrefs.PrimaryColor
		case "background":
			sample.FillColor = tc.currentPrefs.BackgroundColor
		case "button":
			sample.FillColor = tc.currentPrefs.ButtonColor
		}
		sample.Refresh()
	}
}

// resetToDefaults resets all colors to default values
func (tc *ThemeCustomizer) resetToDefaults() {
	// Reset to default preferences
	tc.currentPrefs = DefaultUserThemePreferences()

	// Update all color pickers
	for name, picker := range tc.colorPickers {
		switch name {
		case "primary":
			picker.SetColor(tc.currentPrefs.PrimaryColor)
		case "background":
			picker.SetColor(tc.currentPrefs.BackgroundColor)
		case "foreground":
			picker.SetColor(tc.currentPrefs.ForegroundColor)
		case "button":
			picker.SetColor(tc.currentPrefs.ButtonColor)
		case "hover":
			picker.SetColor(tc.currentPrefs.HoverColor)
		case "pressed":
			picker.SetColor(tc.currentPrefs.PressedColor)
		case "disabled":
			picker.SetColor(tc.currentPrefs.DisabledColor)
		case "title":
			picker.SetColor(tc.currentPrefs.TitleColor)
		case "placeholder":
			picker.SetColor(tc.currentPrefs.PlaceholderColor)
		case "inputBg":
			picker.SetColor(tc.currentPrefs.InputBgColor)
		}
	}

	// Update preview samples
	tc.updatePreviewSamples()
}

// applyChanges applies the current color settings to the app theme
func (tc *ThemeCustomizer) applyChanges() {
	// Apply the current preferences to the theme
	if customTheme, ok := tc.app.Settings().Theme().(*CustomTheme); ok {
		customTheme.SetUserPreferences(tc.currentPrefs)
		tc.app.Settings().SetTheme(customTheme)
	} else {
		// Create a new custom theme with the current preferences
		tc.app.Settings().SetTheme(NewCustomThemeWithPrefs(tc.currentPrefs, tc.useCustom.Checked))
	}

	dialog.ShowInformation("Theme Updated", "The theme colors have been updated.", tc.window)
}

// saveAsDefault saves the current color settings as the default theme
func (tc *ThemeCustomizer) saveAsDefault() {
	// Save the current preferences to file
	err := SaveThemePreferences(tc.currentPrefs)
	if err != nil {
		dialog.ShowError(err, tc.window)
		return
	}

	dialog.ShowInformation("Theme Saved", "The current theme has been saved as the default.", tc.window)
}
