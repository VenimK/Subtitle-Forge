package main

import "fyne.io/fyne/v2"

// AppInfo provides metadata for the application
func AppInfo() fyne.AppMetadata {
	return fyne.AppMetadata{
		ID:      "com.gmm.subtitleforge",
		Name:    "Subtitle Forge",
		Version: AppVersion,
	}
}
