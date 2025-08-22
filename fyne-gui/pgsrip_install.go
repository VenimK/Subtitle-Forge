package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// installPgsrip attempts to install pgsrip and shows progress in the UI
func installPgsrip(result *widget.Entry, statusLabel *widget.Label) bool {
	// Get the current directory
	baseDir := getCurrentDir()
	scriptPath := filepath.Join(baseDir, "install_pgsrip.sh")

	// Check if the installation script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// If it doesn't exist in the current directory, look for it relative to the executable
		execPath := getExecutablePath()
		execDir := filepath.Dir(execPath)
		scriptPath = filepath.Join(execDir, "install_pgsrip.sh")
		
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			fyne.Do(func() {
				if result != nil {
					result.SetText(result.Text + "\nError: Could not find pgsrip installation script")
				}
				if statusLabel != nil {
					statusLabel.SetText("Error: pgsrip installation script not found")
				}
			})
			return false
		}
	}

	// Make the script executable
	os.Chmod(scriptPath, 0755)

	// Run the installation script
	fmt.Println("[DEBUG] Installing pgsrip with script:", scriptPath)
	
	fyne.Do(func() {
		if result != nil {
			result.SetText(result.Text + "\nInstalling pgsrip...")
		}
		if statusLabel != nil {
			statusLabel.SetText("Installing pgsrip...")
		}
	})

	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Installing pgsrip with script: %s\n", scriptPath)
		if err != nil {
			fmt.Fprintf(debugLogger, "pgsrip installation failed: %v\n", err)
			fmt.Fprintf(debugLogger, "Output: %s\n", string(output))
		} else {
			fmt.Fprintf(debugLogger, "pgsrip installation completed successfully\n")
			fmt.Fprintf(debugLogger, "Output: %s\n", string(output))
		}
	}
	
	// Update UI with the result
	fyne.Do(func() {
		if result != nil {
			result.SetText(result.Text + "\n" + string(output))
		}
		
		if err != nil {
			if statusLabel != nil {
				statusLabel.SetText("pgsrip installation failed")
			}
			if result != nil {
				result.SetText(result.Text + "\nInstallation failed: " + err.Error())
			}
			return
		}
		
		if statusLabel != nil {
			statusLabel.SetText("pgsrip installation completed")
		}
		if result != nil {
			result.SetText(result.Text + "\npgsrip installation completed successfully")
		}
	})

	// Check if pgsrip is now available
	return checkPgsrip()
}
