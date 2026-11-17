package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
)

// This variable ensures the init function is executed
var _ = startUpdateCheckerAutomatically()

// startUpdateCheckerAutomatically starts the update checker automatically
func startUpdateCheckerAutomatically() bool {
	// Start the update checker in a goroutine
	go func() {
		log.Println("Auto update checker initialized")

		// Wait for the application to fully initialize (5 seconds should be enough)
		time.Sleep(5 * time.Second)

		// Check for updates if there are any windows open. fyne.CurrentApp() is nil
		// in a test binary, where no app is ever started.
		app := fyne.CurrentApp()
		if app == nil || app.Driver() == nil {
			return
		}
		if windows := app.Driver().AllWindows(); len(windows) > 0 {
			CheckForUpdates(windows[0])
		}
	}()

	// Return true to ensure the variable is initialized
	return true
}
