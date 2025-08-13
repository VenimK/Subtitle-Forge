package main

import (
	"image/color"
	"math"

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
	themeType       ThemeType
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
		defaultTheme:    theme.DefaultTheme(),
		userPrefs:       userPrefs,
		useCustomColors: true, // Default to using custom colors
		themeType:       userPrefs.ThemeType,
	}
}

// NewCustomThemeWithPrefs creates a new instance of our custom theme with specific preferences
func NewCustomThemeWithPrefs(prefs UserThemePreferences, useCustom bool) fyne.Theme {
	return &CustomTheme{
		defaultTheme:    theme.DefaultTheme(),
		userPrefs:       prefs,
		useCustomColors: useCustom,
		themeType:       prefs.ThemeType,
	}
}

// ensureContrast ensures there's enough contrast between text and background colors
// by adjusting the text color if necessary
func ensureContrast(textColor, bgColor color.Color) color.Color {
	// Extract RGBA values
	r1, g1, b1, _ := textColor.RGBA()
	r2, g2, b2, _ := bgColor.RGBA()
	
	// Convert to 8-bit values for easier calculation
	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8
	
	// Calculate relative luminance (simplified formula)
	lum1 := 0.299*float64(r1) + 0.587*float64(g1) + 0.114*float64(b1)
	lum2 := 0.299*float64(r2) + 0.587*float64(g2) + 0.114*float64(b2)
	
	// Calculate contrast ratio (simplified)
	contrast := math.Abs(lum1 - lum2)
	
	// If contrast is too low, adjust the text color
	if contrast < 75 { // Threshold for minimum contrast
		// Invert the color for maximum contrast
		return color.NRGBA{
			R: uint8(255 - r2),
			G: uint8(255 - g2),
			B: uint8(255 - b2),
			A: 255,
		}
	}
	
	return textColor
}

// Color returns the color for the specified name
func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// If not using custom colors, delegate to the default theme
	if !t.useCustomColors {
		return t.defaultTheme.Color(name, variant)
	}

	// Always ensure dialog readability with our predefined themes
	{
		// Special handling for critical UI elements to ensure readability
		if name == theme.ColorNameForeground {
			// For all foreground text, ensure high contrast
			if variant == theme.VariantLight {
				// For text on light backgrounds (like input fields)
				return color.NRGBA{R: 0, G: 0, B: 0, A: 255} // Black text
			} else {
				// For text on dark backgrounds
				return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light text
			}
		}

		// For input backgrounds, always ensure they're readable
		if name == theme.ColorNameInputBackground {
			// Light themes get light input backgrounds
			if t.themeType == ThemeTypeLight {
				return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light background
			} else if t.themeType == ThemeTypeBlue {
				return color.NRGBA{R: 45, G: 45, B: 50, A: 255} // Dark gray for blue theme
			} else if t.themeType == ThemeTypeWarm {
				return color.NRGBA{R: 50, G: 40, B: 30, A: 255} // Warm dark for warm theme
			} else if t.themeType == ThemeTypeGreen {
				return color.NRGBA{R: 20, G: 40, B: 50, A: 255} // Dark green-blue for green theme
			} else if t.themeType == ThemeTypeSpring {
				return color.NRGBA{R: 25, G: 40, B: 25, A: 255} // Dark green input bg
			} else if t.themeType == ThemeTypeSummer {
				return color.NRGBA{R: 60, G: 40, B: 10, A: 255} // Dark amber input bg
			} else if t.themeType == ThemeTypeAutumn {
				return color.NRGBA{R: 50, G: 30, B: 15, A: 255} // Dark brown input bg
			} else if t.themeType == ThemeTypeWinter {
				return color.NRGBA{R: 20, G: 30, B: 50, A: 255} // Dark blue input bg
			} else {
				// Dark theme and other themes get slightly lighter input backgrounds
				return color.NRGBA{R: 45, G: 45, B: 50, A: 255} // Dark gray
			}
		}

		// For placeholder text, ensure good contrast with input background
		if name == theme.ColorNamePlaceHolder {
			// Light themes get darker placeholder text
			switch t.themeType {
			case ThemeTypeLight:
				return color.NRGBA{R: 120, G: 120, B: 120, A: 255} // Medium gray
			case ThemeTypeBlue:
				return color.NRGBA{R: 150, G: 150, B: 160, A: 255} // Blue-tinted gray
			case ThemeTypeWarm:
				return color.NRGBA{R: 200, G: 180, B: 160, A: 255} // Warm-tinted gray
			case ThemeTypeGreen:
				return color.NRGBA{R: 150, G: 180, B: 150, A: 255} // Green-tinted gray
			case ThemeTypeSpring:
				return color.NRGBA{R: 100, G: 150, B: 100, A: 255} // Medium green-gray
			case ThemeTypeSummer:
				return color.NRGBA{R: 180, G: 170, B: 140, A: 255} // Tan placeholder
			case ThemeTypeAutumn:
				return color.NRGBA{R: 160, G: 120, B: 90, A: 255} // Medium brown placeholder
			case ThemeTypeWinter:
				return color.NRGBA{R: 119, G: 136, B: 153, A: 255} // Light slate gray placeholder
			default:
				// Dark themes get lighter placeholder text
				return color.NRGBA{R: 160, G: 160, B: 170, A: 255} // Light gray
			}
		}
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
		// For custom themes, ensure foreground text has good contrast with background
		return ensureContrast(t.userPrefs.ForegroundColor, t.userPrefs.BackgroundColor)
	case theme.ColorNameHover:
		return t.userPrefs.HoverColor
	case theme.ColorNameInputBackground:
		// For custom themes, ensure input backgrounds are readable
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light background for readability
	case theme.ColorNamePlaceHolder:
		// For custom themes, ensure placeholder text has good contrast
		return color.NRGBA{R: 120, G: 120, B: 120, A: 255} // Medium gray
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
