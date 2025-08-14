package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TrackItem represents a subtitle track with UI elements
type TrackItem struct {
	Num        int
	Lang       string
	Codec      string
	Name       string
	State      string
	Check      *widget.Check
	Status     *widget.Label
	ConvertOCR *widget.Check  // Option to convert PGS to SRT using OCR
	LangSelect *widget.Select // Language selection dropdown for OCR
}

// Global debug logger for dependency checks
var debugLogger *os.File

// Helper function to get current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "<error getting working directory>"
	}
	return dir
}

// Helper function to get executable path
func getExecutablePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "<error getting executable path>"
	}
	return exePath
}

// checkDependencies verifies if all required external dependencies are installed
func checkDependencies() map[string]bool {
	// Create a debug log file in the user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		logPath := filepath.Join(homeDir, "subtitle_forge_debug.log")
		logFile, err := os.Create(logPath)
		if err == nil {
			defer logFile.Close()
			fmt.Fprintf(logFile, "=== Subtitle Forge Dependency Check Debug Log ===\n")
			fmt.Fprintf(logFile, "Time: %s\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(logFile, "Working Directory: %s\n", getCurrentDir())
			fmt.Fprintf(logFile, "Executable Path: %s\n", getExecutablePath())
			fmt.Fprintf(logFile, "Environment PATH: %s\n\n", os.Getenv("PATH"))
			
			// Set up a global debug logger that can be used by dependency check functions
			debugLogger = logFile
		}
	}
	dependencyResults := make(map[string]bool)
	dependencyResults["FFmpeg"] = checkFfmpeg()
	dependencyResults["vobsub2srt"] = checkVobsub2srt()
	dependencyResults["MKVMerge"] = checkMkvmerge()
	dependencyResults["MKVExtract"] = checkMkvextract()
	dependencyResults["Deno"] = checkDeno()
	dependencyResults["Tesseract"] = checkTesseract()
	dependencyResults["Go"] = checkGo()
	dependencyResults["PGStoSRT"] = checkPgsToSrt()

	return dependencyResults
}

// Check for ffmpeg installation
func checkFfmpeg() bool {
	ffmpegFound := false
	fmt.Println("[DEBUG] Checking for ffmpeg...")
	
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for FFmpeg ===\n")
	}

	// First try using exec.LookPath to find ffmpeg in PATH
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err == nil {
		fmt.Println("[DEBUG] ffmpeg found in PATH at", ffmpegPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "ffmpeg found in PATH at: %s\n", ffmpegPath)
		}
		ffmpegFound = true
	} else {
		fmt.Println("[DEBUG] ffmpeg not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "ffmpeg not found in PATH: %v\n", err)
		}
		
		// Check common installation paths
		commonPaths := []string{
			"/opt/homebrew/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/usr/bin/ffmpeg",
		}
		
		// Get home directory for user-specific paths
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Add user-specific paths
			userPaths := []string{
				filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg"),
				filepath.Join(homeDir, "anaconda3", "bin", "ffmpeg"),
				filepath.Join(homeDir, "bin", "ffmpeg"),
			}
			commonPaths = append(commonPaths, userPaths...)
		}
		
		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil {
				// Check if executable
				perm := fileInfo.Mode().Perm()
				isExecutable := (perm & 0111) != 0
				
				if isExecutable {
					fmt.Println("[DEBUG] ffmpeg found at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "ffmpeg found at: %s\n", path)
					}
					ffmpegFound = true
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "ffmpeg exists at %s but is not executable\n", path)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !ffmpegFound {
			fmt.Fprintf(debugLogger, "ffmpeg not found in any common paths\n")
		}
	}

	// If still not found, try running the command directly as a last resort
	if !ffmpegFound {
		fmt.Println("[DEBUG] Trying to run ffmpeg command directly")
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "Trying to run ffmpeg command directly\n")
		}
		ffmpegCmd := exec.Command("ffmpeg", "-h")
		output, err := ffmpegCmd.CombinedOutput()
		ffmpegFound = err == nil && strings.Contains(string(output), "usage")
		fmt.Println("[DEBUG] Direct ffmpeg command check result:", ffmpegFound)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "Direct ffmpeg command check result: %v\n", ffmpegFound)
		}
		if err != nil {
			fmt.Println("[DEBUG] Direct ffmpeg command error:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Direct ffmpeg command error: %v\n", err)
			}
		}
	}

	fmt.Println("[DEBUG] Final ffmpeg found status:", ffmpegFound)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final ffmpeg found status: %v\n\n", ffmpegFound)
	}
	return ffmpegFound
}

// Check for vobsub2srt installation
func checkVobsub2srt() bool {
	vobsub2srtFound := false
	fmt.Println("[DEBUG] Checking for vobsub2srt...")

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for vobsub2srt ===\n")
	}

	// First try using exec.LookPath to find vobsub2srt in PATH
	vobsub2srtPath, err := exec.LookPath("vobsub2srt")
	if err == nil {
		fmt.Println("[DEBUG] vobsub2srt found in PATH at", vobsub2srtPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "vobsub2srt found in PATH at: %s\n", vobsub2srtPath)
		}
		vobsub2srtFound = true
	} else {
		fmt.Println("[DEBUG] vobsub2srt not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "vobsub2srt not found in PATH: %v\n", err)
		}
		
		// Check common installation paths
		commonPaths := []string{
			"/usr/local/bin/vobsub2srt",
			"/usr/bin/vobsub2srt",
			"/opt/homebrew/bin/vobsub2srt",
		}
		
		// Get home directory for user-specific paths
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Add user-specific paths
			userPaths := []string{
				filepath.Join(homeDir, "bin", "vobsub2srt"),
			}
			commonPaths = append(commonPaths, userPaths...)
		}
		
		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil {
				// Check if executable
				perm := fileInfo.Mode().Perm()
				isExecutable := (perm & 0111) != 0
				
				if isExecutable {
					fmt.Println("[DEBUG] vobsub2srt found at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "vobsub2srt found at: %s\n", path)
					}
					vobsub2srtFound = true
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "vobsub2srt exists at %s but is not executable\n", path)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !vobsub2srtFound {
			fmt.Fprintf(debugLogger, "vobsub2srt not found in any common paths\n")
		}
	}
	
	// Try standard path using which command
	if !vobsub2srtFound {
		fmt.Println("[DEBUG] Trying to find vobsub2srt in PATH using 'which'")
		whichCmd := exec.Command("which", "vobsub2srt")
		output, err := whichCmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			altPath := strings.TrimSpace(string(output))
			fmt.Println("[DEBUG] Found vobsub2srt at", altPath)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Found vobsub2srt using 'which' at: %s\n", altPath)

				// Check if the file exists and is executable
				info, err := os.Stat(altPath)
				if err == nil {
					// Check if the file is executable (Unix-style permission check)
					perm := info.Mode().Perm()
					isExecutable := (perm & 0111) != 0 // Check if any execute bit is set

					vobsub2srtFound = isExecutable
					fmt.Println("[DEBUG] vobsub2srt executable permission check:", isExecutable)
				}
			}
		}
	}

	fmt.Println("[DEBUG] Final vobsub2srt found status:", vobsub2srtFound)
	return vobsub2srtFound
}

// Check for MKVMerge installation
func checkMkvmerge() bool {
	fmt.Println("[DEBUG] Checking for MKVMerge...")
	mkvmergeFound := false
	
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for MKVMerge ===\n")
	}
	
	// First try using exec.LookPath to find mkvmerge in PATH
	mkvmergePath, err := exec.LookPath("mkvmerge")
	if err == nil {
		fmt.Println("[DEBUG] mkvmerge found in PATH at", mkvmergePath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "mkvmerge found in PATH at: %s\n", mkvmergePath)
		}
		
		// Verify by running the command
		mkvmergeCmd := exec.Command("mkvmerge", "--version")
		mkvmergeOutput, err := mkvmergeCmd.CombinedOutput()
		mkvmergeFound = err == nil && len(mkvmergeOutput) > 0
		
		if mkvmergeFound {
			fmt.Println("[DEBUG] MKVMerge found:", strings.TrimSpace(string(mkvmergeOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVMerge verified with version: %s\n", strings.TrimSpace(string(mkvmergeOutput)))
			}
		} else {
			fmt.Println("[DEBUG] MKVMerge command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVMerge command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] mkvmerge not found in PATH:", err)
		
		// Check common installation paths
		commonPaths := []string{
			"/opt/homebrew/bin/mkvmerge",
			"/usr/local/bin/mkvmerge",
			"/usr/bin/mkvmerge",
			"/Applications/MKVToolNix.app/Contents/MacOS/mkvmerge",
		}
		
		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] mkvmerge found at", path)
				
				// Verify by running the command
				mkvmergeCmd := exec.Command(path, "--version")
				mkvmergeOutput, err := mkvmergeCmd.CombinedOutput()
				if err == nil && len(mkvmergeOutput) > 0 {
					mkvmergeFound = true
					fmt.Println("[DEBUG] MKVMerge verified at", path)
					break
				}
			}
		}
	}

	fmt.Println("[DEBUG] Final MKVMerge found status:", mkvmergeFound)
	return mkvmergeFound
}

// Check for MKVExtract installation
func checkMkvextract() bool {
	fmt.Println("[DEBUG] Checking for MKVExtract...")
	mkvextractFound := false
	
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for MKVExtract ===\n")
	}
	
	// First try using exec.LookPath to find mkvextract in PATH
	mkvextractPath, err := exec.LookPath("mkvextract")
	if err == nil {
		fmt.Println("[DEBUG] mkvextract found in PATH at", mkvextractPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "mkvextract found in PATH at: %s\n", mkvextractPath)
		}
		
		// Verify by running the command
		mkvextractCmd := exec.Command("mkvextract", "--version")
		mkvextractOutput, err := mkvextractCmd.CombinedOutput()
		mkvextractFound = err == nil && len(mkvextractOutput) > 0
		
		if mkvextractFound {
			fmt.Println("[DEBUG] MKVExtract found:", strings.TrimSpace(string(mkvextractOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVExtract verified with version: %s\n", strings.TrimSpace(string(mkvextractOutput)))
			}
		} else {
			fmt.Println("[DEBUG] MKVExtract command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVExtract command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] mkvextract not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "mkvextract not found in PATH: %v\n", err)
		}
		
		// Check common installation paths
		commonPaths := []string{
			"/opt/homebrew/bin/mkvextract",
			"/usr/local/bin/mkvextract",
			"/usr/bin/mkvextract",
			"/Applications/MKVToolNix.app/Contents/MacOS/mkvextract",
		}
		
		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] mkvextract found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "mkvextract found at: %s\n", path)
				}
				
				// Verify by running the command
				mkvextractCmd := exec.Command(path, "--version")
				mkvextractOutput, err := mkvextractCmd.CombinedOutput()
				if err == nil && len(mkvextractOutput) > 0 {
					mkvextractFound = true
					fmt.Println("[DEBUG] MKVExtract verified at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "MKVExtract verified at: %s\n", path)
					}
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "MKVExtract command failed at %s: %v\n", path, err)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !mkvextractFound {
			fmt.Fprintf(debugLogger, "mkvextract not found in any common paths\n")
		}
	}

	fmt.Println("[DEBUG] Final MKVExtract found status:", mkvextractFound)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final MKVExtract found status: %v\n\n", mkvextractFound)
	}
	return mkvextractFound
}

// Check for Deno installation
func checkDeno() bool {
	fmt.Println("[DEBUG] Checking for Deno...")
	denoCmd := exec.Command("deno", "--version")
	denoOutput, err := denoCmd.CombinedOutput()
	denoFound := err == nil && len(denoOutput) > 0

	if denoFound {
		fmt.Println("[DEBUG] Deno found:", strings.TrimSpace(string(denoOutput)))
	} else {
		fmt.Println("[DEBUG] Deno not found or error:", err)
	}

	fmt.Println("[DEBUG] Final Deno found status:", denoFound)
	return denoFound
}

// Check for Tesseract installation
func checkTesseract() bool {
	fmt.Println("[DEBUG] Checking for Tesseract...")
	tesseractCmd := exec.Command("tesseract", "--version")
	tesseractOutput, err := tesseractCmd.CombinedOutput()
	tesseractFound := err == nil && len(tesseractOutput) > 0

	if tesseractFound {
		fmt.Println("[DEBUG] Tesseract found:", strings.TrimSpace(string(tesseractOutput)))
	} else {
		fmt.Println("[DEBUG] Tesseract not found or error:", err)
	}

	fmt.Println("[DEBUG] Final Tesseract found status:", tesseractFound)
	return tesseractFound
}

// PGS to SRT script path - configurable via UI
var pgsToSrtScriptPath = filepath.Join(os.Getenv("HOME"), "pgs-to-srt", "pgs-to-srt.js")

// Check for Go installation
func checkGo() bool {
	fmt.Println("[DEBUG] Checking for Go...")

	goCmd := exec.Command("go", "version")
	goOutput, err := goCmd.CombinedOutput()
	goFound := err == nil && len(goOutput) > 0

	if goFound {
		fmt.Println("[DEBUG] Go found:", strings.TrimSpace(string(goOutput)))
	} else {
		fmt.Println("[DEBUG] Go not found, error:", err)
	}

	fmt.Println("[DEBUG] Final Go found status:", goFound)
	return goFound
}

// Check for PGS to SRT script
func checkPgsToSrt() bool {
	fmt.Println("[DEBUG] Checking for PGS to SRT script...")

	// Use the configurable script path
	scriptPath := pgsToSrtScriptPath

	// Check if the script exists at the specified path
	_, err := os.Stat(scriptPath)
	scriptFound := err == nil

	if scriptFound {
		fmt.Println("[DEBUG] PGS to SRT script found at:", scriptPath)

		// Additionally check if Deno is available to run the script
		denoAvailable := checkDeno()
		if !denoAvailable {
			fmt.Println("[DEBUG] PGS to SRT script found but Deno runtime is missing")
			return false
		}
	} else {
		fmt.Println("[DEBUG] PGS to SRT script not found, error:", err)
	}

	fmt.Println("[DEBUG] Final PGS to SRT script found status:", scriptFound)
	return scriptFound
}

// installDependency handles the installation of a specific dependency
func installDependency(w fyne.Window, tool string) {
	// Show a confirmation dialog before proceeding
	confirmMessage := fmt.Sprintf("This will install %s using Homebrew.\n\nDo you want to continue?", tool)
	dialog.ShowConfirm(fmt.Sprintf("Install %s", tool), confirmMessage, func(confirmed bool) {
		if confirmed {
			// Create a progress dialog
			progress := dialog.NewProgress(fmt.Sprintf("Installing %s", tool), "Preparing installation...", w)
			progress.Show()

			// Run the installation in a goroutine
			go func() {
				// Update progress
				progress.SetValue(0.1)

				// Prepare the installation command based on the tool
				var cmd *exec.Cmd
				var installDesc string

				// Check if brew is installed first
				if tool != "vobsub2srt" { // Skip brew check for vobsub2srt as it uses custom script
					_, err := exec.LookPath("brew")
					if err != nil {
						// Hide progress dialog
						progress.Hide()

						// Show error about Homebrew not being installed
						dialog.ShowError(
							fmt.Errorf("Homebrew is required but not installed. Please install Homebrew first:\n\nhttps://brew.sh"),
							w)
						return
					}
				}

				// Set up command and description based on tool
				// Using a case-insensitive approach to handle various tool name formats
				toolLower := strings.ToLower(tool)
				switch toolLower {
				case "mkvmerge", "mkvextract", "mkvm":
					// Install MKVToolNix via Homebrew
					cmd = exec.Command("brew", "install", "mkvtoolnix")
					installDesc = "Installing MKVToolNix (provides mkvmerge and mkvextract)"
				case "deno":
					// Install Deno via Homebrew
					cmd = exec.Command("brew", "install", "deno")
					installDesc = "Installing Deno runtime"
				case "tesseract":
					// Install Tesseract via Homebrew
					cmd = exec.Command("brew", "install", "tesseract")
					installDesc = "Installing Tesseract OCR engine"
				case "ffmpeg", "ffmp":
					// Install ffmpeg via Homebrew
					cmd = exec.Command("brew", "install", "ffmpeg")
					installDesc = "Installing FFmpeg multimedia framework"
				case "go":
					// Install Go via Homebrew
					cmd = exec.Command("brew", "install", "go")
					installDesc = "Installing Go programming language"
				case "vobsub2srt":
					// Use the custom installation script for VobSub2SRT
					execPath, err := os.Executable()
					if err != nil {
						fmt.Println("[ERROR] Failed to get executable path:", err)
					}

					scriptPath := filepath.Join(filepath.Dir(execPath), "install_vobsub2srt.sh")

					// Check if script exists
					if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
						progress.Hide()
						dialog.ShowError(
							fmt.Errorf("Installation script not found: %s", scriptPath),
							w)
						return
					}

					cmd = exec.Command("bash", scriptPath)
					installDesc = "Installing VobSub2SRT (may require additional dependencies)"
				case "pgstoSRT", "pgs-to-srt", "pgstoSrt", "pgstosrt":
					// Create a temp directory for creating the installation script
					tempDir, err := os.MkdirTemp("", "pgs-to-srt-install")
					if err != nil {
						progress.Hide()
						dialog.ShowError(
							fmt.Errorf("Failed to create temporary directory: %v", err),
							w)
						return
					}
					defer os.RemoveAll(tempDir) // Clean up when done

					// Create a bash script that will clone and install PGS-to-SRT
					// This ensures we can capture output and report progress
					scriptContent := `#!/bin/bash
					set -e
					
					# Check if git is installed
					if ! command -v git &> /dev/null; then
					  echo "Error: git is not installed. Please install git first."
					  exit 1
					fi
					
					# Check if deno is installed
					if ! command -v deno &> /dev/null; then
					  echo "Error: deno is not installed. Please install deno first."
					  exit 1
					fi
					
					# Remove existing directory and zip if they exist
					if [ -d "$HOME/pgs-to-srt" ]; then
					  echo "Removing existing pgs-to-srt directory..."
					  rm -rf "$HOME/pgs-to-srt"
					fi
					
					if [ -f "$HOME/pgs-to-srt.zip" ]; then
					  rm -f "$HOME/pgs-to-srt.zip"
					fi
					
					# Step 1: Clone the repository
					echo "Cloning PGS-to-SRT repository..."
					git clone https://github.com/wydengyre/pgs-to-srt.git "$HOME/pgs-to-srt"
					
					# Step 2: Download the ZIP file
					echo "Downloading PGS-to-SRT ZIP release..."
					ZIP_URL="https://github.com/wydengyre/pgs-to-srt/releases/download/release-5/pgs-to-srt.zip"
					
					# Use curl with fallback to wget
					if command -v curl &> /dev/null; then
					  curl -L "$ZIP_URL" -o "$HOME/pgs-to-srt.zip"
					else
					  wget -O "$HOME/pgs-to-srt.zip" "$ZIP_URL"
					fi
					
					# Step 3: Extract the ZIP file to a temporary directory first
					echo "Extracting files..."
					TMP_EXTRACT_DIR="$HOME/pgs-to-srt-tmp"
					mkdir -p "$TMP_EXTRACT_DIR"
					unzip -o "$HOME/pgs-to-srt.zip" -d "$TMP_EXTRACT_DIR"
					
					# Step 4: Find the pgs-to-srt.js file
					echo "Locating pgs-to-srt.js file..."
					JS_FILE=$(find "$TMP_EXTRACT_DIR" -name "pgs-to-srt.js" | head -n 1)
					
					if [ -z "$JS_FILE" ]; then
					  echo "Installation failed: pgs-to-srt.js not found in extracted files"
					  rm -rf "$TMP_EXTRACT_DIR"
					  rm -f "$HOME/pgs-to-srt.zip"
					  exit 1
					fi
					
					echo "Found pgs-to-srt.js at: $JS_FILE"
					
					# Copy all extracted files to the target location
					echo "Copying all extracted files to $HOME/pgs-to-srt..."
					
					# Find the parent directory containing all extracted files
					JS_DIR=$(dirname "$JS_FILE")
					
					# Copy all files and directories from the extraction directory
					cp -R "$JS_DIR"/* "$HOME/pgs-to-srt/"
					
					# Step 5: Install using deno
					echo "Installing PGS-to-SRT using deno..."
					cd "$HOME/pgs-to-srt"
					deno install --global -f --allow-read "pgs-to-srt.js"
					
					# Clean up
					rm -rf "$TMP_EXTRACT_DIR"
					rm -f "$HOME/pgs-to-srt.zip"
					
					echo "PGS-to-SRT installed successfully"
					exit 0
					`

					// Write the script to a temporary file
					installScriptPath := filepath.Join(tempDir, "install_pgs_to_srt.sh")
					err = os.WriteFile(installScriptPath, []byte(scriptContent), 0755)
					if err != nil {
						progress.Hide()
						dialog.ShowError(
							fmt.Errorf("Failed to create installation script: %v", err),
							w)
						return
					}

					// Execute the script
					cmd = exec.Command("bash", installScriptPath)
					installDesc = "Installing PGS-to-SRT script"

					// Update the global path variable to point to the installed script
					pgsToSrtScriptPath = filepath.Join(os.Getenv("HOME"), "pgs-to-srt", "pgs-to-srt.js")
				default:
					// Hide the progress dialog
					progress.Hide()
					dialog.ShowError(fmt.Errorf("Unknown tool: %s", tool), w)
					return
				}

				// Update progress dialog with specific tool info
				progress.Hide()
				progress = dialog.NewProgress("Installing Dependencies", installDesc, w)
				progress.Show()
				progress.SetValue(0.3)

				// Create a buffer to capture output in real-time
				var outputBuf bytes.Buffer
				cmd.Stdout = &outputBuf
				cmd.Stderr = &outputBuf

				// Start the command
				err := cmd.Start()
				if err != nil {
					progress.Hide()
					dialog.ShowError(fmt.Errorf("Failed to start installation: %v", err), w)
					return
				}

				// Update progress while command is running
				progress.SetValue(0.5)

				// Wait for command to complete
				err = cmd.Wait()
				output := outputBuf.Bytes()

				// Hide the progress dialog
				progress.Hide()

				if err != nil {
					// Show detailed error dialog with output and suggestions
					errorMsg := fmt.Sprintf("Installation of %s failed.\n\nError: %v\n\n", tool, err)

					// Add output but limit it to avoid huge dialog
					outputStr := string(output)
					if len(outputStr) > 500 {
						outputStr = outputStr[:500] + "...\n(output truncated)"
					}
					errorMsg += "Output:\n" + outputStr + "\n\n"

					// Add suggestions based on the tool
					switch tool {
					case "vobsub2srt":
						// Get executable path again for suggestion
						suggestionExecPath, _ := os.Executable()
						errorMsg += "Suggestions:\n" +
							"- Make sure cmake is installed (brew install cmake)\n" +
							"- Make sure tesseract is installed (brew install tesseract)\n" +
							"- Try running the script manually: bash " + filepath.Join(filepath.Dir(suggestionExecPath), "install_vobsub2srt.sh")
					default:
						errorMsg += "Suggestions:\n" +
							"- Make sure Homebrew is properly installed\n" +
							"- Try running 'brew doctor' to diagnose Homebrew issues\n" +
							"- Try installing manually: brew install " + tool
					}

					dialog.ShowError(errors.New(errorMsg), w)
				} else {
					// Verify installation was successful by checking if tool is now available
					successful := false

					// Give the system a moment to register the new installation
					time.Sleep(500 * time.Millisecond)

					// Check if tool is now installed
					dependencyResults := checkDependencies()
					if installed, ok := dependencyResults[tool]; ok && installed {
						successful = true
					}

					if successful {
						// Show success dialog
						dialog.ShowInformation(
							"Installation Complete",
							fmt.Sprintf("%s has been successfully installed.\n\nThe application will now recognize this tool.", tool),
							w)

						// Update dependency status
						updateDependencyStatus(w)
					} else {
						// Installation seemed to succeed but tool still not found
						dialog.ShowInformation(
							"Installation Completed",
							fmt.Sprintf("The installation process completed, but %s may not be properly installed.\n\nYou may need to restart the application or your computer.", tool),
							w)
					}
				}
				// Update the dependency status
				updateDependencyStatus(w)
			}()
		}
	}, w)
}

// updateDependencyStatus checks dependencies and updates the UI
func updateDependencyStatus(w fyne.Window) {
	// Check dependencies
	dependencyResults := checkDependencies()

	// Update the status text with improved formatting
	dependencyStatus := "Current Status:\n"
	allDependenciesInstalled := true

	// Track missing tools
	missingTools := []string{}

	// We'll use a simpler approach with plain text for now since color styling is causing issues

	// Process each dependency
	for tool, installed := range dependencyResults {
		var status string

		if installed {
			status = "✅ Installed"
		} else {
			status = "❌ Not found"
			allDependenciesInstalled = false
			missingTools = append(missingTools, tool)
		}

		dependencyStatus += fmt.Sprintf("- %s: %s\n", tool, status)
	}

	// Add summary message
	if !allDependenciesInstalled {
		dependencyStatus += "\n⚠️ Some required tools are missing. Please install them before using all features.\n"
	} else {
		dependencyStatus += "\n✅ All required tools are installed.\n"
	}

	// Find and update the dependency result label in the Settings tab
	if tabs, ok := w.Content().(*container.AppTabs); ok {
		for _, tab := range tabs.Items {
			if tab.Text == "Settings" {
				if settingsContainer, ok := tab.Content.(*fyne.Container); ok {
					for _, child := range settingsContainer.Objects {
						if label, ok := child.(*widget.Label); ok && strings.Contains(label.Text, "System Dependency Check") {
							label.SetText(dependencyStatus)
							break
						}
					}
				}
			}
		}
	}

	// Update dependency buttons
	// Clear existing buttons
	if tabs, ok := w.Content().(*container.AppTabs); ok {
		for _, tab := range tabs.Items {
			if tab.Text == "Settings" {
				if settingsContainer, ok := tab.Content.(*fyne.Container); ok {
					for _, child := range settingsContainer.Objects {
						if buttonContainer, ok := child.(*fyne.Container); ok && len(buttonContainer.Objects) > 0 {
							if _, ok := buttonContainer.Objects[0].(*widget.Button); ok {
								// Found the button container, clear it
								buttonContainer.Objects = []fyne.CanvasObject{}
								break
							}
						}
					}
				}
			}
		}
	}

	// Add buttons for missing tools
	if len(missingTools) > 0 {
		// Create install all button
		installAllBtn := widget.NewButton("Install All Missing Dependencies", func() {
			installDependencies(missingTools, w)
		})
		installAllBtn.Importance = widget.HighImportance

		// Add to dependency buttons container
		if tabs, ok := w.Content().(*container.AppTabs); ok {
			for _, tab := range tabs.Items {
				if tab.Text == "Settings" {
					if settingsContainer, ok := tab.Content.(*fyne.Container); ok {
						for _, child := range settingsContainer.Objects {
							if buttonContainer, ok := child.(*fyne.Container); ok && len(buttonContainer.Objects) == 0 {
								// Found the empty button container
								buttonContainer.Add(installAllBtn)

								// Add individual install buttons
								for _, tool := range missingTools {
									installBtn := widget.NewButton(fmt.Sprintf("Install %s", tool), func(t string) func() {
										return func() {
											installDependencies([]string{t}, w)
										}
									}(tool))
									buttonContainer.Add(installBtn)
								}
								break
							}
						}
					}
				}
			}
		}
	}
}

// installDependencies installs the specified missing tools
func installDependencies(tools []string, w fyne.Window) {
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Installing Dependencies", "Installing required tools...", w)
	progress.Show()

	// Install dependencies in a goroutine
	go func() {
		successCount := 0
		failureCount := 0

		for _, tool := range tools {
			fmt.Printf("[INFO] Installing %s...\n", tool)

			var cmd *exec.Cmd

			// Determine installation command based on tool
			switch tool {
			case "mkvmerge":
				cmd = exec.Command("brew", "install", "mkvtoolnix")
			case "deno":
				cmd = exec.Command("brew", "install", "deno")
			case "tesseract":
				cmd = exec.Command("brew", "install", "tesseract")
			case "ffmpeg":
				cmd = exec.Command("brew", "install", "ffmpeg")
			case "vobsub2srt":
				// Get the script path relative to the executable
				execPath, err := os.Executable()
				if err != nil {
					fmt.Println("[ERROR] Failed to get executable path:", err)
					execPath = "."
				}
				execDir := filepath.Dir(execPath)
				scriptPath := filepath.Join(execDir, "install_vobsub2srt.sh")
				cmd = exec.Command("bash", scriptPath)
			default:
				fmt.Printf("[ERROR] Unknown tool: %s\n", tool)
				failureCount++
				continue
			}

			// Run the installation command
			_, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("[ERROR] Failed to install %s: %v\n", tool, err)
				failureCount++
			} else {
				successCount++
			}
		}

		// Hide the progress dialog
		progress.Hide()

		// Show results
		if failureCount == 0 {
			dialog.ShowInformation("Installation Complete",
				fmt.Sprintf("All %d dependencies have been successfully installed.\n\nPlease restart the application to use all features.", successCount),
				w)
		} else {
			dialog.ShowInformation("Installation Results",
				fmt.Sprintf("%d dependencies installed successfully.\n%d dependencies failed to install.\n\nPlease check the logs for details and try installing the failed dependencies individually.",
					successCount, failureCount),
				w)
		}

		// Update the dependency status
		updateDependencyStatus(w)
	}()
}

func createUtilitiesTab(result *widget.Label) *fyne.Container {
	// Create a new Label for utilities tab results
	utilitiesResult := widget.NewLabel("Results will appear here...")
	utilitiesResult.Wrapping = fyne.TextWrapWord
	utilitiesResultScroll := container.NewScroll(utilitiesResult)
	utilitiesResultScroll.SetMinSize(fyne.NewSize(850, 200))

	// Create file selection widgets for MKV operations
	mkvFileLabel := widget.NewLabel("No MKV file selected")
	selectMkvBtn := widget.NewButton("Select MKV File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".mkv") {
				dialog.ShowInformation("Invalid File", "Please select an MKV file", fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}

			mkvFileLabel.SetText(filePath)
			utilitiesResult.SetText("MKV file selected: " + filePath)
		}, fyne.CurrentApp().Driver().AllWindows()[0])
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mkv"}))
		fd.Show()
	})

	// Create file selection widgets for SRT operations
	srtFileLabel := widget.NewLabel("No SRT file selected")
	selectSrtBtn := widget.NewButton("Select SRT File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".srt") {
				dialog.ShowInformation("Invalid File", "Please select an SRT file", fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}

			srtFileLabel.SetText(filePath)
			utilitiesResult.SetText("SRT file selected: " + filePath)
		}, fyne.CurrentApp().Driver().AllWindows()[0])
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".srt"}))
		fd.Show()
	})

	// Create MKV utility operations
	mkvInfoBtn := widget.NewButton("MKV Info", func() {
		mkvPath := mkvFileLabel.Text
		if mkvPath == "No MKV file selected" {
			dialog.ShowInformation("No File Selected", "Please select an MKV file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		utilitiesResult.SetText("Getting MKV information...\n")

		// Run mkvinfo command
		go func() {
			cmd := exec.Command("mkvinfo", mkvPath)
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText("MKV Information for: " + mkvPath + "\n\n" + string(output))
			})
		}()
	})

	mkvExtractChaptersBtn := widget.NewButton("Extract Chapters", func() {
		mkvPath := mkvFileLabel.Text
		if mkvPath == "No MKV file selected" {
			dialog.ShowInformation("No File Selected", "Please select an MKV file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		// Get output directory (same as MKV file)
		dir := filepath.Dir(mkvPath)
		baseName := filepath.Base(mkvPath)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		outputPath := filepath.Join(dir, baseName+"_chapters.txt")

		utilitiesResult.SetText("Extracting chapters to: " + outputPath + "\n")

		// Run mkvextract command for chapters
		go func() {
			cmd := exec.Command("mkvextract", mkvPath, "chapters", outputPath)
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText(utilitiesResult.Text + "\nChapters extracted successfully to: " + outputPath + "\n" + string(output))
			})
		}()
	})

	// Create SRT utility operations
	srtFixEncodingBtn := widget.NewButton("Fix SRT Encoding", func() {
		srtPath := srtFileLabel.Text
		if srtPath == "No SRT file selected" {
			dialog.ShowInformation("No File Selected", "Please select an SRT file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		utilitiesResult.SetText("Fixing SRT encoding...\n")

		// Run iconv command to fix encoding
		go func() {
			// Create a backup of the original file
			backupPath := srtPath + ".bak"
			if err := copyFile(srtPath, backupPath); err != nil {
				fyne.Do(func() {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError creating backup: " + err.Error())
				})
				return
			}

			// Try to detect and convert encoding to UTF-8
			cmd := exec.Command("iconv", "-f", "ISO-8859-1", "-t", "UTF-8", srtPath, "-o", srtPath+".tmp")
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				// Replace original with converted file
				if err := os.Rename(srtPath+".tmp", srtPath); err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError replacing file: " + err.Error())
					return
				}

				utilitiesResult.SetText(utilitiesResult.Text + "\nSRT encoding fixed successfully.\nOriginal backup saved to: " + backupPath + "\n" + string(output))
			})
		}()
	})

	srtFixTimingBtn := widget.NewButton("Fix SRT Timing", func() {
		srtPath := srtFileLabel.Text
		if srtPath == "No SRT file selected" {
			dialog.ShowInformation("No File Selected", "Please select an SRT file first", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		// Show dialog to get timing offset
		offsetEntry := widget.NewEntry()
		offsetEntry.SetPlaceHolder("e.g., +1.5 or -2.3 (seconds)")

		dialog.ShowCustomConfirm("Adjust SRT Timing", "Apply", "Cancel",
			container.NewVBox(
				widget.NewLabel("Enter timing offset in seconds:"),
				offsetEntry,
			),
			func(confirmed bool) {
				if !confirmed || offsetEntry.Text == "" {
					return
				}

				offset := offsetEntry.Text
				utilitiesResult.SetText("Adjusting SRT timing with offset: " + offset + " seconds...\n")

				go func() {
					// Create a backup of the original file
					backupPath := srtPath + ".bak"
					if err := copyFile(srtPath, backupPath); err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError creating backup: " + err.Error())
						})
						return
					}

					// Read the SRT file
					content, err := os.ReadFile(srtPath)
					if err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError reading SRT file: " + err.Error())
						})
						return
					}

					// Parse offset
					offsetFloat, err := strconv.ParseFloat(offset, 64)
					if err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nInvalid offset format: " + err.Error())
						})
						return
					}

					// Apply offset to timing
					adjustedContent := adjustSRTTiming(string(content), offsetFloat)

					// Write back to file
					if err := os.WriteFile(srtPath, []byte(adjustedContent), 0644); err != nil {
						fyne.Do(func() {
							utilitiesResult.SetText(utilitiesResult.Text + "\nError writing adjusted SRT file: " + err.Error())
						})
						return
					}

					fyne.Do(func() {
						utilitiesResult.SetText(utilitiesResult.Text + "\nSRT timing adjusted successfully.\nOriginal backup saved to: " + backupPath)
					})
				}()
			},
			fyne.CurrentApp().Driver().AllWindows()[0],
		)
	})

	// Create layout for the Utilities tab
	mkvSection := container.NewVBox(
		widget.NewLabelWithStyle("MKV Utilities", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(selectMkvBtn, mkvFileLabel),
		container.NewHBox(mkvInfoBtn, mkvExtractChaptersBtn),
	)

	srtSection := container.NewVBox(
		widget.NewLabelWithStyle("SRT Utilities", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(selectSrtBtn, srtFileLabel),
		container.NewHBox(srtFixEncodingBtn, srtFixTimingBtn),
	)

	utilitiesTabContent := container.NewVBox(
		mkvSection,
		widget.NewSeparator(),
		srtSection,
		widget.NewSeparator(),
		widget.NewLabel("Results:"),
		utilitiesResultScroll,
	)

	return utilitiesTabContent
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// Helper function to adjust SRT timing


func adjustSRTTiming(content string, offsetSeconds float64) string {
	lines := strings.Split(content, "\n")
	result := []string{}

	// Regular expression to match SRT timestamp format: 00:00:00,000 --> 00:00:00,000
	re := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3}) --> (\d{2}):(\d{2}):(\d{2}),(\d{3})`)

	for _, line := range lines {
		// Check if the line contains timestamps
		if re.MatchString(line) {
			// Apply offset to both start and end timestamps
			adjustedLine := re.ReplaceAllStringFunc(line, func(match string) string {
				parts := re.FindStringSubmatch(match)
				if len(parts) != 9 {
					return match
				}

				// Parse start time
				startHour, _ := strconv.Atoi(parts[1])
				startMin, _ := strconv.Atoi(parts[2])
				startSec, _ := strconv.Atoi(parts[3])
				startMs, _ := strconv.Atoi(parts[4])

				// Parse end time
				endHour, _ := strconv.Atoi(parts[5])
				endMin, _ := strconv.Atoi(parts[6])
				endSec, _ := strconv.Atoi(parts[7])
				endMs, _ := strconv.Atoi(parts[8])

				// Convert to milliseconds and apply offset
				startTimeMs := startHour*3600000 + startMin*60000 + startSec*1000 + startMs
				endTimeMs := endHour*3600000 + endMin*60000 + endSec*1000 + endMs

				offsetMs := int(offsetSeconds * 1000)
				startTimeMs += offsetMs
				endTimeMs += offsetMs

				// Ensure times don't go negative
				if startTimeMs < 0 {
					startTimeMs = 0
				}
				if endTimeMs < 0 {
					endTimeMs = 0
				}

				// Convert back to SRT format
				startHour = startTimeMs / 3600000
				startTimeMs %= 3600000
				startMin = startTimeMs / 60000
				startTimeMs %= 60000
				startSec = startTimeMs / 1000
				startMs = startTimeMs % 1000

				endHour = endTimeMs / 3600000
				endTimeMs %= 3600000
				endMin = endTimeMs / 60000
				endTimeMs %= 60000
				endSec = endTimeMs / 1000
				endMs = endTimeMs % 1000

				return fmt.Sprintf("%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d",
					startHour, startMin, startSec, startMs,
					endHour, endMin, endSec, endMs)
			})

			result = append(result, adjustedLine)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func main() {
	trackList := container.NewVBox()
	// Create a scrollable container for the track list
	trackListScroll := container.NewScroll(trackList)
	// Set a minimum size for the track list scroll area to show more tracks
	trackListScroll.SetMinSize(fyne.NewSize(850, 250))

	// Create app with explicit ID and set metadata directly
	a := app.NewWithID("com.gmm.subtitleforge")
	a.SetIcon(theme.FileTextIcon())

	// Apply theme based on saved preference
	savedTheme := a.Preferences().StringWithFallback("theme", "Dark Theme")
	switch savedTheme {
	case "Light Theme":
		customTheme := NewCustomThemeWithPrefs(LightTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Dark Theme":
		customTheme := NewCustomThemeWithPrefs(DarkTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Blue Theme":
		customTheme := NewCustomThemeWithPrefs(BlueCoolTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Warm Theme":
		customTheme := NewCustomThemeWithPrefs(WarmToneTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Green Theme":
		customTheme := NewCustomThemeWithPrefs(VibrantGreenTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Spring Theme":
		customTheme := NewCustomThemeWithPrefs(SpringTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Summer Theme":
		customTheme := NewCustomThemeWithPrefs(SummerTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Autumn Theme":
		customTheme := NewCustomThemeWithPrefs(AutumnTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	case "Winter Theme":
		customTheme := NewCustomThemeWithPrefs(WinterTheme(), true).(*CustomTheme)
		a.Settings().SetTheme(customTheme)
	default:
		a.Settings().SetTheme(theme.DefaultTheme())
	}

	// Create main window with explicit name
	w := a.NewWindow("Subtitle Forge")
	// Set app metadata on window
	w.SetMaster()
	w.CenterOnScreen()
	// Ensure window has adequate size
	// In Fyne, windows are resizable by default unless explicitly set as fixed size
	w.Resize(fyne.NewSize(1024, 768))
	// Explicitly ensure the window is not fixed size
	w.SetFixedSize(false)

	// Setup keyboard shortcuts
	setupKeyboardShortcuts := func(fileOpenFunc, dirChangeFunc, loadTracksFunc, startExtractFunc func()) {
		// Ctrl+O for opening files
		ctrlO := &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlO, func(shortcut fyne.Shortcut) {
			fileOpenFunc()
		})

		// Ctrl+D for changing directory
		ctrlD := &desktop.CustomShortcut{KeyName: fyne.KeyD, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlD, func(shortcut fyne.Shortcut) {
			dirChangeFunc()
		})

		// Ctrl+L for loading tracks
		ctrlL := &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlL, func(shortcut fyne.Shortcut) {
			loadTracksFunc()
		})

		// Ctrl+E for starting extraction
		ctrlE := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl}
		w.Canvas().AddShortcut(ctrlE, func(shortcut fyne.Shortcut) {
			startExtractFunc()
		})
	}

	// Load window size from preferences or use default size
	defaultWidth := float32(900)
	defaultHeight := float32(700)
	width := float32(a.Preferences().Float("window_width"))
	height := float32(a.Preferences().Float("window_height"))

	if width == 0 || height == 0 {
		// Use default size for first launch
		width = defaultWidth
		height = defaultHeight
	}

	// Resize window to saved or default size
	w.Resize(fyne.NewSize(width, height))

	// Save window size when it changes
	// Use a timer to periodically check and save window size
	var lastSize fyne.Size
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fyne.Do(func() {
				currentSize := w.Canvas().Size()
				// Only save if size has changed
				if currentSize.Width != lastSize.Width || currentSize.Height != lastSize.Height {
					a.Preferences().SetFloat("window_width", float64(currentSize.Width))
					a.Preferences().SetFloat("window_height", float64(currentSize.Height))
					lastSize = currentSize
				}
			})
		}
	}()

	// Also save window size when closing
	w.SetCloseIntercept(func() {
		// Save current window size
		currentSize := w.Canvas().Size()
		a.Preferences().SetFloat("window_width", float64(currentSize.Width))
		a.Preferences().SetFloat("window_height", float64(currentSize.Height))

		// Close the window
		w.Close()
	})

	// Check dependencies at startup
	dependencyResults := checkDependencies()

	var mkvPath string
	var outDir string
	var trackItems []*TrackItem

	selectedFile := widget.NewLabel("No MKV file selected.")
	selectedDir := widget.NewLabel("No output directory selected.")
	result := widget.NewLabel("Results will appear here...")
	result.Wrapping = fyne.TextWrapWord
	// Make the result area larger to show more debug information
	resultScroll := container.NewScroll(result)
	resultScroll.SetMinSize(fyne.NewSize(780, 200))

	// Set up file drop handling
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		// Handle key events if needed
	})

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			filePath := uris[0].Path()
			fileExt := strings.ToLower(filepath.Ext(filePath))

			if fileExt == ".mkv" {
				// Handle MKV file drop
				mkvPath = filePath
				a.SendNotification(&fyne.Notification{
					Title:   "File Dropped",
					Content: "MKV file loaded: " + filepath.Base(filePath),
				})

				// Update UI
				selectedFile.SetText(mkvPath)

				// Set output directory to the same directory as the MKV file
				outDir = filepath.Dir(mkvPath)
				selectedDir.SetText(outDir)

				// Clear previous tracks
				trackItems = []*TrackItem{}
				trackList.Objects = nil
				trackList.Refresh()

				result.SetText("MKV file dropped and loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file.")
			} else {
				a.SendNotification(&fyne.Notification{
					Title:   "Invalid File",
					Content: "Please drop an MKV file only.",
				})
			}
		}
	})

	// Display dependency check results
	dependencyStatus := "System Dependency Check:\n"
	allDependenciesInstalled := true

	for tool, installed := range dependencyResults {
		status := "✅ Installed"
		if !installed {
			status = "❌ Not found"
			allDependenciesInstalled = false
		}
		dependencyStatus += fmt.Sprintf("- %s: %s\n", tool, status)
	}

	if !allDependenciesInstalled {
		dependencyStatus += "\n⚠️ Some required tools are missing. Please install them before using all features.\n"
	} else {
		dependencyStatus += "\n✅ All required tools are installed.\n"
	}

	result.SetText(dependencyStatus)

	// Create a container for dependency-related buttons
	dependencyButtons := container.NewVBox()

	// Create a container for the install all button
	installAllContainer := container.NewHBox()

	// Create a list of missing dependencies
	missingDependencies := []string{}
	for tool, installed := range dependencyResults {
		if !installed {
			missingDependencies = append(missingDependencies, tool)
		}
	}

	// Add individual install buttons for each missing dependency
	if len(missingDependencies) > 0 {
		// Add header for install buttons
		dependencyButtons.Add(widget.NewLabel("Install Missing Dependencies:"))

		// Add buttons for each missing dependency
		for _, tool := range missingDependencies {
			// Create a local copy of the tool name for the closure
			toolName := tool

			// Create button with appropriate label
			buttonLabel := fmt.Sprintf("Install %s", toolName)
			installButton := widget.NewButton(buttonLabel, func() {
				installDependency(w, toolName)
			})

			// Add the install button to the dependency buttons container
			dependencyButtons.Add(installButton)
		}

		// Add an "Install All" button if there are multiple missing dependencies
		if len(missingDependencies) > 1 {
			installAllButton := widget.NewButton("Install All Missing Dependencies", func() {
				// Show confirmation dialog
				dialog.ShowConfirm("Install All Dependencies",
					"This will attempt to install all missing dependencies.\n\nSome installations may require sudo privileges.\n\nDo you want to continue?",
					func(confirmed bool) {
						if confirmed {
							// Create a simple progress dialog
							progress := dialog.NewProgress("Installing Dependencies", "Installing missing dependencies...", w)
							progress.Show()

							// Run installations in a goroutine
							go func() {
								totalTools := len(missingDependencies)
								successCount := 0
								failureCount := 0

								// Install each tool
								for i, tool := range missingDependencies {
									// Update progress value - increment for each tool
									progressValue := float64(i) / float64(totalTools)
									progress.SetValue(progressValue)

									// Prepare the installation command based on the tool
									var cmd *exec.Cmd
									switch tool {
									case "mkvmerge", "mkvextract":
										// MKVToolNix includes both mkvmerge and mkvextract
										cmd = exec.Command("brew", "install", "mkvtoolnix")
									case "deno":
										cmd = exec.Command("brew", "install", "deno")
									case "tesseract":
										cmd = exec.Command("brew", "install", "tesseract")
									case "ffmpeg":
										cmd = exec.Command("brew", "install", "ffmpeg")
									case "vobsub2srt":
										// Get the script path relative to the executable
										execPath, err := os.Executable()
										if err != nil {
											fmt.Println("[ERROR] Failed to get executable path:", err)
											execPath = "."
										}
										execDir := filepath.Dir(execPath)
										scriptPath := filepath.Join(execDir, "install_vobsub2srt.sh")
										cmd = exec.Command("bash", scriptPath)
									default:
										fmt.Printf("[ERROR] Unknown tool: %s\n", tool)
										failureCount++
										continue
									}

									// Run the installation command
									_, err := cmd.CombinedOutput()
									if err != nil {
										fmt.Printf("[ERROR] Failed to install %s: %v\n", tool, err)
										failureCount++
									} else {
										successCount++
									}
								}

								// Hide the progress dialog
								progress.Hide()

								// Show results
								if failureCount == 0 {
									dialog.ShowInformation("Installation Complete",
										fmt.Sprintf("All %d dependencies have been successfully installed.\n\nPlease restart the application to use all features.", successCount),
										w)
								} else {
									dialog.ShowInformation("Installation Results",
										fmt.Sprintf("%d dependencies installed successfully.\n%d dependencies failed to install.\n\nPlease check the logs for details and try installing the failed dependencies individually.",
											successCount, failureCount),
										w)
								}

								// Update the dependency status
								updateDependencyStatus(w)
							}()
						}
					}, w)
			})

			// Add the install all button to the container
			installAllContainer.Add(installAllButton)
			dependencyButtons.Add(installAllContainer)
		}
	}

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)

	currentTrackLabel := widget.NewLabel("")

	// Button to select MKV file
	fileBtn := widget.NewButton("Select MKV File (or Drag & Drop)", func() {
		// Create a file filter for MKV files
		filter := storage.NewExtensionFileFilter([]string{".mkv"})

		// Create a file dialog with explicit styling for readability
		fd := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				return
			}

			filePath := file.URI().Path()
			fileExt := strings.ToLower(filepath.Ext(filePath))

			// Double-check that it's an MKV file
			if fileExt != ".mkv" {
				dialog.ShowError(fmt.Errorf("Please select an MKV file only."), w)
				return
			}

			mkvPath = filePath
			selectedFile.SetText(mkvPath)

			// Set output directory to the same directory as the MKV file
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)

			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()

			result.SetText("MKV file loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file.")
		}, w)

		fd.SetFilter(filter)
		fd.Show()
	})

	// Button to select output directory (optional, as it's auto-set)
	dirBtn := widget.NewButton("Change Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			outDir = uri.Path()
			selectedDir.SetText(outDir)
		}, w)
	})

	// Button to load tracks from MKV file
	loadTracksBtn := widget.NewButton("Load Tracks", func() {
		if mkvPath == "" {
			dialog.ShowError(fmt.Errorf("Please select or drag & drop an MKV file first."), w)
			return
		}

		// Run mkvmerge to get track info
		cmd := exec.Command("mkvmerge", "-J", mkvPath)
		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error running mkvmerge: %v", err), w)
			return
		}

		// Parse JSON output
		var mkvInfo map[string]interface{}
		err = json.Unmarshal(output, &mkvInfo)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error parsing mkvmerge output: %v", err), w)
			return
		}

		// Extract tracks
		tracks, ok := mkvInfo["tracks"].([]interface{})
		if !ok {
			dialog.ShowError(fmt.Errorf("No tracks found in MKV file."), w)
			return
		}

		// Clear previous tracks
		trackItems = []*TrackItem{}
		trackList.Objects = nil

		// Process subtitle tracks
		for _, track := range tracks {
			trackMap, ok := track.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if this is a subtitle track
			trackType, ok := trackMap["type"].(string)
			if !ok || trackType != "subtitles" {
				continue
			}

			// Get track properties
			properties, ok := trackMap["properties"].(map[string]interface{})
			if !ok {
				continue
			}

			trackID := int(trackMap["id"].(float64))

			// Get language with nil check
			var trackLang string
			if properties != nil {
				if lang, ok := properties["language"].(string); ok {
					trackLang = lang
				} else {
					trackLang = "und" // undefined language code
				}
			} else {
				trackLang = "und" // undefined language code
			}

			trackCodec := trackMap["codec"].(string)

			// Get track name if available
			var trackName string
			if name, ok := properties["track_name"].(string); ok {
				trackName = name
			} else {
				trackName = ""
			}

			// Create UI elements for this track
			check := widget.NewCheck("", nil)
			check.SetChecked(true)
			status := widget.NewLabel("[ ]")

			// Create track item
			t := &TrackItem{
				Num:    trackID,
				Lang:   trackLang,
				Codec:  trackCodec,
				Name:   trackName,
				State:  "Pending",
				Check:  check,
				Status: status,
			}

			// Add OCR option for PGS subtitles, ASS/SSA subtitles, and VobSub subtitles
			if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" ||
				strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") ||
				strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station") ||
				t.Codec == "vobsub" || t.Codec == "VobSub" {
				t.ConvertOCR = widget.NewCheck("", nil)
				t.ConvertOCR.SetChecked(true)

				// Add language selection for OCR conversion
				if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" || t.Codec == "vobsub" || t.Codec == "VobSub" {
					// Create language options
					langOptions := []string{
						"Auto (" + t.Lang + ")", // Auto option with detected language
						"English (en)",
						"French (fr)",
						"German (de)",
						"Spanish (es)",
						"Italian (it)",
						"Portuguese (pt)",
						"Dutch (nl)",
						"Russian (ru)",
						"Japanese (ja)",
						"Chinese (zh)",
						"Korean (ko)",
						"Czech (cs)",
						"Polish (pl)",
						"Swedish (sv)",
						"Danish (da)",
						"Finnish (fi)",
						"Norwegian (no)",
						"Hungarian (hu)",
						"Greek (el)",
						"Turkish (tr)",
						"Arabic (ar)",
						"Hebrew (he)",
						"Thai (th)",
					}

					// Create language dropdown
					t.LangSelect = widget.NewSelect(langOptions, nil)
					t.LangSelect.SetSelected("Auto (" + t.Lang + ")")
				} else {
					t.LangSelect = nil
				}
			} else {
				t.ConvertOCR = nil
				t.LangSelect = nil
			}

			trackItems = append(trackItems, t)

			// Create row for this track
			trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", trackID, trackLang, trackCodec, trackName))

			var row *fyne.Container
			if t.ConvertOCR != nil {
				// For PGS/VobSub subtitles, show OCR option and language selection
				ocrLabel := widget.NewLabel("Convert to SRT")

				if t.LangSelect != nil {
					// Add language selection dropdown for OCR-based conversion
					langLabel := widget.NewLabel("OCR Language:")
					row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
				} else {
					// For ASS/SSA conversion (no OCR language needed)
					row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel)
				}
			} else {
				// For other subtitle formats
				row = container.NewHBox(check, status, trackInfo)
			}

			trackList.Add(row)
		}
		trackList.Refresh()

		result.SetText("Tracks loaded. Select the tracks you want to extract, then click 'Start Extraction'")
	})

	// Button to start extraction of selected tracks
	startExtractBtn := widget.NewButton("Start Extraction", func() {
		if mkvPath == "" || outDir == "" {
			dialog.ShowError(fmt.Errorf("Please select both MKV file and output directory."), w)
			return
		}

		go func() {
			selected := []*TrackItem{}
			for _, t := range trackItems {
				if t.Check.Checked {
					selected = append(selected, t)
				}
			}
			if len(selected) == 0 {
				// Thread-safe UI update
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "No Tracks",
					Content: "No tracks selected.",
				})
				return
			}

			// Set up progress bar
			fyne.Do(func() {
				result.SetText("Extracting selected tracks...")
				progress.Max = float64(len(selected))
				progress.SetValue(0)
			})

			tracksDone := 0
			var output []byte
			var err error

			for i, t := range selected {
				// Update UI on main thread
				fyne.Do(func() {
					currentTrackLabel.SetText(fmt.Sprintf("Extracting track %d of %d: %s (%s) %s", i+1, len(selected), t.Lang, t.Codec, t.Name))
				})

				// Extract the subtitle track
				var outFile string

				// Get base filename without extension
				mkvBaseName := filepath.Base(mkvPath)
				mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))

				// Check if this is a PGS track with OCR conversion requested
				if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS") {
					// First extract as PGS
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting PGS extraction process")
					})
					tempPgsFile := fmt.Sprintf("%s.track%d_%s.sup", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absPgsPath := filepath.Join(outDir, tempPgsFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting PGS track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== PGS Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("PGS file: %s\n", tempPgsFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absPgsPath))
					})

					// Extract PGS first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempPgsFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})

					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
					cmd.Dir = outDir

					// Run the command and capture output
					output, err = cmd.CombinedOutput()

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					pgsFilePath := filepath.Join(outDir, tempPgsFile)
					fileInfo, statErr := os.Stat(pgsFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty (0 bytes)")
					} else {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted PGS file (%d bytes)", fileInfo.Size()))
						})
					}

					if err == nil {
						// Debug point after successful extraction
						// Create a detailed progress bar for the conversion process
						conversionProgress := widget.NewProgressBar()
						conversionProgress.Min = 0
						conversionProgress.Max = 100 // Percentage-based progress
						conversionProgress.SetValue(0)

						conversionLabel := widget.NewLabel("Converting PGS to SRT...")
						statusLabel := widget.NewLabel("Initializing OCR process...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Estimated time remaining: calculating...")

						// Track conversion start time and progress data
						conversionStartTime := time.Now()
						var progressMutex sync.Mutex
						var progressData = struct {
							currentFrame int
							totalFrames  int
							frameRate    float64 // frames processed per second
							lastUpdate   time.Time
						}{
							currentFrame: 0,
							totalFrames:  0, // Will be updated when we parse output
							frameRate:    0,
							lastUpdate:   time.Now(),
						}

						// Create a ticker to update elapsed time and estimated remaining time
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastElapsedText, lastRemainingText string

							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

								// Calculate estimated time remaining
								progressMutex.Lock()
								currentFrame := progressData.currentFrame
								totalFrames := progressData.totalFrames
								frameRate := progressData.frameRate
								progressMutex.Unlock()

								var newRemainingText string
								var progressValue float64

								if totalFrames > 0 && currentFrame > 0 && frameRate > 0 {
									// Calculate percentage complete
									progressValue = float64(currentFrame) / float64(totalFrames) * 100

									// Calculate remaining time
									framesRemaining := totalFrames - currentFrame
									secondsRemaining := float64(framesRemaining) / frameRate
									remaining := time.Duration(secondsRemaining * float64(time.Second))
									remaining = remaining.Round(time.Second)

									newRemainingText = fmt.Sprintf("Estimated time remaining: %s", remaining)
								} else {
									newRemainingText = "Estimated time remaining: calculating..."
									progressValue = 0
								}

								// Only update UI if text has changed to reduce UI operations
								if newElapsedText != lastElapsedText || newRemainingText != lastRemainingText {
									lastElapsedText = newElapsedText
									lastRemainingText = newRemainingText

									fyne.Do(func() {
										elapsedLabel.SetText(newElapsedText)
										remainingLabel.SetText(newRemainingText)
										conversionProgress.SetValue(progressValue)
									})
								}
							}
						}()

						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] PGS extraction completed successfully, starting conversion process")

							// Show the conversion progress bar and labels
							currentTrackLabel.SetText("Converting PGS to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
							))
							trackList.Refresh()
						})

						// Use the configured PGS-to-SRT script with Deno
						pgsToSrtScript := pgsToSrtScriptPath
						// Get language from user selection or use track language as default
						langCode := "eng" // Default to English
						if t.Lang != "" {
							langCode = t.Lang
						}

						// Check if user has selected a specific language
						if t.LangSelect != nil && t.LangSelect.Selected != "" && !strings.HasPrefix(t.LangSelect.Selected, "Auto") {
							// Extract the language code from the selection (format: "Language (code)")
							selection := t.LangSelect.Selected
							// Extract the code part between parentheses
							if start := strings.LastIndex(selection, "("); start != -1 {
								if end := strings.LastIndex(selection, ")"); end != -1 && end > start {
									// Extract the 2-letter code
									twoLetterCode := selection[start+1 : end]
									fyne.Do(func() {
										result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] User selected OCR language: %s (code: %s)", selection, twoLetterCode))
									})

									// Map 2-letter code to 3-letter code for Tesseract
									langCodeMap := map[string]string{
										"en": "eng", // English
										"fr": "fra", // French
										"de": "deu", // German
										"it": "ita", // Italian
										"es": "spa", // Spanish
										"pt": "por", // Portuguese
										"nl": "nld", // Dutch
										"sv": "swe", // Swedish
										"no": "nor", // Norwegian
										"da": "dan", // Danish
										"fi": "fin", // Finnish
										"ja": "jpn", // Japanese
										"ko": "kor", // Korean
										"zh": "chi", // Chinese
										"ru": "rus", // Russian
										"pl": "pol", // Polish
										"cs": "ces", // Czech
										"hu": "hun", // Hungarian
										"el": "ell", // Greek
										"tr": "tur", // Turkish
										"ar": "ara", // Arabic
										"he": "heb", // Hebrew
										"th": "tha", // Thai
									}

									// Convert 2-letter code to 3-letter code if a mapping exists
									if threeLetterCode, exists := langCodeMap[twoLetterCode]; exists {
										langCode = threeLetterCode
										fyne.Do(func() {
											result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code for OCR: %s -> %s", twoLetterCode, langCode))
										})
									} else {
										// If no mapping exists, use the 2-letter code directly
										langCode = twoLetterCode
										fyne.Do(func() {
											result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code as-is for OCR: %s", langCode))
										})
									}
								}
							}
						}

						// Define the path to the trained data file with the selected language
						trainedDataPath := filepath.Join(filepath.Dir(pgsToSrtScript), "tessdata_fast", langCode+".traineddata")

						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempPgsFile)
						absOutputPath := filepath.Join(outDir, outFile)

						// Check if the script exists
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n\n[DEBUG] Checking if script exists at: %s", pgsToSrtScript))
						})

						if _, statErr := os.Stat(pgsToSrtScript); statErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Script NOT found: %v", statErr))
							})
							return
						}

						fyne.Do(func() {
							result.SetText(result.Text + "\n[DEBUG] Script found!")
						})

						// Test if Deno is working correctly
						fyne.Do(func() {
							result.SetText(result.Text + "\n[DEBUG] Running Deno version test...")
						})
						testCmd := exec.Command("deno", "--version")
						testOutput, testErr := testCmd.CombinedOutput()
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Deno Version Test ===\n")
							if testErr != nil {
								result.SetText(result.Text + fmt.Sprintf("Deno test error: %v\n", testErr))
							} else {
								result.SetText(result.Text + fmt.Sprintf("Deno version: %s\n", string(testOutput)))
							}
						})

						// Show detailed file information
						// Build text updates in memory before applying to UI
						textUpdate := fmt.Sprintf("\nInput SUP file: %s\nOutput SRT file: %s\nTessdata file: %s\n",
							absInputPath, absOutputPath, trainedDataPath)

						fyne.Do(func() {
							result.SetText(result.Text + textUpdate)

							// Check if input file exists and show size
							if fileInfo, err := os.Stat(absInputPath); err == nil {
								result.SetText(result.Text + fmt.Sprintf("Input file size: %d bytes\n", fileInfo.Size()))
							} else {
								result.SetText(result.Text + fmt.Sprintf("Input file check error: %v\n", err))
							}
						})

						// Variables to track file copy status
						var copyErr error
						var copySuccess bool

						// Create a temporary file for the output to avoid permission issues
						tmpOutputFile, tmpErr := os.CreateTemp("", "pgs_to_srt_*.srt")
						if tmpErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create temporary file: %v", tmpErr))
							})
							return
						}
						tmpOutputPath := tmpOutputFile.Name()
						tmpOutputFile.Close() // Close it so the script can write to it

						// Build and show the command - the script expects trained data path and input file, with output redirected
						cmdStr := fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"", pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
						// Combine text updates to reduce UI operations
						updateText := fmt.Sprintf("\n\n=== Executing Command ===\n%s\n\nConversion started at: %s\n",
							cmdStr, time.Now().Format("15:04:05"))

						fyne.Do(func() {
							result.SetText(result.Text + updateText)
						})

						// Create a log file for real-time monitoring of the PGS to SRT conversion process
						logFileName := filepath.Join(outDir, fmt.Sprintf("%s.track%d_%s.conversion.log", mkvBaseName, t.Num, t.Lang))
						logFile, logErr := os.Create(logFileName)

						// Create a logger that will be used throughout this function
						var logger *log.Logger

						if logErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create log file: %v", logErr))
							})
						} else {
							defer logFile.Close()
							logger = log.New(logFile, "", log.LstdFlags)
							logger.Printf("=== PGS to SRT Conversion Log ===\n")
							logger.Printf("Started at: %s\n", time.Now().Format("15:04:05"))
							logger.Printf("Input file: %s\n", absInputPath)
							logger.Printf("Final output file: %s\n", absOutputPath)
							logger.Printf("Temporary output file: %s\n", tmpOutputPath)
							logger.Printf("Script: %s\n", pgsToSrtScript)
							logger.Printf("Trained data: %s\n", trainedDataPath)
							logger.Printf("Working directory: %s\n", filepath.Dir(pgsToSrtScript))
							logger.Printf("PATH: %s\n\n", os.Getenv("PATH"))

							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n📝 Created log file: %s", logFileName))
								result.SetText(result.Text + fmt.Sprintf("\n📂 Using temporary file: %s", tmpOutputPath))
							})
						}

						// Run the conversion tool with Deno - using shell to enable output redirection
						cmd = exec.Command("sh", "-c", fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"",
							pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))

						// Set the working directory to ensure relative paths work correctly
						cmd.Dir = filepath.Dir(pgsToSrtScript)

						// Print the environment and command for debugging
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Environment ===\n")
							result.SetText(result.Text + fmt.Sprintf("Working directory: %s\n", cmd.Dir))
							result.SetText(result.Text + fmt.Sprintf("PATH: %s\n", os.Getenv("PATH")))
							result.SetText(result.Text + "\n=== Command ===\n")
							result.SetText(result.Text + fmt.Sprintf("deno run --allow-read --allow-write %s %s %s > %s\n",
								pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
						})

						// Set up pipes to capture output in real-time
						stdoutPipe, _ := cmd.StdoutPipe()
						stderrPipe, _ := cmd.StderrPipe()

						// Start the command
						startErr := cmd.Start()
						if startErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n❌ Failed to start command: %v", startErr))
							})
							if logFile != nil && logger != nil {
								logger.Printf("Failed to start command: %v\n", startErr)
							}
							err = startErr
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + "\n\n=== Starting Conversion Process ===\n")
								result.SetText(result.Text + "Check the log file for real-time output\n")
							})

							// Create a multi-writer to write to both the log file and capture the output
							var outputBuffer strings.Builder
							var stdoutWriter, stderrWriter io.Writer
							if logFile != nil && logger != nil {
								stdoutWriter = io.MultiWriter(logFile, &outputBuffer)
								stderrWriter = io.MultiWriter(logFile, &outputBuffer)
								logger.Printf("Command started successfully\n")
							} else {
								stdoutWriter = &outputBuffer
								stderrWriter = &outputBuffer
							}

							// Regular expressions to extract progress information from the output
							frameProgressRegex := regexp.MustCompile(`Processing frame (\d+)/(\d+)`)
							statusUpdateRegex := regexp.MustCompile(`Status: (.+)`)

							// Copy stdout and stderr to the writers in a buffered way to reduce UI updates
							go func() {
								bufReader := bufio.NewReaderSize(stdoutPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"

									// Check for progress information in the output
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Extract current frame and total frames
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
										if progressData.totalFrames == 0 {
											progressData.totalFrames = totalFrames
										}

										// Calculate frame rate
										if progressData.currentFrame > 0 {
											timeDiff := time.Since(progressData.lastUpdate).Seconds()
											frameDiff := currentFrame - progressData.currentFrame
											if timeDiff > 0 && frameDiff > 0 {
												// Smooth the frame rate calculation with a weighted average
												newFrameRate := float64(frameDiff) / timeDiff
												if progressData.frameRate > 0 {
													// 70% old rate, 30% new rate for smoother estimates
													progressData.frameRate = progressData.frameRate*0.7 + newFrameRate*0.3
												} else {
													progressData.frameRate = newFrameRate
												}
											}
										}

										progressData.currentFrame = currentFrame
										progressData.lastUpdate = time.Now()
										progressMutex.Unlock()

										// Update status label
										percentComplete := float64(currentFrame) / float64(totalFrames) * 100
										fyne.Do(func() {
											statusLabel.SetText(fmt.Sprintf("Processing frame %d of %d (%.1f%%)",
												currentFrame, totalFrames, percentComplete))
										})
									} else if matches := statusUpdateRegex.FindStringSubmatch(line); len(matches) == 2 {
										// Update status message
										statusMsg := matches[1]
										fyne.Do(func() {
											statusLabel.SetText(statusMsg)
										})
									}

									if _, writeErr := stdoutWriter.Write([]byte(line)); writeErr != nil {
										break
									}
								}
							}()

							go func() {
								bufReader := bufio.NewReaderSize(stderrPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"

									// Also check stderr for progress information
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Process frame progress from stderr (same as stdout handler)
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
										if progressData.totalFrames == 0 {
											progressData.totalFrames = totalFrames
										}
										progressData.currentFrame = currentFrame
										progressMutex.Unlock()
									}

									if _, writeErr := stderrWriter.Write([]byte(line)); writeErr != nil {
										break
									}
								}
							}()

							// Wait for the command to complete
							err = cmd.Wait()
							output = []byte(outputBuffer.String())

							// Log the completion status
							if logFile != nil && logger != nil {
								if err != nil {
									logger.Printf("\n\nCommand completed with error: %v\n", err)
								} else {
									logger.Printf("\n\nCommand completed successfully\n")
								}
								logger.Printf("Finished at: %s\n", time.Now().Format("15:04:05"))
							}

							// Copy the temporary file to the final destination regardless of command success/failure
							// This allows us to potentially recover partial conversions even if the command had issues

							// Check if the temporary file exists before attempting to copy
							if _, statErr := os.Stat(tmpOutputPath); statErr == nil {
								if logFile != nil && logger != nil {
									logger.Printf("Copying temporary file %s to final destination %s\n", tmpOutputPath, absOutputPath)
								}

								// Create the parent directory for the output file if it doesn't exist
								outputDir := filepath.Dir(absOutputPath)
								if mkdirErr := os.MkdirAll(outputDir, 0755); mkdirErr != nil {
									copyErr = fmt.Errorf("failed to create output directory: %v", mkdirErr)
									if logFile != nil && logger != nil {
										logger.Printf("Error creating output directory: %v\n", mkdirErr)
									}
								} else {
									// Read the temporary file
									tmpContent, readErr := os.ReadFile(tmpOutputPath)
									if readErr != nil {
										copyErr = fmt.Errorf("failed to read temporary file: %v", readErr)
										if logFile != nil && logger != nil {
											logger.Printf("Error reading temporary file: %v\n", readErr)
										}
									} else {
										// Write to the final destination
										writeErr := os.WriteFile(absOutputPath, tmpContent, 0644)
										if writeErr != nil {
											copyErr = fmt.Errorf("failed to write to final destination: %v", writeErr)
											if logFile != nil && logger != nil {
												logger.Printf("Error writing to final destination: %v\n", writeErr)
											}
										} else {
											copySuccess = true
											if logFile != nil && logger != nil {
												logger.Printf("Successfully copied temporary file to final destination\n")
											}

											// Clean up the temporary file
											removeErr := os.Remove(tmpOutputPath)
											if removeErr != nil && logFile != nil && logger != nil {
												logger.Printf("Warning: Could not remove temporary file: %v\n", removeErr)
											} else if logFile != nil && logger != nil {
												logger.Printf("Removed temporary file\n")
											}
										}
									}
								}
							} else {
								copyErr = fmt.Errorf("temporary file not found: %v", statErr)
								if logFile != nil && logger != nil {
									logger.Printf("Error: Temporary file not found: %v\n", statErr)
								}
							}

							// If the command succeeded but copy failed, update the error
							if err == nil && copyErr != nil {
								err = copyErr
							}
						}

						// Prepare output text in memory before updating UI
						var outputText strings.Builder
						outputText.WriteString("\nFull command output:\n")

						// Limit output size to prevent UI sluggishness with very large outputs
						outputStr := string(output)
						const maxOutputLen = 10000 // Limit output to 10K chars
						if len(outputStr) > maxOutputLen {
							outputText.WriteString(outputStr[:maxOutputLen])
							outputText.WriteString("\n... [Output truncated, full output in log file] ...")
						} else {
							outputText.WriteString(outputStr)
						}

						// Add error message if needed
						if err != nil {
							outputText.WriteString("\n\n❌ Command error: " + err.Error())
						}

						// Update UI in a single operation
						fyne.Do(func() {
							result.SetText(result.Text + outputText.String())
						})

						// Show output
						fyne.Do(func() {
							// Calculate total conversion time
							conversionTime := time.Since(conversionStartTime).Round(time.Second)

							// Update status based on success or failure
							if err != nil {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion failed after %s", conversionTime))
							} else {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion completed in %s", conversionTime))
							}
							progress.Show()

							// Stop the ticker by removing the spinner container
							// Find and remove the conversion spinner container
							for i, obj := range trackList.Objects {
								if box, ok := obj.(*fyne.Container); ok {
									for _, child := range box.Objects {
										if label, ok := child.(*widget.Label); ok && label.Text == "Converting PGS to SRT..." {
											trackList.Objects = append(trackList.Objects[:i], trackList.Objects[i+1:]...)
											break
										}
									}
								}
							}
							trackList.Refresh()

							result.SetText(result.Text + "\n\n=== Conversion Results ===\n")
							result.SetText(result.Text + "Completed at: " + time.Now().Format("15:04:05") + "\n")

							// Always show the full output for better debugging
							outputStr := string(output)
							result.SetText(result.Text + "\nFull output: \n" + outputStr + "\n")

							if err != nil {
								result.SetText(result.Text + "\n❌ Error: " + err.Error() + "\n")
							} else {
								result.SetText(result.Text + "\n✅ Command completed successfully\n")

								// Show file copy operation status
								result.SetText(result.Text + "\n=== File Operations ===\n")
								result.SetText(result.Text + fmt.Sprintf("✓ Temporary file created: %s\n", tmpOutputPath))
								if copySuccess {
									result.SetText(result.Text + fmt.Sprintf("✓ Copied to final destination: %s\n", absOutputPath))
									result.SetText(result.Text + "✓ Temporary file cleaned up\n")
								} else if copyErr != nil {
									result.SetText(result.Text + fmt.Sprintf("❌ Failed to copy to final destination: %v\n", copyErr))
								}
							}

							// Ensure the text area scrolls to the bottom to show the latest output
							// No need to set cursor position for Label widget
						})

						// Check current directory for debugging
						currentDir, _ := os.Getwd()
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Path Debugging ===\n")
							result.SetText(result.Text + fmt.Sprintf("Current working directory: %s\n", currentDir))
							result.SetText(result.Text + fmt.Sprintf("Looking for output file at: %s\n", absOutputPath))
						})

						// List files in output directory to see what was created
						files, _ := os.ReadDir(outDir)
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nFiles in output directory (%s):\n", outDir))
							for _, file := range files {
								result.SetText(result.Text + fmt.Sprintf("- %s\n", file.Name()))
							}
						})

						// Check if SRT file was created and show details
						if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
							fyne.Do(func() {
								result.SetText(result.Text + "\n✅ SRT file created successfully!")
								result.SetText(result.Text + fmt.Sprintf("\n   - Path: %s", absOutputPath))
								result.SetText(result.Text + fmt.Sprintf("\n   - Size: %d bytes", fileInfo.Size()))
								result.SetText(result.Text + fmt.Sprintf("\n   - Modified: %s", fileInfo.ModTime().Format("15:04:05")))

								// Try to count lines in SRT file
								if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
									lines := strings.Split(string(srtContent), "\n")
									result.SetText(result.Text + fmt.Sprintf("\n   - Lines: %d", len(lines)))

									// Count subtitle entries (every 4 lines is typically one subtitle)
									subtitleCount := (len(lines) + 3) / 4 // rough estimate
									result.SetText(result.Text + fmt.Sprintf("\n   - Estimated subtitles: ~%d", subtitleCount))
								}
							})
						} else {
							err = fmt.Errorf("SRT file was not created: %v", statErr)
							fyne.Do(func() {
								result.SetText(result.Text + "\n❌ Error: " + err.Error())
							})
						}
					}
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") || strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station")) {
					// ASS/SSA to SRT conversion
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting ASS/SSA to SRT conversion process")
					})
					tempAssFile := fmt.Sprintf("%s.track%d_%s.ass", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absAssPath := filepath.Join(outDir, tempAssFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting ASS/SSA track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== ASS/SSA Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("ASS/SSA file: %s\n", tempAssFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absAssPath))
					})

					// Extract ASS/SSA first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempAssFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})

					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
					cmd.Dir = outDir

					// Run the command and capture output
					output, err = cmd.CombinedOutput()

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					assFilePath := filepath.Join(outDir, tempAssFile)
					fileInfo, statErr := os.Stat(assFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty (0 bytes)")
					} else {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted ASS/SSA file (%d bytes)", fileInfo.Size()))
						})
					}

					if err == nil {
						// Create a progress bar for the conversion process
						conversionProgress := widget.NewProgressBar()
						conversionProgress.Min = 0
						conversionProgress.Max = 100
						conversionProgress.SetValue(0)

						conversionLabel := widget.NewLabel("Converting ASS/SSA to SRT...")
						statusLabel := widget.NewLabel("Processing ASS/SSA file...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Converting...")

						// Track conversion start time
						conversionStartTime := time.Now()

						// Create a ticker to update elapsed time
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastElapsedText string

							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

								// Only update UI if text has changed
								if newElapsedText != lastElapsedText {
									lastElapsedText = newElapsedText
									fyne.Do(func() {
										elapsedLabel.SetText(newElapsedText)
										conversionProgress.SetValue(50) // Simple indeterminate progress
									})
								}
							}
						}()

						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] ASS/SSA extraction completed successfully, starting conversion process")

							// Show the conversion progress bar and labels
							currentTrackLabel.SetText("Converting ASS/SSA to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
							))
							trackList.Refresh()
						})

						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempAssFile)
						absOutputPath := filepath.Join(outDir, outFile)

						// Use ffmpeg to convert ASS/SSA to SRT
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] Using ffmpeg to convert ASS/SSA to SRT")
							statusLabel.SetText("Running ffmpeg conversion...")
						})

						// Get ffmpeg path - prioritize Homebrew version
						ffmpegPath := "ffmpeg" // Default fallback path

						// First check Homebrew path (preferred)
						homebrewPath := "/opt/homebrew/bin/ffmpeg"
						if _, err := os.Stat(homebrewPath); err == nil {
							ffmpegPath = homebrewPath
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] Using Homebrew ffmpeg: " + homebrewPath)
							})
						} else {
							// If Homebrew not found, check Miniconda as fallback
							homeDir, err := os.UserHomeDir()
							if err == nil {
								minicondaPath := filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg")
								if _, err := os.Stat(minicondaPath); err == nil {
									ffmpegPath = minicondaPath
									fyne.Do(func() {
										result.SetText(result.Text + "\n[DEBUG] Using Miniconda ffmpeg: " + minicondaPath)
									})
								}
							}
						}

						// Create the ffmpeg command with the appropriate path
						cmd = exec.Command(ffmpegPath, "-i", absInputPath, "-f", "srt", absOutputPath)
						cmd.Dir = outDir

						// Run the command and capture output
						output, err = cmd.CombinedOutput()

						// Stop the ticker
						ticker.Stop()

						// Update UI with results
						fyne.Do(func() {
							result.SetText(result.Text + "\nffmpeg output: " + string(output))

							if err != nil {
								result.SetText(result.Text + "\nError converting ASS/SSA to SRT: " + err.Error())
								statusLabel.SetText("Conversion failed!")
								conversionProgress.SetValue(0)
							} else {
								result.SetText(result.Text + "\nSuccessfully converted ASS/SSA to SRT")
								statusLabel.SetText("Conversion completed!")
								conversionProgress.SetValue(100)

								// Check if the output file was created
								if _, statErr := os.Stat(absOutputPath); statErr == nil {
									result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
								} else {
									result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
								}
							}

							// Update elapsed time one last time
							elapsed := time.Since(conversionStartTime).Round(time.Second)
							elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
							remainingLabel.SetText("Completed")
						})
					}
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "vobsub" || t.Codec == "VobSub") {
					// VobSub to SRT conversion
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting VobSub to SRT conversion process")
					})

					// For VobSub, we extract both .idx and .sub files
					// The .idx file is the main file that contains timing and positioning information
					// The .sub file contains the actual subtitle images
					idxFile := fmt.Sprintf("%s.track%d_%s.idx", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absIdxPath := filepath.Join(outDir, idxFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting VobSub track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== VobSub Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("IDX file: %s\n", idxFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absIdxPath))
					})

					// Extract VobSub first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, idxFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})

					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
					cmd.Dir = outDir

					// Run the command and capture output
					output, err = cmd.CombinedOutput()

					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})

					// Check if the file was created and has content
					idxFilePath := filepath.Join(outDir, idxFile)
					fileInfo, statErr := os.Stat(idxFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty")
					} else {
						// File exists and has content, proceed with conversion
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nIDX file extracted successfully (%d bytes)", fileInfo.Size()))
							result.SetText(result.Text + "\n\n=== VobSub to SRT Conversion ===\n")
						})

						// Create UI elements for conversion progress
						conversionStartTime := time.Now()
						conversionLabel := widget.NewLabel("Converting VobSub to SRT...")
						statusLabel := widget.NewLabel("Starting conversion...")
						conversionProgress := widget.NewProgressBar()
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Estimating...")

						// Start a ticker to update the elapsed time
						ticker := time.NewTicker(time.Second)
						go func() {
							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								fyne.Do(func() {
									elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
								})
							}
						}()

						// Show the conversion progress bar and labels
						fyne.Do(func() {
							currentTrackLabel.SetText("Converting VobSub to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
							))
							trackList.Refresh()
						})

						// Get absolute paths for input and output
						// For vobsub2srt, we need the base path without extension
						basePath := strings.TrimSuffix(idxFilePath, filepath.Ext(idxFilePath))
						absOutputPath := basePath + ".srt" // vobsub2srt will create this file

						// Check if both .idx and .sub files exist
						idxFile := basePath + ".idx"
						subFile := basePath + ".sub"

						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for IDX file: %s", idxFile))
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for SUB file: %s", subFile))
						})

						// Check if the files exist
						var filesExist bool = true
						if _, err := os.Stat(idxFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file exists: %s", idxFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file does not exist: %s - %v", idxFile, err))
							})
						}

						if _, err := os.Stat(subFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file exists: %s", subFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file does not exist: %s - %v", subFile, err))
							})
						}

						// If either file is missing, show a warning
						if !filesExist {
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] ⚠️ Warning: IDX or SUB file is missing, conversion may fail")
							})
						}

						// Get language from user selection or use track language as default
						langCode := t.Lang
						if langCode == "" {
							langCode = "eng" // Default to English if no language code is available
						}

						// Check if user has selected a specific language
						if t.LangSelect != nil && t.LangSelect.Selected != "" && !strings.HasPrefix(t.LangSelect.Selected, "Auto") {
							// Extract the language code from the selection (format: "Language (code)")
							selection := t.LangSelect.Selected
							// Extract the code part between parentheses
							if start := strings.LastIndex(selection, "("); start != -1 {
								if end := strings.LastIndex(selection, ")"); end != -1 && end > start {
									// Extract the 2-letter code directly
									twoLetterCode := selection[start+1 : end]
									fyne.Do(func() {
										result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] User selected language: %s (code: %s)", selection, twoLetterCode))
									})
									langCode = twoLetterCode
								}
							}
						} else {
							// Using auto-detected language, map 3-letter code to 2-letter code
							langCodeMap := map[string]string{
								"eng": "en", // English
								"fre": "fr", // French
								"fra": "fr", // French (alternate)
								"ger": "de", // German
								"deu": "de", // German (alternate)
								"ita": "it", // Italian
								"spa": "es", // Spanish
								"por": "pt", // Portuguese
								"dut": "nl", // Dutch
								"nld": "nl", // Dutch (alternate)
								"swe": "sv", // Swedish
								"nor": "no", // Norwegian
								"dan": "da", // Danish
								"fin": "fi", // Finnish
								"jpn": "ja", // Japanese
								"kor": "ko", // Korean
								"chi": "zh", // Chinese
								"zho": "zh", // Chinese (alternate)
								"rus": "ru", // Russian
								"pol": "pl", // Polish
								"cze": "cs", // Czech
								"ces": "cs", // Czech (alternate)
								"hun": "hu", // Hungarian
								"gre": "el", // Greek
								"ell": "el", // Greek (alternate)
								"tur": "tr", // Turkish
								"ara": "ar", // Arabic
								"heb": "he", // Hebrew
								"tha": "th", // Thai
							}

							// Convert 3-letter code to 2-letter code if a mapping exists
							if twoLetterCode, exists := langCodeMap[strings.ToLower(langCode)]; exists {
								fyne.Do(func() {
									// Use bold formatting for important debug information to improve readability
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code: ** %s -> %s **", langCode, twoLetterCode))
								})
								langCode = twoLetterCode
							} else {
								fyne.Do(func() {
									// Use bold formatting for important debug information to improve readability
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] No mapping found for language code: ** %s **, using as-is", langCode))
								})
							}
						}

						// Use vobsub2srt binary for conversion
						conversionScript := "/usr/local/bin/vobsub2srt"

						// Check if the binary exists
						if _, err := os.Stat(conversionScript); err != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[ERROR] vobsub2srt binary not found at %s", conversionScript))
							})
							err = fmt.Errorf("vobsub2srt binary not found at %s", conversionScript)
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using vobsub2srt binary: %s", conversionScript))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code: %s for VobSub conversion", langCode))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Base path for vobsub2srt: %s", basePath))
							})

							// Check if the output SRT file already exists and delete it if it does
							outputSrtFile := basePath + ".srt"
							if _, err := os.Stat(outputSrtFile); err == nil {
								fyne.Do(func() {
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Removing existing SRT file: %s", outputSrtFile))
								})
								os.Remove(outputSrtFile)
							}

							// Run vobsub2srt with the language parameter
							cmdStr = fmt.Sprintf("%s --lang %s \"%s\"", conversionScript, langCode, basePath)
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] Running command: " + cmdStr)
								statusLabel.SetText("Running vobsub2srt conversion...")
							})

							// Create the command
							cmd = exec.Command(conversionScript, "--lang", langCode, basePath)
							cmd.Dir = outDir

							// Run the command and capture output
							output, err = cmd.CombinedOutput()

							// Stop the ticker
							ticker.Stop()

							// Update UI with results
							fyne.Do(func() {
								result.SetText(result.Text + "\nvobsub2srt output: " + string(output))

								if err != nil {
									result.SetText(result.Text + "\nError converting VobSub to SRT: " + err.Error())
									statusLabel.SetText("Conversion failed!")
									conversionProgress.SetValue(0)
								} else {
									result.SetText(result.Text + "\nSuccessfully ran vobsub2srt command")
									statusLabel.SetText("Conversion completed!")
									conversionProgress.SetValue(100)

									// Check if the output file was created
									if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
										result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
										result.SetText(result.Text + fmt.Sprintf("\nSRT file size: %d bytes", fileInfo.Size()))

										// Try to count lines in SRT file
										if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
											lines := strings.Split(string(srtContent), "\n")
											result.SetText(result.Text + fmt.Sprintf("\nSRT file lines: %d", len(lines)))

											// Count subtitle entries (every 4 lines is typically one subtitle)
											subtitleCount := (len(lines) + 3) / 4 // rough estimate
											result.SetText(result.Text + fmt.Sprintf("\nEstimated subtitles: ~%d", subtitleCount))
										}
									} else {
										result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
									}
								}

								// Update elapsed time one last time
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
								remainingLabel.SetText("Completed")
							})
						}
					}
				} else {
					// Normal extraction without conversion
					// Use proper file extension based on codec
					var fileExt string

					// Handle special case for "SubRip/SRT" format
					if strings.Contains(t.Codec, "SubRip") || strings.Contains(t.Codec, "subrip") || strings.Contains(t.Codec, "SRT") || strings.Contains(t.Codec, "srt") {
						fileExt = "srt"
						fyne.Do(func() {
							result.SetText(result.Text + "\nDetected SRT format, using .srt extension")
						})
					} else if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" {
						fileExt = "sup"
					} else if t.Codec == "ass" || t.Codec == "ssa" || t.Codec == "ASS" || t.Codec == "SSA" {
						fileExt = "ass"
					} else if t.Codec == "vobsub" || t.Codec == "VobSub" {
						fileExt = "idx"
					} else {
						// Use lowercase codec name as fallback but remove any slashes
						cleanCodec := strings.ReplaceAll(t.Codec, "/", "_")
						fileExt = strings.ToLower(cleanCodec)
					}

					// Debug output for file naming
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n=== Track Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s - %s)\n", t.Num, t.Lang, t.Codec))
					})

					outFile = fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, t.Num, t.Lang, fileExt)

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("Output file: %s\n", outFile))
					})
					// Use absolute paths for all subtitle extractions to avoid directory creation issues
					absOutFile := filepath.Join(outDir, outFile)
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\nExtracting to: %s", absOutFile))
					})

					output, err = cmd.CombinedOutput()

					// Set proper file permissions for subtitle files (read/write for user, read for group/others)
					if err == nil {
						outFilePath := filepath.Join(outDir, outFile)
						os.Chmod(outFilePath, 0644) // rw-r--r--
					}
				}

				// Update UI on main thread
				fyne.Do(func() {
					if err != nil {
						t.State = "Error"
						t.Status.SetText(fmt.Sprintf("[!] Track %d: %s (%s) %s - Error", t.Num, t.Lang, t.Codec, t.Name))
						result.SetText(string(output) + "\nExtraction failed: " + err.Error())
					} else {
						t.State = "Done"
						t.Status.SetText(fmt.Sprintf("[✓] Track %d: %s (%s) %s - Done", t.Num, t.Lang, t.Codec, t.Name))
						progress.SetValue(float64(tracksDone + 1))
					}

					// Update track list
					trackList.Objects = nil
					for _, tt := range trackItems {
						trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", tt.Num, tt.Lang, tt.Codec, tt.Name))

						if tt.ConvertOCR != nil {
							// For PGS subtitles, show OCR option
							ocrLabel := widget.NewLabel("Convert to SRT")
							row := container.NewHBox(tt.Check, tt.Status, trackInfo, tt.ConvertOCR, ocrLabel)
							trackList.Add(row)
						} else {
							// For other subtitle formats
							row := container.NewHBox(tt.Check, tt.Status, trackInfo)
							trackList.Add(row)
						}
					}
					trackList.Refresh()
				})

				tracksDone++
			}

			// Final UI update on main thread
			fyne.Do(func() {
				currentTrackLabel.SetText("")
				if tracksDone == len(selected) {
					result.SetText("Extraction complete!")
					progress.SetValue(progress.Max)
				} else {
					result.SetText(fmt.Sprintf("Extraction stopped after %d of %d tracks", tracksDone, len(selected)))
				}
			})
		}()
	})

	// Create Support button with improved UX
	supportBtn := widget.NewButton("Donate ☕", func() {
		// Show a confirmation dialog with information about the donation
		confirm := dialog.NewConfirm(
			"Support Subtitle Forge",
			"Your donation helps maintain and improve Subtitle Forge. Would you like to proceed to PayPal?",
			func(ok bool) {
				if ok {
					supportURL, _ := url.Parse("https://paypal.me/VenimK")
					fyne.CurrentApp().OpenURL(supportURL)
				}
			},
			w,
		)
		confirm.SetDismissText("Cancel")
		confirm.SetConfirmText("Donate")
		confirm.Show()
	})
	supportBtn.Importance = widget.HighImportance

	// Create button row for better layout
	buttonRow := container.NewHBox(loadTracksBtn, startExtractBtn, layout.NewSpacer(), supportBtn)

	// Setup keyboard shortcuts for main actions
	setupKeyboardShortcuts(fileBtn.OnTapped, dirBtn.OnTapped, loadTracksBtn.OnTapped, startExtractBtn.OnTapped)

	// Use app.NewWithID for better performance and to avoid preferences API warnings
	// This was already set at the beginning of main()

	// Use a more efficient layout with container.NewBorder for better performance
	// Create app title with version
	titleLabel := widget.NewLabel("Subtitle Forge v1.6.5")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	topContent := container.NewVBox(
		titleLabel,
		fileBtn,
		selectedFile,
		dirBtn,
		selectedDir,
		buttonRow,
		currentTrackLabel,
		progress,
	)

	// Track control buttons (select/deselect all)
	selectAllBtn := widget.NewButton("Select All", func() {
		for _, t := range trackItems {
			t.Check.SetChecked(true)
		}
	})

	deselectAllBtn := widget.NewButton("Deselect All", func() {
		for _, t := range trackItems {
			t.Check.SetChecked(false)
		}
	})

	// Track filter
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter tracks by language, codec, or name...")

	// Function to filter tracks based on search text
	filterTracks := func(filterText string) {
		// Clear the track list UI
		trackList.Objects = nil

		// If no filter, show all tracks
		if filterText == "" {
			for _, t := range trackItems {
				// Create row for this track
				trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name))

				var row *fyne.Container
				if t.ConvertOCR != nil {
					// For PGS/VobSub subtitles, show OCR option and language selection
					ocrLabel := widget.NewLabel("Convert to SRT")

					if t.LangSelect != nil {
						// Add language selection dropdown for OCR-based conversion
						langLabel := widget.NewLabel("OCR Language:")
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
					} else {
						// For ASS/SSA conversion (no OCR language needed)
						row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
					}
				} else {
					// For other subtitle formats
					row = container.NewHBox(t.Check, t.Status, trackInfo)
				}

				trackList.Add(row)
			}
		} else {
			// Convert filter text to lowercase for case-insensitive comparison
			lowerFilter := strings.ToLower(filterText)

			// Add only tracks that match the filter
			for _, t := range trackItems {
				// Check if the track matches the filter criteria
				matchesFilter := strings.Contains(strings.ToLower(t.Lang), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Codec), lowerFilter) ||
					strings.Contains(strings.ToLower(t.Name), lowerFilter) ||
					strings.Contains(strings.ToLower(fmt.Sprintf("Track %d", t.Num)), lowerFilter)

				if matchesFilter {
					// Create row for this track
					trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", t.Num, t.Lang, t.Codec, t.Name))

					var row *fyne.Container
					if t.ConvertOCR != nil {
						// For PGS/VobSub subtitles, show OCR option and language selection
						ocrLabel := widget.NewLabel("Convert to SRT")

						if t.LangSelect != nil {
							// Add language selection dropdown for OCR-based conversion
							langLabel := widget.NewLabel("OCR Language:")
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel, langLabel, t.LangSelect)
						} else {
							// For ASS/SSA conversion (no OCR language needed)
							row = container.NewHBox(t.Check, t.Status, trackInfo, t.ConvertOCR, ocrLabel)
						}
					} else {
						// For other subtitle formats
						row = container.NewHBox(t.Check, t.Status, trackInfo)
					}

					trackList.Add(row)
				}
			}
		}

		trackList.Refresh()
	}

	// Set up filter entry change handler
	filterEntry.OnChanged = func(text string) {
		filterTracks(text)
	}

	// Track control container with buttons and filter
	// Make the filter entry take more space by setting its placeholder to be longer
	filterEntry.SetPlaceHolder("Filter tracks by language, codec, name, or track number...                                                 ")

	// Using a grid layout to give the filter entry more space
	filterBox := container.New(
		layout.NewFormLayout(),
		widget.NewLabel("Filter:"),
		filterEntry,
	)

	trackControlsContainer := container.NewVBox(
		container.NewHBox(selectAllBtn, deselectAllBtn),
		filterBox,
	)

	middleContent := container.NewVBox(
		widget.NewLabel("Subtitle Tracks:"),
		trackControlsContainer,
		trackListScroll,
	)

	bottomContent := container.NewVBox(
		widget.NewLabel("Results:"),
		resultScroll,
		dependencyButtons,
	)

	// Create tab for subtitle extraction (existing functionality)
	extractTabContent := container.NewBorder(
		topContent,
		bottomContent,
		nil,
		nil,
		middleContent,
	)

	// Create tab for subtitle insertion
	// Create file selection widgets for subtitle insertion
	insertMkvFileLabel := widget.NewLabel("No MKV file selected")
	insertSrtFileLabel := widget.NewLabel("No SRT file selected")

	selectInsertMkvBtn := widget.NewButton("Select MKV File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".mkv") {
				dialog.ShowInformation("Invalid File", "Please select an MKV file", w)
				return
			}

			insertMkvFileLabel.SetText(filePath)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mkv"}))
		fd.Show()
	})

	selectInsertSrtBtn := widget.NewButton("Select SRT File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			if !strings.HasSuffix(strings.ToLower(filePath), ".srt") {
				dialog.ShowInformation("Invalid File", "Please select an SRT file", w)
				return
			}

			insertSrtFileLabel.SetText(filePath)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".srt"}))
		fd.Show()
	})

	// Create language selection for subtitle insertion
	// Define common languages with their 3-letter ISO codes
	languages := map[string]string{
		"English":    "eng",
		"Spanish":    "spa",
		"French":     "fre",
		"German":     "ger",
		"Italian":    "ita",
		"Japanese":   "jpn",
		"Korean":     "kor",
		"Chinese":    "chi",
		"Russian":    "rus",
		"Portuguese": "por",
		"Arabic":     "ara",
		"Hindi":      "hin",
		"Dutch":      "dut",
		"Swedish":    "swe",
		"Polish":     "pol",
		"Turkish":    "tur",
		"Czech":      "cze",
		"Greek":      "gre",
		"Hungarian":  "hun",
		"Finnish":    "fin",
		"Danish":     "dan",
		"Norwegian":  "nor",
		"Romanian":   "rum",
		"Thai":       "tha",
		"Vietnamese": "vie",
		"Bulgarian":  "bul",
		"Croatian":   "hrv",
		"Slovak":     "slo",
		"Slovenian":  "slv",
		"Ukrainian":  "ukr",
	}

	// Define common language codes for dropdown
	langCodes := []string{
		"eng", "spa", "fre", "ger", "ita", "jpn", "kor", "chi", "rus", "por",
		"ara", "hin", "dut", "swe", "pol", "tur", "cze", "gre", "hun", "fin",
		"dan", "nor", "rum", "tha", "vie", "bul", "hrv", "slo", "slv", "ukr",
		"alb", "amh", "aze", "ben", "bos", "cat", "est", "fil", "glg", "geo",
		"heb", "ice", "ind", "kan", "kaz", "khm", "lao", "lat", "lit",
		"mac", "mal", "mar", "mon", "nep", "per", "srp", "swa", "tam", "tel",
		"tgl", "urd", "uzb", "wel", "yid", "zul",
	}

	// Create sorted list of language names for dropdown
	langNames := make([]string, 0, len(languages))
	for name := range languages {
		langNames = append(langNames, name)
	}
	sort.Strings(langNames)

	// Add "Custom" option at the end
	langNames = append(langNames, "Custom")

	// Create language dropdown
	selectedLang := "English"
	langDropdown := widget.NewSelect(langNames, func(selected string) {
		selectedLang = selected
	})
	langDropdown.SetSelected("English")

	// Create custom language code dropdown with improved readability
	selectedLangCode := "eng"
	
	// Create the dropdown with explicit text color
	customLangDropdown := widget.NewSelect(langCodes, func(selected string) {
		selectedLangCode = selected
	})
	customLangDropdown.SetSelected("eng")
	
	// Create a high-contrast container for the dropdown
	padded := container.NewPadded(customLangDropdown)
	langCodeContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the dropdown directly
		padded,
	)
	
	// Create a card with the high-contrast container
	langCodeCard := widget.NewCard("", "", langCodeContainer)
	
	// Initially hide both elements
	customLangDropdown.Hide()
	langCodeCard.Hide()

	// Create track name entry
	trackNameEntry := widget.NewEntry()
	trackNameEntry.SetPlaceHolder("English")
	trackNameEntry.SetText("English")

	// Create result label for subtitle insertion
	insertResultLabel := widget.NewLabel("")
	insertResultScroll := container.NewScroll(insertResultLabel)
	insertResultScroll.SetMinSize(fyne.NewSize(800, 150))

	// Create default track options
	defaultTrack := widget.NewCheck("Set as default subtitle track", nil)
	defaultTrack.SetChecked(true)

	// Create forced track option
	forcedTrack := widget.NewCheck("Mark as forced subtitle track", nil)

	// Create option to remove other subtitle tracks
	removeOtherTracks := widget.NewCheck("Remove all other subtitle tracks", nil)

	// Create output file name options
	outputNameEntry := widget.NewEntry()
	outputNameEntry.SetPlaceHolder("Leave empty for auto naming")

	// Show language dropdown change handler
	langDropdown.OnChanged = func(selected string) {
		selectedLang = selected
		if selected == "Custom" {
			customLangDropdown.Show()
			langCodeCard.Show()
			// Don't auto-update track name for custom selection
		} else {
			customLangDropdown.Hide()
			langCodeCard.Hide()
			// Automatically select the corresponding language code
			if code, ok := languages[selected]; ok {
				// Find the matching code in langCodes
				for _, langCode := range langCodes {
					if langCode == code {
						customLangDropdown.SetSelected(langCode)
						selectedLangCode = langCode
						break
					}
				}

				// Auto-update track name to match selected language
				if trackNameEntry.Text == "" || trackNameEntry.Text == "English" ||
					containsLanguageName(trackNameEntry.Text, languages) {
					trackNameEntry.SetText(selected)
				}
			}
		}
	}

	// Create insert button
	insertSubtitleBtn := widget.NewButton("Insert Subtitle", func() {
		// Check if files are selected
		mkvPath := insertMkvFileLabel.Text
		srtPath := insertSrtFileLabel.Text

		if mkvPath == "No MKV file selected" || srtPath == "No SRT file selected" {
			dialog.ShowInformation("Missing Files", "Please select both MKV and SRT files", w)
			return
		}

		// Get language code based on selection
		var lang string
		if selectedLang == "Custom" {
			lang = selectedLangCode // Use the selected language code from dropdown
		} else {
			lang = languages[selectedLang]
		}

		// Get track name
		trackName := trackNameEntry.Text
		if trackName == "" {
			trackName = selectedLang // Use selected language name as default
		}

		// Create output file path
		dir := filepath.Dir(mkvPath)
		baseName := filepath.Base(mkvPath)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

		// Use custom output name if provided
		outputName := outputNameEntry.Text
		if outputName == "" {
			outputName = baseName + "_with_subtitles.mkv"
		} else if !strings.HasSuffix(strings.ToLower(outputName), ".mkv") {
			outputName = outputName + ".mkv"
		}

		outputPath := filepath.Join(dir, outputName)

		insertResultLabel.SetText("Adding subtitle to MKV file...\n")

		// Build mkvmerge command with options
		mkvmergeArgs := []string{
			"-o", outputPath,
		}

		// If removing other subtitle tracks is checked, use --no-subtitles option
		if removeOtherTracks.Checked {
			mkvmergeArgs = append(mkvmergeArgs, "--no-subtitles", mkvPath)
			insertResultLabel.SetText(insertResultLabel.Text + "\nRemoving all existing subtitle tracks...")
		} else {
			mkvmergeArgs = append(mkvmergeArgs, mkvPath)
		}

		// Add language and track name options for the SRT file
		mkvmergeArgs = append(mkvmergeArgs,
			"--language", "0:"+lang,
			"--track-name", "0:"+trackName,
		)

		// Add default track option if checked
		if defaultTrack.Checked {
			mkvmergeArgs = append(mkvmergeArgs, "--default-track", "0:yes")
		}

		// Add forced track option if checked
		if forcedTrack.Checked {
			mkvmergeArgs = append(mkvmergeArgs, "--forced-track", "0:yes")
		}

		// Add SRT file at the end
		mkvmergeArgs = append(mkvmergeArgs, srtPath)

		// Run mkvmerge command to add subtitle
		go func() {
			cmd := exec.Command("mkvmerge", mkvmergeArgs...)

			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					insertResultLabel.SetText(insertResultLabel.Text + "\nError: " + err.Error() + "\n" + string(output))
					return
				}

				insertResultLabel.SetText(insertResultLabel.Text + "\nSubtitle added successfully!\nOutput file: " + outputPath + "\n" + string(output))
			})
		}()
	})

	// Create layout for subtitle insertion tab
	insertTitleLabel := widget.NewLabelWithStyle("Insert Subtitles into MKV", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create visual drop areas (these are just for visual indication, actual drop handling is at window level)
	mkvDropArea := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	mkvDropLabel := widget.NewLabelWithStyle("Drop MKV File Here", fyne.TextAlignCenter, fyne.TextStyle{})
	mkvDropContainer := container.NewStack(
		mkvDropArea,
		mkvDropLabel,
	)
	mkvDropContainer.Resize(fyne.NewSize(300, 60))

	srtDropArea := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	srtDropLabel := widget.NewLabelWithStyle("Drop SRT File Here", fyne.TextAlignCenter, fyne.TextStyle{})
	srtDropContainer := container.NewStack(
		srtDropArea,
		srtDropLabel,
	)
	srtDropContainer.Resize(fyne.NewSize(300, 60))

	// Group file selection
	fileSelectionGroup := widget.NewCard("File Selection", "", container.NewVBox(
		container.NewHBox(selectInsertMkvBtn, insertMkvFileLabel),
		mkvDropContainer,
		container.NewHBox(selectInsertSrtBtn, insertSrtFileLabel),
		srtDropContainer,
	))

	// Group subtitle options
	// Set placeholders with extra spaces to make input fields wider
	trackNameEntry.SetPlaceHolder("Enter track name...                                                ")

	// Create section titles with explicit styling for guaranteed readability
	languageTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText("Language Settings", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	languageTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	languageTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	trackOptionsTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText("Track Options", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	trackOptionsTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	trackOptionsTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	// Create separator for visual distinction
	languageSeparator := widget.NewSeparator()
	trackOptionsSeparator := widget.NewSeparator()

	// Create form layout for better alignment of labels and inputs with enhanced readability
	// Create labels with explicit styling for guaranteed readability
	languageLabelContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText("Language:", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	languageLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	langCodeLabelContainer := container.NewMax(
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		canvas.NewText("Language Code:", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	langCodeLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	trackNameLabelContainer := container.NewMax(
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		canvas.NewText("Track Name:", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	trackNameLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	// Create a high-contrast container for the track name entry
	trackNamePadded := container.NewPadded(trackNameEntry)
	trackNameContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the entry directly
		trackNamePadded,
	)
	
	// Create a card with the high-contrast container
	trackNameCard := widget.NewCard("", "", trackNameContainer)
	
	// Create form layout with high-contrast labels
	languageForm := container.New(layout.NewFormLayout(),
		languageLabelContainer,
		langDropdown,
		langCodeLabelContainer,
		langCodeCard, // Using the card we created earlier
		trackNameLabelContainer,
		trackNameCard,
	)
	
	// Create a container for the language section with title, separator, and form
	languageSection := container.NewVBox(
		languageTitleContainer,
		languageSeparator,
		languageForm,
	)

	// Group track options with separator for visual distinction
	trackOptionsContainer := container.NewVBox(
		trackOptionsTitleContainer,
		trackOptionsSeparator,
		defaultTrack,
		forcedTrack,
		removeOtherTracks,
	)

	// Group subtitle options with improved organization and readability
	subtitleOptionsGroup := widget.NewCard("Subtitle Options", "", container.NewVBox(
		container.NewPadded(languageSection), // Using our new language section with title and separator
		container.NewPadded(trackOptionsContainer), // Track options already include title and separator
	))

	// Group output options
	// Make output filename entry wider with placeholder
	outputNameEntry.SetPlaceHolder("Enter output filename (leave empty to use original filename)...                                         ")

	// Create output title with explicit styling for guaranteed readability
	outputTitleContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText("Output Configuration", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	outputTitleContainer.Objects[1].(*canvas.Text).TextSize = 16
	outputTitleContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	// Create a separator for visual distinction
	outputSeparator := widget.NewSeparator()
	
	// Create output filename label with explicit styling for guaranteed readability
	outputFilenameLabelContainer := container.NewMax(
		// Light background for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Bold text with dark color for maximum readability
		canvas.NewText("Output Filename:", color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
	)
	outputFilenameLabelContainer.Objects[1].(*canvas.Text).TextStyle.Bold = true
	
	// Create a high-contrast container for the output filename entry
	outputNamePadded := container.NewPadded(outputNameEntry)
	outputNameContainer := container.NewMax(
		// Light background rectangle for contrast
		canvas.NewRectangle(color.NRGBA{R: 240, G: 240, B: 240, A: 255}),
		// Add the entry directly
		outputNamePadded,
	)
	
	// Create a card with the high-contrast container
	outputNameCard := widget.NewCard("", "", outputNameContainer)
	
	// Create form layout for better alignment with high-contrast labels
	outputForm := container.New(layout.NewFormLayout(),
		outputFilenameLabelContainer,
		outputNameCard,
	)
	
	// Create a container for the output section with title and separator
	outputSection := container.NewVBox(
		outputTitleContainer,
		outputSeparator,
		outputForm,
	)

	// Add helpful text
	helpText := widget.NewRichText(
		&widget.TextSegment{Text: "Note: ", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
		&widget.TextSegment{Text: "Leave filename empty to use the original filename with \"-subtitled\" suffix."},
	)
	helpText.Wrapping = fyne.TextWrapWord

	// Style the insert button
	insertSubtitleBtn.Importance = widget.HighImportance

	// Group output options with improved organization and readability
	outputOptionsGroup := widget.NewCard("Output Options", "", container.NewVBox(
		container.NewPadded(outputSection), // Using our new output section with title and separator
		container.NewPadded(helpText),
		container.NewHBox(layout.NewSpacer(), insertSubtitleBtn, layout.NewSpacer()),
	))

	// Results group
	resultsGroup := widget.NewCard("Results", "", insertResultScroll)

	// Create layout for subtitle insertion tab
	insertTabContent := container.NewVBox(
		insertTitleLabel,
		fileSelectionGroup,
		subtitleOptionsGroup,
		outputOptionsGroup,
		resultsGroup,
	)

	// Create settings tab content with improved styling
	// Create a title with bold styling
	settingsTitle := canvas.NewText("Application Settings", color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	settingsTitle.TextSize = 18
	settingsTitle.TextStyle.Bold = true

	// Create a header for dependencies section
	dependencyTitle := widget.NewLabelWithStyle("System Dependencies", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Create a placeholder for the dynamic dependency status updates
	settingsLabel := widget.NewLabel("Checking dependencies...")
	settingsLabel.Wrapping = fyne.TextWrapWord

	// Create a card for theme settings
	themeTitle := canvas.NewText("Theme Settings", color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	themeTitle.TextSize = 16
	themeTitle.TextStyle.Bold = true

	// Theme selector with styled label
	themeOptions := []string{"System Default", "Light Theme", "Dark Theme", "Blue Theme", "Warm Theme", "Green Theme", "Spring Theme", "Summer Theme", "Autumn Theme", "Winter Theme"}
	themeSelector := widget.NewSelect(themeOptions, func(selected string) {
		// Save the theme preference
		a.Preferences().SetString("theme", selected)
		
		switch selected {
		case "Light Theme":
			// Use our predefined light theme
			customTheme := NewCustomThemeWithPrefs(LightTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Dark Theme":
			// Use our predefined dark theme
			customTheme := NewCustomThemeWithPrefs(DarkTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Blue Theme":
			// Use our predefined blue theme
			customTheme := NewCustomThemeWithPrefs(BlueCoolTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Warm Theme":
			// Use our predefined warm theme
			customTheme := NewCustomThemeWithPrefs(WarmToneTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Green Theme":
			// Use our predefined green theme
			customTheme := NewCustomThemeWithPrefs(VibrantGreenTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Spring Theme":
			// Use our spring theme
			customTheme := NewCustomThemeWithPrefs(SpringTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Summer Theme":
			// Use our summer theme
			customTheme := NewCustomThemeWithPrefs(SummerTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Autumn Theme":
			// Use our autumn theme
			customTheme := NewCustomThemeWithPrefs(AutumnTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Winter Theme":
			// Use our winter theme
			customTheme := NewCustomThemeWithPrefs(WinterTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		default:
			a.Settings().SetTheme(theme.DefaultTheme())
		}
	})
	
	// Load saved theme preference or default to Dark Theme
	selectedTheme := a.Preferences().StringWithFallback("theme", "Dark Theme")
	themeSelector.SetSelected(selectedTheme)

	// Create a styled theme label with custom color
	themeLabel := widget.NewLabelWithStyle("Application Theme:", fyne.TextAlignLeading, fyne.TextStyle{
		Bold:      true,
		Italic:    false,
		Monospace: false,
	})

	// Create a colored rectangle background for the label
	labelRect := canvas.NewRectangle(color.NRGBA{R: 40, G: 40, B: 80, A: 255})
	labelContainer := container.NewStack(labelRect, container.NewPadded(themeLabel))

	// Note: Standard labels don't support direct color setting
	// Instead, we're using a colored background with the default text color

	// Create a button to apply theme changes with custom styling and color
	applyThemeBtn := widget.NewButtonWithIcon("Apply Theme", theme.ConfirmIcon(), func() {
		// Get the currently selected theme
		selected := themeSelector.Selected
		
		// Save the theme preference
		a.Preferences().SetString("theme", selected)
		
		// Apply the theme based on selection
		switch selected {
		case "Light Theme":
			customTheme := NewCustomThemeWithPrefs(LightTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Dark Theme":
			customTheme := NewCustomThemeWithPrefs(DarkTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Blue Theme":
			customTheme := NewCustomThemeWithPrefs(BlueCoolTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Warm Theme":
			customTheme := NewCustomThemeWithPrefs(WarmToneTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		case "Green Theme":
			customTheme := NewCustomThemeWithPrefs(VibrantGreenTheme(), true).(*CustomTheme)
			a.Settings().SetTheme(customTheme)
		default:
			a.Settings().SetTheme(theme.DefaultTheme())
		}
		
		dialog.ShowInformation("Theme Applied", "Application theme has been updated and saved.", w)
	})
	applyThemeBtn.Importance = widget.HighImportance

	// Create a custom colored apply button
	applyBtnBackground := canvas.NewRectangle(color.NRGBA{R: 0, G: 120, B: 80, A: 255})
	applyBtnContainer := container.NewStack(applyBtnBackground, container.NewPadded(applyThemeBtn))

	// No theme customization option - using predefined themes only

	// Help section
	helpTitle := canvas.NewText("Help & Information", color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	helpTitle.TextSize = 16
	helpTitle.TextStyle.Bold = true

	// App information
	versionInfo := widget.NewRichText(
		&widget.TextSegment{Text: "Subtitle Forge v1.6.5\n", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
		&widget.TextSegment{Text: "A tool for extracting and converting subtitles from MKV files.\n\n"},
		&widget.TextSegment{Text: " 2025 VenimK@David Software\n", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Italic: true}}},
	)
	versionInfo.Wrapping = fyne.TextWrapWord

	// Add a helpful description of dependencies
	dependencyDescription := widget.NewLabel("The application requires these external tools to function properly:")
	dependencyDescription.Wrapping = fyne.TextWrapWord

	// Create a list of dependencies with descriptions
	dependencyList := widget.NewLabel("• FFmpeg: Used for video and subtitle processing\n• vobsub2srt: Converts VobSub subtitles to SRT format\n• MKVMerge: Used for MKV file manipulation\n• MKVExtract: Extracts content from MKV files\n• Deno: JavaScript runtime for scripts\n• Tesseract: Optical character recognition for subtitles\n• Go: Required for building the application\n• PGStoSRT: Script for converting PGS subtitles to SRT format")
	dependencyList.Wrapping = fyne.TextWrapWord

	// Instructions for missing dependencies
	dependencyInstructions := widget.NewLabel("If any dependencies are missing, use the buttons below to install them.")
	dependencyInstructions.Wrapping = fyne.TextWrapWord
	dependencyInstructions.TextStyle = fyne.TextStyle{Italic: true}

	// Create a section for PGS to SRT script configuration
	pgsToSrtTitle := canvas.NewText("PGS to SRT Script Configuration", color.NRGBA{R: 0, G: 0, B: 180, A: 255})
	pgsToSrtTitle.TextSize = 16
	pgsToSrtTitle.TextStyle.Bold = true

	// Create a label to display the current script path
	pgsToSrtPathLabel := widget.NewLabel(pgsToSrtScriptPath)
	pgsToSrtPathLabel.Wrapping = fyne.TextWrapWord

	// Create a button to browse for the script file
	pgsToSrtBrowseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		// Create a file open dialog
		dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}

			// Update the script path
			pgsToSrtScriptPath = reader.URI().Path()
			pgsToSrtPathLabel.SetText(pgsToSrtScriptPath)
			reader.Close()

			// Run the dependency check again to update status
			updateDependencyStatus(w)
		}, w)

		// Set filters for JavaScript files
		dlg.SetFilter(storage.NewExtensionFileFilter([]string{".js"}))
		dlg.Show()
	})

	// Create a form layout for the PGS to SRT script configuration
	pgsToSrtForm := container.New(layout.NewFormLayout(),
		widget.NewLabel("Script Path:"),
		container.NewBorder(nil, nil, nil, pgsToSrtBrowseBtn, pgsToSrtPathLabel),
	)

	// Add a description for the PGS to SRT script
	pgsToSrtDescription := widget.NewLabel("The PGS to SRT script is used to convert PGS subtitles to SRT format. It requires Deno runtime.")
	pgsToSrtDescription.Wrapping = fyne.TextWrapWord

	// Combine all dependency components
	dependencySection := container.NewVBox(
		dependencyTitle,
		container.NewPadded(dependencyDescription),
		container.NewPadded(dependencyList),
		container.NewPadded(settingsLabel),
		container.NewPadded(dependencyInstructions),
		container.NewPadded(dependencyButtons),
		canvas.NewLine(color.NRGBA{R: 200, G: 200, B: 200, A: 128}),
		container.NewPadded(pgsToSrtTitle),
		container.NewPadded(pgsToSrtDescription),
		container.NewPadded(pgsToSrtForm),
	)

	// Custom themed button for resetting to default settings
	resetSettingsBtn := widget.NewButtonWithIcon("Reset to Defaults", theme.ViewRefreshIcon(), func() {
		// Reset theme to dark theme
		a.Settings().SetTheme(theme.DarkTheme())
		themeSelector.SetSelected("Dark Theme")
		dialog.ShowInformation("Settings Reset", "Settings have been reset to defaults.", w)
	})

	// Style the reset button
	resetSettingsBtn.Importance = widget.MediumImportance

	// Create a custom colored reset button
	resetBtnBackground := canvas.NewRectangle(color.NRGBA{R: 120, G: 60, B: 0, A: 255})
	resetBtnContainer := container.NewStack(resetBtnBackground, container.NewPadded(resetSettingsBtn))

	// Create a styled container for theme buttons
	themeButtonsContainer := container.NewHBox(
		applyBtnContainer,
		layout.NewSpacer(),
		resetBtnContainer,
	)

	// Create info label with custom color styling
	themeInfoLabel := widget.NewRichTextWithText("Select a theme and click Apply to change the application appearance.")
	themeInfoLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		TextStyle: fyne.TextStyle{Italic: true},
		ColorName: theme.ColorNameForeground,
	}

	// Create a colored background for the info text
	infoBackground := canvas.NewRectangle(color.NRGBA{R: 40, G: 40, B: 60, A: 255})
	infoContainer := container.NewStack(infoBackground, container.NewPadded(themeInfoLabel))

	// Set a custom color for the info text - using ColorName from theme
	// Note: RichTextStyle doesn't have a direct Color field
	themeInfoLabel.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNamePrimary

	// Assemble theme section with styled and colored components
	themeSection := container.NewVBox(
		container.NewPadded(themeTitle),
		container.NewPadded(container.New(layout.NewFormLayout(),
			labelContainer,
			themeSelector,
		)),
		container.NewPadded(infoContainer),
		container.NewPadded(themeButtonsContainer),
	)

	helpSection := container.NewVBox(
		container.NewPadded(helpTitle),
		container.NewPadded(versionInfo),
	)

	// Create cards for each section
	dependencyCard := widget.NewCard("", "", dependencySection)
	themeCard := widget.NewCard("", "", themeSection)
	helpCard := widget.NewCard("", "", helpSection)

	// Assemble settings tab content
	settingsTabContent := container.NewVBox(
		container.NewPadded(settingsTitle),
		dependencyCard,
		themeCard,
		helpCard,
	)
	updateDependencyStatus(w)

	// Wrap each tab content in a scroll container to ensure proper resizability
	extractScroll := container.NewScroll(extractTabContent)
	insertScroll := container.NewScroll(insertTabContent)
	settingsScroll := container.NewScroll(settingsTabContent)
	
	// Create tabs with scrollable content
	tabs := container.NewAppTabs(
		container.NewTabItem("Extract Subtitles", extractScroll),
		container.NewTabItem("Insert Subtitles", insertScroll),
		container.NewTabItem("Settings", settingsScroll),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Set up tab change handler for drag and drop
	tabs.OnChanged = func(tab *container.TabItem) {
		if tab.Text == "Insert Subtitles" {
			// Set up drag and drop for Insert Subtitles tab
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) > 0 {
					filePath := uris[0].Path()
					fileExt := strings.ToLower(filepath.Ext(filePath))

					if fileExt == ".mkv" {
						// Handle MKV file drop
						insertMkvFileLabel.SetText(filePath)
						mkvDropLabel.SetText(filepath.Base(filePath))
						mkvDropArea.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
						mkvDropArea.Refresh()
						a.SendNotification(&fyne.Notification{
							Title:   "File Dropped",
							Content: "MKV file loaded: " + filepath.Base(filePath),
						})
					} else if fileExt == ".srt" {
						// Handle SRT file drop
						insertSrtFileLabel.SetText(filePath)
						srtDropLabel.SetText(filepath.Base(filePath))
						srtDropArea.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
						srtDropArea.Refresh()
						a.SendNotification(&fyne.Notification{
							Title:   "File Dropped",
							Content: "SRT file loaded: " + filepath.Base(filePath),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Invalid File",
							Content: "Please drop an MKV or SRT file only.",
						})
					}
				}
			})
		} else if tab.Text == "Extract Subtitles" {
			// Restore original drag and drop for Extract Subtitles tab
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) > 0 {
					filePath := uris[0].Path()
					fileExt := strings.ToLower(filepath.Ext(filePath))

					if fileExt == ".mkv" {
						// Handle MKV file drop
						mkvPath = filePath
						a.SendNotification(&fyne.Notification{
							Title:   "File Dropped",
							Content: "MKV file loaded: " + filepath.Base(filePath),
						})

						// Update UI
						selectedFile.SetText(mkvPath)

						// Set output directory to the same directory as the MKV file
						outDir = filepath.Dir(mkvPath)
						selectedDir.SetText(outDir)

						// Clear previous tracks
						trackItems = []*TrackItem{}
						trackList.Objects = nil
						trackList.Refresh()

						result.SetText("MKV file dropped and loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file.")
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Invalid File",
							Content: "Please drop an MKV file only.",
						})
					}
				}
			})
		}
	}

	// Set the tabs as the window content
	w.SetContent(tabs)

	// Trigger the OnChanged handler for the initial tab
	tabs.OnChanged(tabs.Selected())

	w.ShowAndRun()
}
