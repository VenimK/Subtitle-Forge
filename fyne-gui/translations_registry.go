package main

// translations maps language codes to their translation maps.
// Populated at init time from the individual translation files.
var translations = map[string]map[string]string{
	"en": translationsEN,
	"nl": translationsNL,
	"fr": translationsFR,
	"de": translationsDE,
	"es": translationsES,
}
