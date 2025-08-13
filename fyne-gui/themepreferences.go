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

// UserThemePreferences stores the user's custom color preferences
type UserThemePreferences struct {
	PrimaryColor       color.NRGBA `json:"primaryColor"`
	ButtonColor        color.NRGBA `json:"buttonColor"`
	BackgroundColor    color.NRGBA `json:"backgroundColor"`
	ForegroundColor    color.NRGBA `json:"foregroundColor"`
	HoverColor         color.NRGBA `json:"hoverColor"`
	InputBgColor       color.NRGBA `json:"inputBgColor"`
	PlaceholderColor   color.NRGBA `json:"placeholderColor"`
	PressedColor       color.NRGBA `json:"pressedColor"`
	ScrollBarColor     color.NRGBA `json:"scrollBarColor"`
	ShadowColor        color.NRGBA `json:"shadowColor"`
	DisabledColor      color.NRGBA `json:"disabledColor"`
	TitleColor         color.NRGBA `json:"titleColor"`
	LastModified       time.Time   `json:"lastModified"`
}

// DefaultUserThemePreferences returns the default theme preferences
func DefaultUserThemePreferences() UserThemePreferences {
	return UserThemePreferences{
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
