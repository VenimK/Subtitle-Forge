package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Global variable to store the path to mkvmerge
var mkvmergeBinaryPath string

// Global variable to store the path to mkvextract
var mkvextractBinaryPath string

// Global variable to store the path to deno
var denoBinaryPath string

// PGS to SRT script path - configurable via UI
var pgsToSrtScriptPath = filepath.Join(os.Getenv("HOME"), "pgs-to-srt", "pgs-to-srt.js")

// findHomebrewPath checks for Homebrew installation in common locations
func findHomebrewPath() (string, error) {
	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for Homebrew ===\n")
	}

	// First try using exec.LookPath to find brew in PATH
	brewPath, err := exec.LookPath("brew")
	if err == nil {
		fmt.Println("[DEBUG] Homebrew found in PATH at", brewPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "Homebrew found in PATH at: %s\n", brewPath)
		}
		return brewPath, nil
	} else {
		fmt.Println("[DEBUG] Homebrew not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "Homebrew not found in PATH: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/opt/homebrew/bin/brew", // Apple Silicon Macs
			"/usr/local/bin/brew",    // Intel Macs
			"/usr/bin/brew",
		}

		// Get home directory for user-specific paths
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Add user-specific paths
			userPaths := []string{
				filepath.Join(homeDir, "homebrew", "bin", "brew"),
				filepath.Join(homeDir, ".homebrew", "bin", "brew"),
				filepath.Join(homeDir, "bin", "brew"),
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
					fmt.Println("[DEBUG] Homebrew found at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "Homebrew found at: %s\n", path)
					}
					return path, nil
				}
			}
		}

		// If we get here, Homebrew was not found
		fmt.Println("[DEBUG] Homebrew not found in any common locations")
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "Homebrew not found in any common locations\n")
		}
		return "", fmt.Errorf("Homebrew not found")
	}
}

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
	// Create a debug log file in the application's log directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		logDir := filepath.Join(homeDir, ".subtitle-forge", "logs")
		os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "dependency_check_debug.log")
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
	dependencyResults["pgsrip"] = checkPgsrip()

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
		mkvmergeCmd := exec.Command(mkvmergePath, "--version")
		mkvmergeOutput, err := mkvmergeCmd.CombinedOutput()
		mkvmergeFound = err == nil && len(mkvmergeOutput) > 0

		if mkvmergeFound {
			fmt.Println("[DEBUG] MKVMerge found:", strings.TrimSpace(string(mkvmergeOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVMerge verified with version: %s\n", strings.TrimSpace(string(mkvmergeOutput)))
			}
			// Store the path for later use
			mkvmergeBinaryPath = mkvmergePath
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
					// Store the path for later use
					mkvmergeBinaryPath = path
					break
				}
			}
		}
	}

	fmt.Println("[DEBUG] Final MKVMerge found status:", mkvmergeFound, "Path:", mkvmergeBinaryPath)
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
		mkvextractCmd := exec.Command(mkvextractPath, "--version")
		mkvextractOutput, err := mkvextractCmd.CombinedOutput()
		mkvextractFound = err == nil && len(mkvextractOutput) > 0

		if mkvextractFound {
			fmt.Println("[DEBUG] MKVExtract found:", strings.TrimSpace(string(mkvextractOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "MKVExtract verified with version: %s\n", strings.TrimSpace(string(mkvextractOutput)))
			}
			// Store the path for later use
			mkvextractBinaryPath = mkvextractPath
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
					// Store the path for later use
					mkvextractBinaryPath = path
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
	denoFound := false

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for Deno ===\n")
	}

	// First try using exec.LookPath to find deno in PATH
	denoPath, err := exec.LookPath("deno")
	if err == nil {
		fmt.Println("[DEBUG] deno found in PATH at", denoPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "deno found in PATH at: %s\n", denoPath)
		}

		// Verify by running the command
		denoCmd := exec.Command(denoPath, "--version")
		denoOutput, err := denoCmd.CombinedOutput()
		denoFound = err == nil && len(denoOutput) > 0

		if denoFound {
			fmt.Println("[DEBUG] Deno found:", strings.TrimSpace(string(denoOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Deno verified with version: %s\n", strings.TrimSpace(string(denoOutput)))
			}
			// Store the path for later use
			denoBinaryPath = denoPath
		} else {
			fmt.Println("[DEBUG] Deno command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Deno command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] deno not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "deno not found in PATH: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/usr/local/bin/deno",
			"/opt/homebrew/bin/deno",
			"/usr/bin/deno",
			"/bin/deno",
			filepath.Join(os.Getenv("HOME"), ".deno", "bin", "deno"),
		}

		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] deno found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "deno found at: %s\n", path)
				}

				// Verify by running the command
				denoCmd := exec.Command(path, "--version")
				denoOutput, err := denoCmd.CombinedOutput()
				if err == nil && len(denoOutput) > 0 {
					denoFound = true
					fmt.Println("[DEBUG] Deno verified at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "Deno verified at: %s\n", path)
					}
					// Store the path for later use
					denoBinaryPath = path
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "Deno command failed at %s: %v\n", path, err)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !denoFound {
			fmt.Fprintf(debugLogger, "deno not found in any common paths\n")
		}
	}

	fmt.Println("[DEBUG] Final Deno found status:", denoFound, "Path:", denoBinaryPath)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final Deno found status: %v, Path: %s\n\n", denoFound, denoBinaryPath)
	}
	return denoFound
}

// Check for Tesseract installation
func checkTesseract() bool {
	fmt.Println("[DEBUG] Checking for Tesseract...")
	tesseractFound := false

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for Tesseract ===\n")
	}

	// First try using exec.LookPath to find tesseract in PATH
	tesseractPath, err := exec.LookPath("tesseract")
	if err == nil {
		fmt.Println("[DEBUG] tesseract found in PATH at", tesseractPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "tesseract found in PATH at: %s\n", tesseractPath)
		}

		// Verify by running the command
		tesseractCmd := exec.Command(tesseractPath, "--version")
		tesseractOutput, err := tesseractCmd.CombinedOutput()
		tesseractFound = err == nil && len(tesseractOutput) > 0

		if tesseractFound {
			fmt.Println("[DEBUG] Tesseract found:", strings.TrimSpace(string(tesseractOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Tesseract verified with version: %s\n", strings.TrimSpace(string(tesseractOutput)))
			}
		} else {
			fmt.Println("[DEBUG] Tesseract command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Tesseract command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] tesseract not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "tesseract not found in PATH: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/usr/local/bin/tesseract",
			"/opt/homebrew/bin/tesseract",
			"/usr/bin/tesseract",
			"/bin/tesseract",
		}

		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] tesseract found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "tesseract found at: %s\n", path)
				}

				// Verify by running the command
				tesseractCmd := exec.Command(path, "--version")
				tesseractOutput, err := tesseractCmd.CombinedOutput()
				if err == nil && len(tesseractOutput) > 0 {
					tesseractFound = true
					fmt.Println("[DEBUG] Tesseract verified at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "Tesseract verified at: %s\n", path)
					}
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "Tesseract command failed at %s: %v\n", path, err)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !tesseractFound {
			fmt.Fprintf(debugLogger, "tesseract not found in any common paths\n")
		}
	}

	fmt.Println("[DEBUG] Final Tesseract found status:", tesseractFound)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final Tesseract found status: %v\n\n", tesseractFound)
	}
	return tesseractFound
}

// Check for Go installation
func checkGo() bool {
	fmt.Println("[DEBUG] Checking for Go...")
	goFound := false

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for Go ===\n")
	}

	// First try using exec.LookPath to find go in PATH
	goPath, err := exec.LookPath("go")
	if err == nil {
		fmt.Println("[DEBUG] go found in PATH at", goPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "go found in PATH at: %s\n", goPath)
		}

		// Verify by running the command
		goCmd := exec.Command(goPath, "version")
		goOutput, err := goCmd.CombinedOutput()
		goFound = err == nil && len(goOutput) > 0

		if goFound {
			fmt.Println("[DEBUG] Go found:", strings.TrimSpace(string(goOutput)))
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Go verified with version: %s\n", strings.TrimSpace(string(goOutput)))
			}
		} else {
			fmt.Println("[DEBUG] Go command failed:", err)
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Go command failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("[DEBUG] go not found in PATH:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "go not found in PATH: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/usr/local/go/bin/go",
			"/usr/local/bin/go",
			"/opt/homebrew/bin/go",
			"/usr/bin/go",
			"/bin/go",
			filepath.Join(os.Getenv("HOME"), "go", "bin", "go"),
		}

		// Check each path
		for _, path := range commonPaths {
			if fileInfo, err := os.Stat(path); err == nil && fileInfo.Mode().Perm()&0111 != 0 {
				fmt.Println("[DEBUG] go found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "go found at: %s\n", path)
				}

				// Verify by running the command
				goCmd := exec.Command(path, "version")
				goOutput, err := goCmd.CombinedOutput()
				if err == nil && len(goOutput) > 0 {
					goFound = true
					fmt.Println("[DEBUG] Go verified at", path)
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "Go verified at: %s\n", path)
					}
					break
				} else if debugLogger != nil {
					fmt.Fprintf(debugLogger, "Go command failed at %s: %v\n", path, err)
				}
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
		if debugLogger != nil && !goFound {
			fmt.Fprintf(debugLogger, "go not found in any common paths\n")
		}
	}

	fmt.Println("[DEBUG] Final Go found status:", goFound)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final Go found status: %v\n\n", goFound)
	}
	return goFound
}

// Check for PGS to SRT script
func checkPgsToSrt() bool {
	fmt.Println("[DEBUG] Checking for PGS to SRT script...")
	scriptFound := false

	// Log to debug file if available
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "=== Checking for PGS to SRT script ===\n")
	}

	// Use the configurable script path first
	scriptPath := pgsToSrtScriptPath
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Checking configured script path: %s\n", scriptPath)
	}

	// Check if the script exists at the specified path
	_, err := os.Stat(scriptPath)
	scriptFound = err == nil

	if scriptFound {
		fmt.Println("[DEBUG] PGS to SRT script found at:", scriptPath)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "PGS to SRT script found at: %s\n", scriptPath)
		}

		// Additionally check if Deno is available to run the script
		denoAvailable := checkDeno()
		if !denoAvailable {
			fmt.Println("[DEBUG] PGS to SRT script found but Deno runtime is missing")
			if debugLogger != nil {
				fmt.Fprintf(debugLogger, "PGS to SRT script found but Deno runtime is missing\n")
			}
			return false
		}
	} else {
		fmt.Println("[DEBUG] PGS to SRT script not found at configured path, error:", err)
		if debugLogger != nil {
			fmt.Fprintf(debugLogger, "PGS to SRT script not found at configured path: %v\n", err)
		}

		// Check common installation paths
		commonPaths := []string{
			"/usr/local/bin/pgs-to-srt.js",
			filepath.Join(os.Getenv("HOME"), "pgs-to-srt", "pgs-to-srt.js"),
			filepath.Join(os.Getenv("HOME"), ".deno", "bin", "pgs-to-srt.js"),
			filepath.Join(os.Getenv("HOME"), ".deno", "bin", "pgs-to-srt"),
		}

		// Check each path
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				fmt.Println("[DEBUG] PGS to SRT script found at", path)
				if debugLogger != nil {
					fmt.Fprintf(debugLogger, "PGS to SRT script found at: %s\n", path)
				}
				scriptFound = true
				pgsToSrtScriptPath = path // Update the global path variable

				// Check if Deno is available to run the script
				denoAvailable := checkDeno()
				if !denoAvailable {
					fmt.Println("[DEBUG] PGS to SRT script found but Deno runtime is missing")
					if debugLogger != nil {
						fmt.Fprintf(debugLogger, "PGS to SRT script found but Deno runtime is missing\n")
					}
					return false
				}
				break
			} else if debugLogger != nil {
				fmt.Fprintf(debugLogger, "Checked path %s: %v\n", path, err)
			}
		}
	}

	fmt.Println("[DEBUG] Final PGS to SRT script found status:", scriptFound)
	if debugLogger != nil {
		fmt.Fprintf(debugLogger, "Final PGS to SRT script found status: %v\n\n", scriptFound)
	}
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
				var brewPath string

				// Check if brew is installed first
				if tool != "vobsub2srt" { // Skip brew check for vobsub2srt as it uses custom script
					// Use robust Homebrew detection
					var err error
					brewPath, err = findHomebrewPath()
					if err != nil {
						// Hide progress dialog
						progress.Hide()

						// Show confirmation dialog to install Homebrew
						confirmDialog := dialog.NewConfirm(
							"Homebrew Required",
							"Homebrew is required but not installed. Would you like to install Homebrew now?\n\nThis will run the official Homebrew installation script.",
							func(install bool) {
								if install {
									// Show progress dialog for Homebrew installation
									brewProgress := dialog.NewProgress("Installing Homebrew", "Running Homebrew installation script...", w)
									brewProgress.Show()

									// Run Homebrew installation in a goroutine
									go func() {
										// Create the installation command
										// Use a more reliable approach to run the Homebrew installation script
										tempScript := filepath.Join(os.TempDir(), "homebrew_install.sh")

										// Download the script first
										downloadCmd := exec.Command("curl", "-fsSL", "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh", "-o", tempScript)
										downloadErr := downloadCmd.Run()
										if downloadErr != nil {
											fyne.Do(func() {
												brewProgress.Hide()
												dialog.ShowError(
													fmt.Errorf("Failed to download Homebrew installation script: %v", downloadErr),
													w)
											})
											return
										}

										// Make the script executable
										os.Chmod(tempScript, 0755)

										// Run the installation script
										cmd := exec.Command("/bin/bash", tempScript)

										// Run the command
										err := cmd.Run()

										// Update UI on main thread
										fyne.Do(func() {
											// Hide progress dialog
											brewProgress.Hide()

											if err != nil {
												// Show error if installation failed
												dialog.ShowError(
													fmt.Errorf("Failed to install Homebrew: %v\n\nPlease install manually from https://brew.sh", err),
													w)
											} else {
												// Show success message and offer to continue with dependency installation
												dialog.ShowConfirm(
													"Homebrew Installed",
													"Homebrew was successfully installed. Would you like to continue installing "+tool+"?",
													func(continueInstall bool) {
														if continueInstall {
															// Restart the dependency installation process
															installDependency(w, tool)
														}
													},
													w)
											}
										})
									}()
								}
							},
							w)
						confirmDialog.SetDismissText("Cancel")
						confirmDialog.SetConfirmText("Install Homebrew")
						confirmDialog.Show()
						return
					}
				}

				// Set up command and description based on tool
				// Using a case-insensitive approach to handle various tool name formats
				toolLower := strings.ToLower(tool)

				// Store the Homebrew path for use in commands
				var brewCommand string
				if tool != "vobsub2srt" { // Skip for vobsub2srt as it uses custom script
					brewCommand = brewPath // Use the detected Homebrew path
				} else {
					brewCommand = "brew" // Fallback, though this shouldn't be reached for vobsub2srt
				}

				switch toolLower {
				case "mkvmerge", "mkvextract", "mkvm":
					// Install MKVToolNix via Homebrew
					cmd = exec.Command(brewCommand, "install", "mkvtoolnix")
					installDesc = "Installing MKVToolNix (provides mkvmerge and mkvextract)"
				case "deno":
					// Install Deno via Homebrew
					cmd = exec.Command(brewCommand, "install", "deno")
					installDesc = "Installing Deno runtime"
				case "tesseract":
					// Install Tesseract via Homebrew
					cmd = exec.Command(brewCommand, "install", "tesseract")
					installDesc = "Installing Tesseract OCR engine"
				case "ffmpeg", "ffmp":
					// Install ffmpeg via Homebrew
					cmd = exec.Command(brewCommand, "install", "ffmpeg")
					installDesc = "Installing FFmpeg multimedia framework"
				case "go":
					// Install Go via Homebrew
					cmd = exec.Command(brewCommand, "install", "go")
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
				case "pgsrip":
					// Use the custom installation script for pgsrip
					execPath, err := os.Executable()
					if err != nil {
						fmt.Println("[ERROR] Failed to get executable path:", err)
					}

					scriptPath := filepath.Join(filepath.Dir(execPath), "install_pgsrip.sh")

					// Check if script exists
					if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
						progress.Hide()
						dialog.ShowError(
							fmt.Errorf("Installation script not found: %s", scriptPath),
							w)
						return
					}

					cmd = exec.Command("bash", scriptPath)
					installDesc = "Installing pgsrip Python package for PGS OCR"
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
			case "pgsrip":
				// Get the script path relative to the executable
				execPath, err := os.Executable()
				if err != nil {
					fmt.Println("[ERROR] Failed to get executable path:", err)
					execPath = "."
				}
				execDir := filepath.Dir(execPath)
				scriptPath := filepath.Join(execDir, "install_pgsrip.sh")
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
