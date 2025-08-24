package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Global variable to store the path to the pgsrip binary
var pgsripBinaryPath = ""

// PgsConversionSettings stores configuration for PGS to SRT conversion
type PgsConversionSettings struct {
	Verbose bool // Display verbose debug messages - only option we're using
}

// Check for pgsrip installation
func checkPgsrip() bool {
	fmt.Println("[DEBUG] Checking for pgsrip...")
	pgsripFound := false

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for pgsrip ===\n")
	}

	// Extend PATH with common Go binary locations to help find pgsrip when launched from Applications
	currentPath := os.Getenv("PATH")
	homeDir, _ := os.UserHomeDir()
	gopathBin := filepath.Join(homeDir, "go", "bin")
	homebrewBin := "/opt/homebrew/bin"
	userLocalBin := "/usr/local/bin"
	
	// Add these directories to PATH if they're not already there
	newPath := currentPath
	if !strings.Contains(newPath, gopathBin) {
		newPath += ":" + gopathBin
	}
	if !strings.Contains(newPath, homebrewBin) {
		newPath += ":" + homebrewBin
	}
	if !strings.Contains(newPath, userLocalBin) {
		newPath += ":" + userLocalBin
	}
	
	// Set the updated PATH
	os.Setenv("PATH", newPath)
	
	// Set TESSDATA_PREFIX to point to tessdata_best in user's home directory
	tessdataPath := filepath.Join(homeDir, "tessdata_best")
	os.Setenv("TESSDATA_PREFIX", tessdataPath)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Set TESSDATA_PREFIX to: %s\n", tessdataPath)
	}
	fmt.Println("[DEBUG] Set TESSDATA_PREFIX to:", tessdataPath)
	
	// Debug the PATH
	fmt.Println("[DEBUG] Extended PATH: " + newPath)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Extended PATH: %s\n", newPath)
	}
	
	// Try to find pgsrip in the updated PATH
	if pgsripPath, err := exec.LookPath("pgsrip"); err == nil {
		fmt.Println("[DEBUG] pgsrip found in PATH:", pgsripPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "pgsrip found in PATH: %s\n", pgsripPath)
		}

		// Verify by running the command
		pgsripCmd := exec.Command(pgsripPath, "--version")
		output, err := pgsripCmd.CombinedOutput()
		if err == nil {
			pgsripFound = true
			pgsripBinaryPath = pgsripPath
			fmt.Println("[DEBUG] pgsrip verified:", string(output))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "pgsrip verified: %s\n", string(output))
			}
		} else {
			fmt.Println("[DEBUG] pgsrip command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "pgsrip command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] pgsrip not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "pgsrip not found in PATH: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/usr/local/bin/pgsrip",
			"/opt/homebrew/bin/pgsrip",
			"/usr/bin/pgsrip",
			"/bin/pgsrip",
		}

		// Get home directory for user-specific paths
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Add user-specific paths
			userPaths := []string{
				filepath.Join(homeDir, "bin", "pgsrip"),
				filepath.Join(homeDir, "go", "bin", "pgsrip"),
			}
			commonPaths = append(commonPaths, userPaths...)
		}

		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] pgsrip found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "pgsrip found at: %s\n", path)
				}

				// Verify by running the command
				pgsripCmd := exec.Command(path, "--help")
				pgsripOutput, err := pgsripCmd.CombinedOutput()
				if err == nil && len(pgsripOutput) > 0 {
					pgsripFound = true
					fmt.Println("[DEBUG] pgsrip verified at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "pgsrip verified at: %s\n", path)
					}
					// Store the path for later use
					pgsripBinaryPath = path
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "pgsrip command failed at %s: %v\n", path, err)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !pgsripFound {
			fmt.Fprintf(debugLogger, "pgsrip not found in any common paths\n")
		}
	}

	// Print the final status
	fmt.Println("[DEBUG] Final pgsrip found status:", pgsripFound, "Path:", pgsripBinaryPath)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final pgsrip found status: %v, Path: %s\n\n", pgsripFound, pgsripBinaryPath)
	}
	return pgsripFound
}

// convertPgsWithPgsrip converts a PGS subtitle file to SRT format using pgsrip
func convertPgsWithPgsrip(pgsFilePath, outFilePath, langCode string, result *widget.Label, statusLabel *widget.Label, progressBar *widget.ProgressBar, conversionSettings PgsConversionSettings) error {
	// Check if pgsrip is available
	if pgsripBinaryPath == "" {
		if !checkPgsrip() {
			fyne.Do(func() {
				statusLabel.SetText("Error: pgsrip not found")
				result.SetText(result.Text + "\nError: Could not find pgsrip. Please install it via Homebrew: brew install pgsrip")
			})
			return fmt.Errorf("pgsrip not found. Please install it via Homebrew")
		}
	}

	// Create temporary output file path
	dir := filepath.Dir(outFilePath)
	tmpOutputPath := filepath.Join(dir, "tmp_"+filepath.Base(outFilePath))

	// Build command arguments - only use input file, output file and --verbose
	cmdArgs := []string{
		pgsFilePath,
		tmpOutputPath,
		"--verbose", // Only include the verbose flag
	}

	// Show command in UI
	commandStr := pgsripBinaryPath
	for _, arg := range cmdArgs {
		commandStr += " " + arg
	}
	fyne.Do(func() {
		result.SetText(result.Text + "\nCommand: " + commandStr)
	})

	// Run the command
	cmd := exec.Command(pgsripBinaryPath, cmdArgs...)
	cmd.Dir = filepath.Dir(pgsFilePath) // Set working directory to where the PGS file is

	// Set up a pipe to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fyne.Do(func() {
			result.SetText(result.Text + "\nError setting up output capture: " + err.Error())
		})
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fyne.Do(func() {
			result.SetText(result.Text + "\nError setting up error output capture: " + err.Error())
		})
		return err
	}

	// Variable to track progress
	var outputBuffer strings.Builder
	var stdoutWriter, stderrWriter io.Writer

	// Decide if we should use detailed progress output
	if progressBar != nil {
		// Create a multiplexer for each output stream
		stdoutWriter = io.MultiWriter(os.Stdout, &outputBuffer)
		stderrWriter = io.MultiWriter(os.Stderr, &outputBuffer)
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
			line := scanner.Text()

			if strings.HasPrefix(line, "Progress: ") {
				// Parse progress percentage
				progMatch := regexp.MustCompile(`Progress: ([0-9.]+)%`).FindStringSubmatch(line)
				if len(progMatch) == 2 {
					prog, _ := strconv.ParseFloat(progMatch[1], 64)
					fyne.Do(func() {
						progressBar.SetValue(prog / 100)
					})
				}
			}

			// Check for progress information in the output
			if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
				current, _ := strconv.Atoi(matches[1])
				total, _ := strconv.Atoi(matches[2])
				if total > 0 {
					progress := float64(current) / float64(total)
					fyne.Do(func() {
						progressBar.SetValue(progress)
					})
				}
			}

			// Check for status updates
			if matches := statusUpdateRegex.FindStringSubmatch(line); len(matches) == 2 {
				status := matches[1]
				fyne.Do(func() {
					statusLabel.SetText(status)
				})
			}

			// Write to our multiplexed writers
			_, _ = stdoutWriter.Write([]byte(line))
		}
	}()

	go func() {
		bufReader := bufio.NewReaderSize(stderrPipe, 4096)
		scanner := bufio.NewScanner(bufReader)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			_, _ = stderrWriter.Write([]byte(line))
		}
	}()

	// Start the command
	if err := cmd.Start(); err != nil {
		fyne.Do(func() {
			result.SetText(result.Text + "\nError starting conversion: " + err.Error())
		})
		return err
	}

	// Wait for the command to complete
	err = cmd.Wait()

	// Display the output
	fyne.Do(func() {
		result.SetText(result.Text + "\n" + outputBuffer.String())
		if err != nil {
			result.SetText(result.Text + "\nConversion error: " + err.Error())
			statusLabel.SetText("Conversion failed!")
		} else {
			// Copy the output file to the desired location
			if tmpOutputPath != outFilePath {
				if copyErr := copyFile(tmpOutputPath, outFilePath); copyErr != nil {
					result.SetText(result.Text + "\nError copying output file: " + copyErr.Error())
				} else {
					os.Remove(tmpOutputPath) // Clean up the temp file
				}
			}
			result.SetText(result.Text + "\nPGS to SRT conversion completed.")
			statusLabel.SetText("Conversion completed!")
			progressBar.SetValue(1.0) // Set progress to 100%
		}
	})

	return err
}
