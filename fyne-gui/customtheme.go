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
	// User preferences for theme colors
	userPrefs UserThemePreferences
	// Whether to use custom colors or default theme colors
	useCustomColors bool
}

// NewCustomTheme creates a new instance of our custom theme
func NewCustomTheme() fyne.Theme {
	// Load user preferences
	userPrefs, err := LoadThemePreferences()
	if err != nil {
		// If there's an error loading preferences, use defaults
		userPrefs = DefaultUserThemePreferences()
	}

	return &CustomTheme{
		defaultTheme: theme.DefaultTheme(),
		userPrefs:    userPrefs,
		useCustomColors: true, // Default to using custom colors
	}
}

// NewCustomThemeWithPrefs creates a new instance of our custom theme with specific preferences
func NewCustomThemeWithPrefs(prefs UserThemePreferences, useCustom bool) fyne.Theme {
	return &CustomTheme{
		defaultTheme: theme.DefaultTheme(),
		userPrefs:    prefs,
		useCustomColors: useCustom,
	}
}

// Color returns the color for the specified name
func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// If not using custom colors, delegate to the default theme
	if !t.useCustomColors {
		return t.defaultTheme.Color(name, variant)
	}

	// Otherwise use our custom colors from user preferences
	switch name {
	case theme.ColorNamePrimary:
		return t.userPrefs.PrimaryColor
	case theme.ColorNameBackground:
		return t.userPrefs.BackgroundColor
	case theme.ColorNameButton:
		return t.userPrefs.ButtonColor
	case theme.ColorNameDisabled:
		return t.userPrefs.DisabledColor
	case theme.ColorNameForeground:
		return t.userPrefs.ForegroundColor
	case theme.ColorNameHover:
		return t.userPrefs.HoverColor
	case theme.ColorNameInputBackground:
		return t.userPrefs.InputBgColor
	case theme.ColorNamePlaceHolder:
		return t.userPrefs.PlaceholderColor
	case theme.ColorNamePressed:
		return t.userPrefs.PressedColor
	case theme.ColorNameScrollBar:
		return t.userPrefs.ScrollBarColor
	case theme.ColorNameShadow:
		return t.userPrefs.ShadowColor
	}

	return t.defaultTheme.Color(name, variant)
}

// GetUserPreferences returns the current user preferences
func (t *CustomTheme) GetUserPreferences() UserThemePreferences {
	return t.userPrefs
}

// SetUserPreferences updates the theme with new user preferences
func (t *CustomTheme) SetUserPreferences(prefs UserThemePreferences) {
	t.userPrefs = prefs
}

// SetUseCustomColors sets whether to use custom colors or default theme colors
func (t *CustomTheme) SetUseCustomColors(useCustom bool) {
	t.useCustomColors = useCustom
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
