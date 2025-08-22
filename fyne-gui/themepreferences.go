package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// ThemeType represents a predefined theme type
type ThemeType string

// Predefined theme types
const (
	ThemeTypeLight   ThemeType = "light"
	ThemeTypeDark    ThemeType = "dark"
	ThemeTypeBlue    ThemeType = "blue"
	ThemeTypeWarm    ThemeType = "warm"
	ThemeTypeGreen   ThemeType = "green"
	ThemeTypeSpring  ThemeType = "spring"
	ThemeTypeSummer  ThemeType = "summer"
	ThemeTypeAutumn  ThemeType = "autumn"
	ThemeTypeWinter  ThemeType = "winter"
)

// UserThemePreferences stores the user's theme color preferences
type UserThemePreferences struct {
	ThemeType        ThemeType   `json:"themeType"`
	PrimaryColor     color.NRGBA `json:"primaryColor"`
	ButtonColor      color.NRGBA `json:"buttonColor"`
	BackgroundColor  color.NRGBA `json:"backgroundColor"`
	ForegroundColor  color.NRGBA `json:"foregroundColor"`
	HoverColor       color.NRGBA `json:"hoverColor"`
	InputBgColor     color.NRGBA `json:"inputBgColor"`
	PlaceholderColor color.NRGBA `json:"placeholderColor"`
	PressedColor     color.NRGBA `json:"pressedColor"`
	ScrollBarColor   color.NRGBA `json:"scrollBarColor"`
	ShadowColor      color.NRGBA `json:"shadowColor"`
	DisabledColor    color.NRGBA `json:"disabledColor"`
	TitleColor       color.NRGBA `json:"titleColor"`
	LastModified     time.Time   `json:"lastModified"`
}

// DefaultUserThemePreferences returns the default theme preferences (Light theme)
func DefaultUserThemePreferences() UserThemePreferences {
	return LightTheme()
}

// LightTheme returns a light-colored theme with good readability
func LightTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeLight,
		PrimaryColor:     color.NRGBA{R: 0, G: 100, B: 150, A: 255},   // Blue-ish
		ButtonColor:      color.NRGBA{R: 220, G: 220, B: 240, A: 255}, // Light purple-blue
		BackgroundColor:  color.NRGBA{R: 240, G: 240, B: 245, A: 255}, // Very light blue-gray
		ForegroundColor:  color.NRGBA{R: 40, G: 40, B: 50, A: 255},    // Dark blue-gray
		HoverColor:       color.NRGBA{R: 0, G: 120, B: 180, A: 30},    // Translucent blue
		InputBgColor:     color.NRGBA{R: 230, G: 230, B: 240, A: 255}, // Very light purple
		PlaceholderColor: color.NRGBA{R: 150, G: 150, B: 170, A: 255}, // Muted blue-gray
		PressedColor:     color.NRGBA{R: 0, G: 80, B: 120, A: 255},    // Darker blue
		ScrollBarColor:   color.NRGBA{R: 200, G: 200, B: 220, A: 255}, // Light purple-blue
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 40},        // Semi-transparent
		DisabledColor:    color.NRGBA{R: 180, G: 180, B: 180, A: 128}, // Gray with transparency
		TitleColor:       color.NRGBA{R: 0, G: 0, B: 180, A: 255},     // Deep blue
		LastModified:     time.Now(),
	}
}

// DarkTheme returns a dark-colored theme with good readability
func DarkTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeDark,
		PrimaryColor:     color.NRGBA{R: 65, G: 160, B: 220, A: 255},  // Bright blue
		ButtonColor:      color.NRGBA{R: 60, G: 60, B: 70, A: 255},    // Dark gray-blue
		BackgroundColor:  color.NRGBA{R: 30, G: 30, B: 35, A: 255},    // Very dark blue-gray
		ForegroundColor:  color.NRGBA{R: 220, G: 220, B: 230, A: 255}, // Light gray-blue
		HoverColor:       color.NRGBA{R: 70, G: 170, B: 230, A: 30},   // Translucent bright blue
		InputBgColor:     color.NRGBA{R: 45, G: 45, B: 50, A: 255},    // Dark gray
		PlaceholderColor: color.NRGBA{R: 130, G: 130, B: 140, A: 255}, // Medium gray
		PressedColor:     color.NRGBA{R: 40, G: 130, B: 190, A: 255},  // Darker blue
		ScrollBarColor:   color.NRGBA{R: 60, G: 60, B: 70, A: 255},    // Dark gray-blue
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},        // Darker shadow
		DisabledColor:    color.NRGBA{R: 80, G: 80, B: 80, A: 128},    // Dark gray with transparency
		TitleColor:       color.NRGBA{R: 100, G: 180, B: 255, A: 255}, // Bright blue
		LastModified:     time.Now(),
	}
}

// BlueCoolTheme returns a cool blue dark theme
func BlueCoolTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeBlue,
		PrimaryColor:     color.NRGBA{R: 70, G: 130, B: 180, A: 255},
		ButtonColor:      color.NRGBA{R: 60, G: 60, B: 70, A: 255},
		BackgroundColor:  color.NRGBA{R: 30, G: 30, B: 35, A: 255},
		ForegroundColor:  color.NRGBA{R: 220, G: 220, B: 230, A: 255},
		HoverColor:       color.NRGBA{R: 100, G: 180, B: 220, A: 30},
		InputBgColor:     color.NRGBA{R: 45, G: 45, B: 50, A: 255},
		PlaceholderColor: color.NRGBA{R: 150, G: 150, B: 160, A: 255},
		PressedColor:     color.NRGBA{R: 60, G: 140, B: 200, A: 255},
		ScrollBarColor:   color.NRGBA{R: 50, G: 50, B: 60, A: 255},
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},
		DisabledColor:    color.NRGBA{R: 80, G: 80, B: 80, A: 128},
		TitleColor:       color.NRGBA{R: 100, G: 150, B: 200, A: 255},
		LastModified:     time.Now(),
	}
}

// WarmToneTheme returns a warm-toned dark theme
func WarmToneTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeWarm,
		PrimaryColor:     color.NRGBA{R: 255, G: 165, B: 0, A: 255},
		ButtonColor:      color.NRGBA{R: 80, G: 50, B: 20, A: 255},
		BackgroundColor:  color.NRGBA{R: 30, G: 20, B: 15, A: 255},
		ForegroundColor:  color.NRGBA{R: 240, G: 220, B: 200, A: 255},
		HoverColor:       color.NRGBA{R: 255, G: 180, B: 50, A: 30},
		InputBgColor:     color.NRGBA{R: 50, G: 40, B: 30, A: 255},
		PlaceholderColor: color.NRGBA{R: 200, G: 180, B: 160, A: 255},
		PressedColor:     color.NRGBA{R: 220, G: 140, B: 20, A: 255},
		ScrollBarColor:   color.NRGBA{R: 60, G: 40, B: 30, A: 255},
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 100},
		DisabledColor:    color.NRGBA{R: 100, G: 80, B: 60, A: 128},
		TitleColor:       color.NRGBA{R: 255, G: 200, B: 50, A: 255},
		LastModified:     time.Now(),
	}
}

// VibrantGreenTheme returns a vibrant green dark theme
func VibrantGreenTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeGreen,
		PrimaryColor:     color.NRGBA{R: 60, G: 179, B: 113, A: 255},
		ButtonColor:      color.NRGBA{R: 30, G: 50, B: 30, A: 255},
		BackgroundColor:  color.NRGBA{R: 10, G: 10, B: 15, A: 255},
		ForegroundColor:  color.NRGBA{R: 220, G: 255, B: 220, A: 255},
		HoverColor:       color.NRGBA{R: 80, G: 200, B: 130, A: 30},
		InputBgColor:     color.NRGBA{R: 20, G: 40, B: 50, A: 255},
		PlaceholderColor: color.NRGBA{R: 150, G: 180, B: 150, A: 255},
		PressedColor:     color.NRGBA{R: 50, G: 150, B: 100, A: 255},
		ScrollBarColor:   color.NRGBA{R: 20, G: 30, B: 40, A: 255},
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},
		DisabledColor:    color.NRGBA{R: 60, G: 60, B: 60, A: 128},
		TitleColor:       color.NRGBA{R: 50, G: 180, B: 80, A: 255},
		LastModified:     time.Now(),
	}
}

// SpringTheme returns a fresh spring-inspired dark theme
func SpringTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeSpring,
		PrimaryColor:     color.NRGBA{R: 120, G: 190, B: 33, A: 255},  // Fresh green
		ButtonColor:      color.NRGBA{R: 50, G: 120, B: 50, A: 255},  // Dark green button
		BackgroundColor:  color.NRGBA{R: 15, G: 30, B: 15, A: 255},   // Very dark green background
		ForegroundColor:  color.NRGBA{R: 180, G: 230, B: 180, A: 255}, // Light green text
		HoverColor:       color.NRGBA{R: 100, G: 160, B: 100, A: 30},  // Medium green hover
		InputBgColor:     color.NRGBA{R: 25, G: 40, B: 25, A: 255},    // Dark green input bg
		PlaceholderColor: color.NRGBA{R: 120, G: 160, B: 120, A: 255}, // Medium green-gray
		PressedColor:     color.NRGBA{R: 80, G: 140, B: 80, A: 255},   // Medium green pressed
		ScrollBarColor:   color.NRGBA{R: 60, G: 100, B: 60, A: 255},   // Dark green scrollbar
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},        // Dark shadow
		DisabledColor:    color.NRGBA{R: 70, G: 90, B: 70, A: 128},    // Dark green disabled
		TitleColor:       color.NRGBA{R: 100, G: 200, B: 50, A: 255},  // Bright green title
		LastModified:     time.Now(),
	}
}

// SummerTheme returns a warm summer-inspired dark theme
func SummerTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeSummer,
		PrimaryColor:     color.NRGBA{R: 255, G: 196, B: 0, A: 255},   // Bright yellow
		ButtonColor:      color.NRGBA{R: 180, G: 100, B: 0, A: 255},   // Dark orange button
		BackgroundColor:  color.NRGBA{R: 40, G: 25, B: 0, A: 255},     // Very dark amber background
		ForegroundColor:  color.NRGBA{R: 255, G: 220, B: 150, A: 255}, // Light amber text
		HoverColor:       color.NRGBA{R: 200, G: 150, B: 50, A: 30},   // Gold hover
		InputBgColor:     color.NRGBA{R: 60, G: 40, B: 10, A: 255},    // Dark amber input bg
		PlaceholderColor: color.NRGBA{R: 180, G: 150, B: 100, A: 255}, // Gold-tinted placeholder
		PressedColor:     color.NRGBA{R: 200, G: 120, B: 0, A: 255},   // Deep orange pressed
		ScrollBarColor:   color.NRGBA{R: 150, G: 100, B: 50, A: 255},  // Medium amber scrollbar
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},        // Dark shadow
		DisabledColor:    color.NRGBA{R: 100, G: 80, B: 50, A: 128},   // Dark amber disabled
		TitleColor:       color.NRGBA{R: 255, G: 180, B: 0, A: 255},   // Bright gold title
		LastModified:     time.Now(),
	}
}

// AutumnTheme returns a warm autumn-inspired dark theme
func AutumnTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeAutumn,
		PrimaryColor:     color.NRGBA{R: 204, G: 85, B: 0, A: 255},   // Burnt orange
		ButtonColor:      color.NRGBA{R: 140, G: 70, B: 20, A: 255},  // Dark sienna
		BackgroundColor:  color.NRGBA{R: 30, G: 20, B: 10, A: 255},   // Very dark brown background
		ForegroundColor:  color.NRGBA{R: 220, G: 180, B: 130, A: 255}, // Light tan text
		HoverColor:       color.NRGBA{R: 180, G: 100, B: 50, A: 30},   // Copper hover
		InputBgColor:     color.NRGBA{R: 50, G: 30, B: 15, A: 255},    // Dark brown input bg
		PlaceholderColor: color.NRGBA{R: 160, G: 120, B: 90, A: 255},  // Medium brown placeholder
		PressedColor:     color.NRGBA{R: 165, G: 42, B: 42, A: 255},   // Brown pressed
		ScrollBarColor:   color.NRGBA{R: 120, G: 80, B: 40, A: 255},   // Medium brown scrollbar
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},        // Dark shadow
		DisabledColor:    color.NRGBA{R: 100, G: 80, B: 60, A: 128},   // Dark tan disabled
		TitleColor:       color.NRGBA{R: 220, G: 100, B: 40, A: 255},  // Bright copper title
		LastModified:     time.Now(),
	}
}

// WinterTheme returns a cool winter-inspired dark theme
func WinterTheme() UserThemePreferences {
	return UserThemePreferences{
		ThemeType:        ThemeTypeWinter,
		PrimaryColor:     color.NRGBA{R: 70, G: 130, B: 180, A: 255},  // Steel blue
		ButtonColor:      color.NRGBA{R: 40, G: 80, B: 120, A: 255},   // Dark steel blue
		BackgroundColor:  color.NRGBA{R: 10, G: 15, B: 30, A: 255},    // Very dark blue background
		ForegroundColor:  color.NRGBA{R: 180, G: 200, B: 230, A: 255}, // Light blue text
		HoverColor:       color.NRGBA{R: 70, G: 100, B: 150, A: 30},   // Medium blue hover
		InputBgColor:     color.NRGBA{R: 20, G: 30, B: 50, A: 255},    // Dark blue input bg
		PlaceholderColor: color.NRGBA{R: 100, G: 120, B: 150, A: 255}, // Medium slate gray placeholder
		PressedColor:     color.NRGBA{R: 60, G: 100, B: 170, A: 255},  // Medium blue pressed
		ScrollBarColor:   color.NRGBA{R: 50, G: 70, B: 100, A: 255},   // Dark blue scrollbar
		ShadowColor:      color.NRGBA{R: 0, G: 0, B: 0, A: 80},        // Dark shadow
		DisabledColor:    color.NRGBA{R: 60, G: 70, B: 90, A: 128},    // Dark blue-gray disabled
		TitleColor:       color.NRGBA{R: 100, G: 150, B: 240, A: 255}, // Bright blue title
		LastModified:     time.Now(),
	}
}

// GetThemeByType returns a theme based on the specified theme type
func GetThemeByType(themeType ThemeType) UserThemePreferences {
	switch themeType {
	case ThemeTypeLight:
		return LightTheme()
	case ThemeTypeDark:
		return DarkTheme()
	case ThemeTypeBlue:
		return BlueCoolTheme()
	case ThemeTypeWarm:
		return WarmToneTheme()
	case ThemeTypeGreen:
		return VibrantGreenTheme()
	case ThemeTypeSpring:
		return SpringTheme()
	case ThemeTypeSummer:
		return SummerTheme()
	case ThemeTypeAutumn:
		return AutumnTheme()
	case ThemeTypeWinter:
		return WinterTheme()
	default:
		// Default to dark theme if unknown
		return DarkTheme()
	}
}

// getThemePreferencesDir returns the directory where theme preferences are stored
func getThemePreferencesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	prefsDir := filepath.Join(homeDir, ".subtitle-forge", "preferences")
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create preferences directory: %w", err)
	}

	return prefsDir, nil
}

// SaveThemePreferences saves the user's theme preferences to a file
func SaveThemePreferences(prefs UserThemePreferences) error {
	prefsDir, err := getThemePreferencesDir()
	if err != nil {
		return err
	}

	// Update the last modified time
	prefs.LastModified = time.Now()

	// Create the preferences file path
	prefsFile := filepath.Join(prefsDir, "theme_preferences.json")

	// Create a backup of the existing file if it exists
	if _, err := os.Stat(prefsFile); err == nil {
		backupFile := filepath.Join(prefsDir, fmt.Sprintf("theme_preferences_backup_%s.json",
			time.Now().Format("20060102_150405")))

		if err := copyFile(prefsFile, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Marshal the preferences to JSON
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Write the preferences to the file
	if err := ioutil.WriteFile(prefsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write preferences file: %w", err)
	}

	return nil
}

// LoadThemePreferences loads the user's theme preferences from a file
func LoadThemePreferences() (UserThemePreferences, error) {
	prefsDir, err := getThemePreferencesDir()
	if err != nil {
		return DefaultUserThemePreferences(), err
	}

	prefsFile := filepath.Join(prefsDir, "theme_preferences.json")

	// Check if the preferences file exists
	if _, err := os.Stat(prefsFile); os.IsNotExist(err) {
		// If the file doesn't exist, return the default preferences
		defaultPrefs := DefaultUserThemePreferences()
		// Save the default preferences for future use
		_ = SaveThemePreferences(defaultPrefs)
		return defaultPrefs, nil
	}

	// Read the preferences file
	data, err := ioutil.ReadFile(prefsFile)
	if err != nil {
		return DefaultUserThemePreferences(), fmt.Errorf("failed to read preferences file: %w", err)
	}

	// Unmarshal the preferences from JSON
	var prefs UserThemePreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return DefaultUserThemePreferences(), fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	return prefs, nil
}

// RestoreThemePreferencesFromBackup restores theme preferences from a backup file
func RestoreThemePreferencesFromBackup(backupFilename string) error {
	prefsDir, err := getThemePreferencesDir()
	if err != nil {
		return err
	}

	backupFile := filepath.Join(prefsDir, backupFilename)
	prefsFile := filepath.Join(prefsDir, "theme_preferences.json")

	// Check if the backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// Copy the backup file to the preferences file
	if err := copyFile(backupFile, prefsFile); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// Note: copyFile function is now defined in main.go

// ListThemeBackups returns a list of available theme preference backups
func ListThemeBackups() ([]string, error) {
	prefsDir, err := getThemePreferencesDir()
	if err != nil {
		return nil, err
	}

	// Read the directory
	files, err := ioutil.ReadDir(prefsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences directory: %w", err)
	}

	// Filter for backup files
	var backups []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" &&
			file.Name() != "theme_preferences.json" {
			backups = append(backups, file.Name())
		}
	}

	return backups, nil
}
