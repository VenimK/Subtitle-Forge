package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// installPgsripCli attempts to install pgsrip using the installation script from the command line
func installPgsripCli() bool {
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
			fmt.Println("[ERROR] Could not find pgsrip installation script")
			return false
		}
	}

	// Make the script executable
	os.Chmod(scriptPath, 0755)

	// Run the installation script
	fmt.Println("[DEBUG] Installing pgsrip with script:", scriptPath)
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Installing pgsrip with script: %s\n", scriptPath)
	}

	err := cmd.Run()
	success := err == nil
	
	if success {
		fmt.Println("[DEBUG] pgsrip installation script completed successfully")
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "pgsrip installation script completed successfully\n")
		}
	} else {
		fmt.Println("[DEBUG] pgsrip installation failed:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "pgsrip installation failed: %v\n", err)
		}
	}

	// Check if pgsrip is now available
	return checkPgsrip()
}
