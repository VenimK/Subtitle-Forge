package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// Supported UI languages
var supportedLanguages = map[string]string{
	"en": "English",
	"nl": "Nederlands",
	"fr": "Français",
	"de": "Deutsch",
	"es": "Español",
}

// SupportedLanguageCodes returns the language codes in display order.
func SupportedLanguageCodes() []string {
	return []string{"en", "nl", "fr", "de", "es"}
}

// SupportedLanguageNames returns "Code - Name" labels for the settings dropdown.
func SupportedLanguageNames() []string {
	codes := SupportedLanguageCodes()
	names := make([]string, len(codes))
	for i, c := range codes {
		names[i] = fmt.Sprintf("%s - %s", strings.ToUpper(c), supportedLanguages[c])
	}
	return names
}

// currentLang holds the active UI language code (default "en").
var currentLang = "en"

// SetLanguage changes the active UI language.
func SetLanguage(code string) {
	code = strings.ToLower(code)
	if _, ok := translations[code]; ok {
		currentLang = code
	}
}

// GetLanguage returns the active UI language code.
func GetLanguage() string {
	return currentLang
}

// T returns the translated string for the given key in the current language.
// If the key is missing it falls back to English, then to the key itself.
func T(key string) string {
	if lang, ok := translations[currentLang]; ok {
		if val, ok := lang[key]; ok {
			return val
		}
	}
	// Fallback to English
	if en, ok := translations["en"]; ok {
		if val, ok := en[key]; ok {
			return val
		}
	}
	return key // last resort: return the key
}

// Tf is like T but with fmt.Sprintf formatting.
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// DetectSystemLanguage returns the 2-letter language code of the OS.
func DetectSystemLanguage() string {
	// macOS: defaults read -g AppleLocale → e.g. "nl_NL"
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err == nil {
		locale := strings.TrimSpace(string(out))
		if len(locale) >= 2 {
			code := strings.ToLower(locale[:2])
			if _, ok := translations[code]; ok {
				return code
			}
		}
	}

	// Fallback: LANG env is checked by Go runtime but let's try LC_ALL / LANG
	// Not needed on macOS typically, so just default to English
	return "en"
}
