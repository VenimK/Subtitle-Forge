package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// CustomTheme is a custom theme for the Subtitle Forge application
type CustomTheme struct {
	// Embed the default theme to get default values for unspecified theme elements
	defaultTheme fyne.Theme
}

// NewCustomTheme creates a new instance of our custom theme
func NewCustomTheme() fyne.Theme {
	return &CustomTheme{
		defaultTheme: theme.DefaultTheme(),
	}
}

// Color returns the color for the specified name
func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 100, B: 150, A: 255} // Blue-ish
	case theme.ColorNameBackground:
		return color.NRGBA{R: 240, G: 240, B: 245, A: 255} // Very light blue-gray
	case theme.ColorNameButton:
		return color.NRGBA{R: 220, G: 220, B: 240, A: 255} // Light purple-blue
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 180, G: 180, B: 180, A: 128} // Gray with transparency
	case theme.ColorNameForeground:
		return color.NRGBA{R: 40, G: 40, B: 50, A: 255} // Dark blue-gray, almost black
	case theme.ColorNameHover:
		return color.NRGBA{R: 0, G: 120, B: 180, A: 30} // Translucent blue for hover effect
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 230, G: 230, B: 240, A: 255} // Very light purple for input fields
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 150, G: 150, B: 170, A: 255} // Muted blue-gray for placeholders
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0, G: 80, B: 120, A: 255} // Darker blue for pressed state
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 200, G: 200, B: 220, A: 255} // Light purple-blue for scrollbar
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 40} // Semi-transparent shadow
	}

	return t.defaultTheme.Color(name, variant)
}

// Font returns the font resource for the specified style and size
func (t *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.defaultTheme.Font(style)
}

// Icon returns the icon resource for the specified name
func (t *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.defaultTheme.Icon(name)
}

// Size returns the size for the specified name
func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6 // Slightly more padding
	case theme.SizeNameInlineIcon:
		return 24 // Slightly larger inline icons
	case theme.SizeNameText:
		return 14 // Slightly larger text
	case theme.SizeNameHeadingText:
		return 26 // Larger headings
	}

	return t.defaultTheme.Size(name)
}
