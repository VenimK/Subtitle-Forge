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

		// Check for updates if there are any windows open
		if len(fyne.CurrentApp().Driver().AllWindows()) > 0 {
			w := fyne.CurrentApp().Driver().AllWindows()[0]
			CheckForUpdates(w)
		}
	}()

	// Return true to ensure the variable is initialized
	return true
}
