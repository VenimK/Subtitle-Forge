package main

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Helper function to check if a string matches any language name in the map
func containsLanguageName(text string, languages map[string]string) bool {
	for langName := range languages {
		if text == langName {
			return true
		}
	}
	return false
}

// getSystemLanguage tries to detect the system language and returns the best matching language name from the provided map
func getSystemLanguage(availableLangs map[string]string) string {
	// Common environment variables that might contain language info
	envVars := []string{"LANGUAGE", "LANG", "LC_ALL", "LC_MESSAGES"}
	var langCode string

	// Check each environment variable
	for _, envVar := range envVars {
		if lang := os.Getenv(envVar); lang != "" {
			langCode = lang
			break
		}
	}

	// If no language found or it's the default C locale, try macOS-specific detection
	if langCode == "" || langCode == "C" || langCode == "C.UTF-8" {
		// Try to get language from macOS system preferences
		cmd := exec.Command("defaults", "read", "-g", "AppleLanguages")
		if output, err := cmd.Output(); err == nil {
			// Output is like: ("nl-NL", "en-NL", "en")
			if matches := regexp.MustCompile(`"([a-z]{2})(?:-[A-Z]{2})?"`).FindStringSubmatch(string(output)); len(matches) > 1 {
				langCode = matches[1] // Get the language code (e.g., "nl" from "nl-NL")
			}
		}
	}

	if langCode == "" || langCode == "C" || langCode == "C.UTF-8" {
		return "English" // Default fallback
	}

	// Clean up the language code (e.g., "en_US.UTF-8" -> "en")
	langCode = strings.Split(langCode, "_")[0]
	langCode = strings.Split(langCode, ".")[0]

	// Map of common language codes to our available language names
	codeToLang := map[string]string{
		// English variants
		"en": "English",
		// Spanish variants
		"es": "Spanish",
		// French variants
		"fr": "French",
		// German variants
		"de": "German",
		// Italian variants
		"it": "Italian",
		// Portuguese variants
		"pt": "Portuguese",
		// Russian variants
		"ru": "Russian",
		// Japanese variants
		"ja": "Japanese",
		// Korean variants
		"ko": "Korean",
		// Chinese variants
		"zh": "Chinese",
		// Arabic variants
		"ar": "Arabic",
		// Hindi variants
		"hi": "Hindi",
		// Dutch variants
		"nl": "Dutch",
		// Swedish variants
		"sv": "Swedish",
		// Norwegian variants
		"no": "Norwegian",
		"nb": "Norwegian",
		"nn": "Norwegian",
		// Danish variants
		"da": "Danish",
		// Finnish variants
		"fi": "Finnish",
		// Polish variants
		"pl": "Polish",
		// Turkish variants
		"tr": "Turkish",
		// Greek variants
		"el": "Greek",
		// Hungarian variants
		"hu": "Hungarian",
		// Czech variants
		"cs": "Czech",
		// Thai variants
		"th": "Thai",
	}

	// Check if we have a direct match
	if langName, exists := codeToLang[strings.ToLower(langCode)]; exists {
		// Verify the language is in our available languages
		if _, exists := availableLangs[langName]; exists {
			return langName
		}
	}

	// No match found, return default
	return "English"
}
