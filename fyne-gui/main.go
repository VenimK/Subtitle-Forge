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
	FilePath   string        // Source MKV file path (for batch processing)
	Check      *widget.Check
	Status     *widget.Label
	ConvertOCR *widget.Check  // Option to convert PGS to SRT using OCR
	LangSelect *widget.Select // Language selection dropdown for OCR
}

// Global debug logger for dependency checks
var debugLogger *os.File

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

// Global variable to store the path to mkvmerge
var mkvmergeBinaryPath string

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

	// Global variable to store the path to mkvextract
	var mkvextractBinaryPath string
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

// Global variable to store the path to deno
var denoBinaryPath string

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

// PGS to SRT script path - configurable via UI
var pgsToSrtScriptPath = filepath.Join(os.Getenv("HOME"), "pgs-to-srt", "pgs-to-srt.js")

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

// detectSubtitleFormat detects the subtitle format based on file extension and content
func detectSubtitleFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".srt":
		return "SRT"
	case ".ass":
		return "ASS"
	case ".ssa":
		return "SSA"
	case ".vtt":
		return "VTT"
	case ".sub":
		// Could be VobSub or MicroDVD, check content
		if isVobSubFile(filePath) {
			return "VobSub"
		}
		return "SUB"
	case ".idx":
		return "VobSub"
	case ".sup":
		return "PGS"
	case ".txt":
		return "TXT"
	default:
		// Try to detect by content
		return detectFormatByContent(filePath)
	}
}

// isVobSubFile checks if a .sub file is VobSub format
func isVobSubFile(filePath string) bool {
	// Check if there's a corresponding .idx file
	idxPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".idx"
	if _, err := os.Stat(idxPath); err == nil {
		return true
	}
	return false
}

// detectFormatByContent tries to detect subtitle format by examining file content
func detectFormatByContent(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return "Unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	
	for scanner.Scan() && lineCount < 10 {
		line := strings.TrimSpace(scanner.Text())
		lineCount++
		
		// SRT format detection
		if regexp.MustCompile(`^\d+$`).MatchString(line) {
			// Next line should be timestamp
			if scanner.Scan() {
				nextLine := strings.TrimSpace(scanner.Text())
				if regexp.MustCompile(`\d{2}:\d{2}:\d{2},\d{3} --> \d{2}:\d{2}:\d{2},\d{3}`).MatchString(nextLine) {
					return "SRT"
				}
			}
		}
		
		// ASS/SSA format detection
		if strings.Contains(line, "[Script Info]") || strings.Contains(line, "[V4+ Styles]") {
			return "ASS"
		}
		
		// VTT format detection
		if strings.HasPrefix(line, "WEBVTT") {
			return "VTT"
		}
	}
	
	return "Unknown"
}

// convertSubtitleFile performs the actual subtitle conversion
func convertSubtitleFile(inputPath, inputFormat, outputFormat, outputDir string, 
	preserveTiming, preserveStyle bool, encoding string, 
	progress *widget.ProgressBar, result *widget.Label) bool {
	
	// Determine output path
	var outputPath string
	if outputDir != "" {
		fileName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outputPath = filepath.Join(outputDir, fileName+"."+outputFormat)
	} else {
		outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "." + outputFormat
	}
	
	fyne.Do(func() {
		progress.SetValue(0.1)
		result.SetText(fmt.Sprintf("🔄 Converting %s to %s...", inputFormat, strings.ToUpper(outputFormat)))
	})
	
	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error reading input file: %v", err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(0.3)
		result.SetText("🔄 Parsing subtitle format...")
	})
	
	// Parse input format
	subtitles, err := parseSubtitleFile(string(inputData), inputFormat)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error parsing %s format: %v", inputFormat, err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(0.6)
		result.SetText("🔄 Converting to target format...")
	})
	
	// Convert to output format
	outputData, err := convertToFormat(subtitles, outputFormat, preserveTiming, preserveStyle)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error converting to %s format: %v", outputFormat, err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(0.8)
		result.SetText("🔄 Writing output file...")
	})
	
	// Write output file
	err = os.WriteFile(outputPath, []byte(outputData), 0644)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error writing output file: %v", err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(1.0)
		result.SetText(fmt.Sprintf("✅ Successfully converted to: %s", outputPath))
	})
	
	return true
}

// SubtitleEntry represents a single subtitle entry
type SubtitleEntry struct {
	Index     int
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
	Style     string
}

// parseSubtitleFile parses different subtitle formats
func parseSubtitleFile(content, format string) ([]SubtitleEntry, error) {
	switch strings.ToUpper(format) {
	case "SRT":
		return parseSRT(content)
	case "ASS", "SSA":
		return parseASS(content)
	case "VTT":
		return parseVTT(content)
	case "PGS":
		return nil, fmt.Errorf("PGS format detected - OCR conversion will be applied automatically")
	default:
		return nil, fmt.Errorf("unsupported input format: %s", format)
	}
}

// parseSRT parses SRT format
func parseSRT(content string) ([]SubtitleEntry, error) {
	var entries []SubtitleEntry
	lines := strings.Split(content, "\n")
	
	for i := 0; i < len(lines); {
		// Skip empty lines
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		
		// Parse index
		indexStr := strings.TrimSpace(lines[i])
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			i++
			continue
		}
		i++
		
		// Parse timestamp
		if i >= len(lines) {
			break
		}
		timeLine := strings.TrimSpace(lines[i])
		startTime, endTime, err := parseSRTTimestamp(timeLine)
		if err != nil {
			i++
			continue
		}
		i++
		
		// Parse text
		var textLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textLines = append(textLines, lines[i])
			i++
		}
		
		if len(textLines) > 0 {
			entries = append(entries, SubtitleEntry{
				Index:     index,
				StartTime: startTime,
				EndTime:   endTime,
				Text:      strings.Join(textLines, "\n"),
			})
		}
	}
	
	return entries, nil
}

// parseSRTTimestamp parses SRT timestamp format
func parseSRTTimestamp(line string) (time.Duration, time.Duration, error) {
	parts := strings.Split(line, " --> ")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid timestamp format: %s", line)
	}
	
	start, err := parseSRTTime(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	
	end, err := parseSRTTime(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	
	return start, end, nil
}

// parseSRTTime parses individual SRT time
func parseSRTTime(timeStr string) (time.Duration, error) {
	// Format: HH:MM:SS,mmm
	re := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	
	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	milliseconds, _ := strconv.Atoi(matches[4])
	
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond
	
	return duration, nil
}

// parseASS parses ASS/SSA format (simplified)
func parseASS(content string) ([]SubtitleEntry, error) {
	var entries []SubtitleEntry
	lines := strings.Split(content, "\n")
	
	inEvents := false
	index := 1
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "[Events]" {
			inEvents = true
			continue
		}
		
		if inEvents && strings.HasPrefix(line, "Dialogue:") {
			parts := strings.Split(line, ",")
			if len(parts) >= 10 {
				startTime, err := parseASSTime(parts[1])
				if err != nil {
					continue
				}
				
				endTime, err := parseASSTime(parts[2])
				if err != nil {
					continue
				}
				
				// Text is everything after the 9th comma
				textParts := parts[9:]
				text := strings.Join(textParts, ",")
				
				entries = append(entries, SubtitleEntry{
					Index:     index,
					StartTime: startTime,
					EndTime:   endTime,
					Text:      text,
				})
				index++
			}
		}
	}
	
	return entries, nil
}

// parseASSTime parses ASS time format
func parseASSTime(timeStr string) (time.Duration, error) {
	// Format: H:MM:SS.cc
	re := regexp.MustCompile(`(\d+):(\d{2}):(\d{2})\.(\d{2})`)
	matches := re.FindStringSubmatch(strings.TrimSpace(timeStr))
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid ASS time format: %s", timeStr)
	}
	
	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	centiseconds, _ := strconv.Atoi(matches[4])
	
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(centiseconds*10)*time.Millisecond
	
	return duration, nil
}

// parseVTT parses WebVTT format (simplified)
func parseVTT(content string) ([]SubtitleEntry, error) {
	var entries []SubtitleEntry
	lines := strings.Split(content, "\n")
	
	index := 1
	
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		
		// Skip WEBVTT header and empty lines
		if line == "" || strings.HasPrefix(line, "WEBVTT") || strings.HasPrefix(line, "NOTE") {
			i++
			continue
		}
		
		// Check if this line contains timestamp
		if strings.Contains(line, "-->") {
			startTime, endTime, err := parseVTTTimestamp(line)
			if err != nil {
				i++
				continue
			}
			i++
			
			// Parse text
			var textLines []string
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				textLines = append(textLines, lines[i])
				i++
			}
			
			if len(textLines) > 0 {
				entries = append(entries, SubtitleEntry{
					Index:     index,
					StartTime: startTime,
					EndTime:   endTime,
					Text:      strings.Join(textLines, "\n"),
				})
				index++
			}
		} else {
			i++
		}
	}
	
	return entries, nil
}

// parseVTTTimestamp parses WebVTT timestamp
func parseVTTTimestamp(line string) (time.Duration, time.Duration, error) {
	parts := strings.Split(line, " --> ")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid VTT timestamp format: %s", line)
	}
	
	start, err := parseVTTTime(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	
	end, err := parseVTTTime(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	
	return start, end, nil
}

// parseVTTTime parses individual VTT time
func parseVTTTime(timeStr string) (time.Duration, error) {
	// Format: MM:SS.mmm or HH:MM:SS.mmm
	re := regexp.MustCompile(`(?:(\d+):)?(\d{2}):(\d{2})\.(\d{3})`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) < 4 {
		return 0, fmt.Errorf("invalid VTT time format: %s", timeStr)
	}
	
	var hours, minutes, seconds, milliseconds int
	
	if matches[1] != "" {
		hours, _ = strconv.Atoi(matches[1])
		minutes, _ = strconv.Atoi(matches[2])
		seconds, _ = strconv.Atoi(matches[3])
		milliseconds, _ = strconv.Atoi(matches[4])
	} else {
		minutes, _ = strconv.Atoi(matches[2])
		seconds, _ = strconv.Atoi(matches[3])
		milliseconds, _ = strconv.Atoi(matches[4])
	}
	
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond
	
	return duration, nil
}

// convertToFormat converts subtitles to the specified output format
func convertToFormat(entries []SubtitleEntry, format string, preserveTiming, preserveStyle bool) (string, error) {
	switch strings.ToUpper(format) {
	case "SRT":
		return convertToSRT(entries), nil
	case "ASS":
		return convertToASS(entries, preserveStyle), nil
	case "SSA":
		return convertToSSA(entries, preserveStyle), nil
	case "VTT":
		return convertToVTT(entries), nil
	case "SUB":
		return convertToSUB(entries), nil
	case "TXT":
		return convertToTXT(entries), nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// convertToSRT converts to SRT format
func convertToSRT(entries []SubtitleEntry) string {
	var result strings.Builder
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("%d\n", entry.Index))
		result.WriteString(fmt.Sprintf("%s --> %s\n", 
			formatSRTTime(entry.StartTime), 
			formatSRTTime(entry.EndTime)))
		result.WriteString(entry.Text)
		result.WriteString("\n\n")
	}
	
	return result.String()
}

// formatSRTTime formats duration to SRT time format
func formatSRTTime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Nanoseconds()/1000000) % 1000
	
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

// convertToASS converts to ASS format (basic)
func convertToASS(entries []SubtitleEntry, preserveStyle bool) string {
	var result strings.Builder
	
	// ASS header
	result.WriteString("[Script Info]\n")
	result.WriteString("Title: Converted by Subtitle Forge\n")
	result.WriteString("ScriptType: v4.00+\n\n")
	
	result.WriteString("[V4+ Styles]\n")
	result.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	result.WriteString("Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n\n")
	
	result.WriteString("[Events]\n")
	result.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			formatASSTime(entry.StartTime),
			formatASSTime(entry.EndTime),
			entry.Text))
	}
	
	return result.String()
}

// formatASSTime formats duration to ASS time format
func formatASSTime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	centiseconds := int(d.Nanoseconds()/10000000) % 100
	
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

// convertToVTT converts to WebVTT format
func convertToVTT(entries []SubtitleEntry) string {
	var result strings.Builder
	
	result.WriteString("WEBVTT\n\n")
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("%s --> %s\n", 
			formatVTTTime(entry.StartTime), 
			formatVTTTime(entry.EndTime)))
		result.WriteString(entry.Text)
		result.WriteString("\n\n")
	}
	
	return result.String()
}

// formatVTTTime formats duration to VTT time format
func formatVTTTime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Nanoseconds()/1000000) % 1000
	
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
	}
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)
}

// convertToSSA converts to SSA format (SubStation Alpha v4)
func convertToSSA(entries []SubtitleEntry, preserveStyle bool) string {
	var result strings.Builder
	
	// SSA header
	result.WriteString("[Script Info]\n")
	result.WriteString("Title: Converted by Subtitle Forge\n")
	result.WriteString("ScriptType: v4.00\n\n")
	
	result.WriteString("[V4 Styles]\n")
	result.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding\n")
	result.WriteString("Style: Default,Arial,20,16777215,255,0,0,0,0,1,2,0,2,10,10,10,0,1\n\n")
	
	result.WriteString("[Events]\n")
	result.WriteString("Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("Dialogue: Marked=0,%s,%s,Default,,0,0,0,,%s\n",
			formatSSATime(entry.StartTime),
			formatSSATime(entry.EndTime),
			entry.Text))
	}
	
	return result.String()
}

// formatSSATime formats duration to SSA time format (same as ASS but for v4.00)
func formatSSATime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	centiseconds := int(d.Nanoseconds()/10000000) % 100
	
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

// convertToSUB converts to SUB format (MicroDVD)
func convertToSUB(entries []SubtitleEntry) string {
	var result strings.Builder
	
	// MicroDVD uses frame numbers instead of time codes
	// Assuming 25 FPS as default (can be made configurable)
	fps := 25.0
	
	// First line should contain FPS info
	result.WriteString(fmt.Sprintf("{1}{1}%.1f\n", fps))
	
	for _, entry := range entries {
		startFrame := int(entry.StartTime.Seconds() * fps)
		endFrame := int(entry.EndTime.Seconds() * fps)
		
		// Replace newlines with | for MicroDVD format
		text := strings.ReplaceAll(entry.Text, "\n", "|")
		
		result.WriteString(fmt.Sprintf("{%d}{%d}%s\n", startFrame, endFrame, text))
	}
	
	return result.String()
}

// convertToASSAdvanced converts to ASS format with advanced options
func convertToASSAdvanced(entries []SubtitleEntry, options ConversionOptions) string {
	var result strings.Builder
	
	// Parse font color (hex to decimal)
	fontColor := parseHexColor(options.FontColor, 16777215) // Default white
	
	// ASS header
	result.WriteString("[Script Info]\n")
	result.WriteString("Title: Converted by Subtitle Forge\n")
	result.WriteString("ScriptType: v4.00+\n\n")
	
	result.WriteString("[V4+ Styles]\n")
	result.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	
	// Apply style template
	bold, italic, outline, shadow := getStyleTemplate(options.StyleTemplate)
	
	result.WriteString(fmt.Sprintf("Style: Default,%s,%d,%d,&H000000FF,&H00000000,&H80000000,%d,%d,0,0,100,100,0,0,1,%d,%d,2,%d,%d,%d,1\n\n",
		options.FontFamily, options.FontSize, fontColor, bold, italic, outline, shadow,
		options.MarginLeft, options.MarginRight, options.MarginVertical))
	
	result.WriteString("[Events]\n")
	result.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			formatASSTime(entry.StartTime),
			formatASSTime(entry.EndTime),
			entry.Text))
	}
	
	return result.String()
}

// convertToSSAAdvanced converts to SSA format with advanced options
func convertToSSAAdvanced(entries []SubtitleEntry, options ConversionOptions) string {
	var result strings.Builder
	
	// Parse font color for SSA (different format)
	fontColor := parseHexColorSSA(options.FontColor, 16777215)
	
	// SSA header
	result.WriteString("[Script Info]\n")
	result.WriteString("Title: Converted by Subtitle Forge\n")
	result.WriteString("ScriptType: v4.00\n\n")
	
	result.WriteString("[V4 Styles]\n")
	result.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding\n")
	
	// Apply style template
	bold, italic, outline, shadow := getStyleTemplate(options.StyleTemplate)
	
	result.WriteString(fmt.Sprintf("Style: Default,%s,%d,%d,255,0,0,%d,%d,1,%d,%d,2,%d,%d,%d,0,1\n\n",
		options.FontFamily, options.FontSize, fontColor, bold, italic, outline, shadow,
		options.MarginLeft, options.MarginRight, options.MarginVertical))
	
	result.WriteString("[Events]\n")
	result.WriteString("Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("Dialogue: Marked=0,%s,%s,Default,,0,0,0,,%s\n",
			formatSSATime(entry.StartTime),
			formatSSATime(entry.EndTime),
			entry.Text))
	}
	
	return result.String()
}

// convertToSUBAdvanced converts to SUB format with custom frame rate
func convertToSUBAdvanced(entries []SubtitleEntry, options ConversionOptions) string {
	var result strings.Builder
	
	// Use custom frame rate
	fps := options.FrameRate
	
	// First line should contain FPS info
	result.WriteString(fmt.Sprintf("{1}{1}%.3f\n", fps))
	
	for _, entry := range entries {
		startFrame := int(entry.StartTime.Seconds() * fps)
		endFrame := int(entry.EndTime.Seconds() * fps)
		
		// Replace newlines with | for MicroDVD format
		text := strings.ReplaceAll(entry.Text, "\n", "|")
		
		result.WriteString(fmt.Sprintf("{%d}{%d}%s\n", startFrame, endFrame, text))
	}
	
	return result.String()
}

// Helper functions for color and style processing
func parseHexColor(hexColor string, defaultColor int) int {
	if !strings.HasPrefix(hexColor, "#") {
		return defaultColor
	}
	
	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 {
		return defaultColor
	}
	
	if color, err := strconv.ParseInt(hexColor, 16, 64); err == nil {
		// Convert RGB to BGR for ASS format
		r := (color >> 16) & 0xFF
		g := (color >> 8) & 0xFF
		b := color & 0xFF
		// ASS format uses BGR (Blue-Green-Red) instead of RGB
		// So we swap R and B positions: BGR = B<<16 | G<<8 | R
		return int(b<<16 | g<<8 | r)
	}
	
	return defaultColor
}

func parseHexColorSSA(hexColor string, defaultColor int) int {
	// SSA also uses BGR format like ASS
	if !strings.HasPrefix(hexColor, "#") {
		return defaultColor
	}
	
	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 {
		return defaultColor
	}
	
	if color, err := strconv.ParseInt(hexColor, 16, 64); err == nil {
		// Convert RGB to BGR for SSA format (same as ASS)
		r := (color >> 16) & 0xFF
		g := (color >> 8) & 0xFF
		b := color & 0xFF
		// SSA format uses BGR (Blue-Green-Red) instead of RGB
		// So we swap R and B positions: BGR = B<<16 | G<<8 | R
		return int(b<<16 | g<<8 | r)
	}
	
	return defaultColor
}

func getStyleTemplate(template string) (bold, italic, outline, shadow int) {
	switch template {
	case "Bold":
		return 1, 0, 2, 0
	case "Italic":
		return 0, 1, 2, 0
	case "Bold Italic":
		return 1, 1, 2, 0
	case "Outline":
		return 0, 0, 3, 0
	case "Shadow":
		return 0, 0, 2, 2
	default: // "Default"
		return 0, 0, 2, 0
	}
}

// parseHexColorSimple parses a hex color string to NRGBA
func parseHexColorSimple(hexColor string) (color.NRGBA, bool) {
	var col color.NRGBA
	
	// Remove the # prefix if present
	if len(hexColor) > 0 && hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}
	
	// Parse RGB format (6 characters)
	if len(hexColor) == 6 {
		r, err1 := strconv.ParseUint(hexColor[0:2], 16, 8)
		g, err2 := strconv.ParseUint(hexColor[2:4], 16, 8)
		b, err3 := strconv.ParseUint(hexColor[4:6], 16, 8)
		if err1 == nil && err2 == nil && err3 == nil {
			col = color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			return col, true
		}
	}
	
	return col, false
}

// convertToTXT converts to plain text format
func convertToTXT(entries []SubtitleEntry) string {
	var result strings.Builder
	
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("[%s - %s]\n", 
			formatSRTTime(entry.StartTime), 
			formatSRTTime(entry.EndTime)))
		result.WriteString(entry.Text)
		result.WriteString("\n\n")
	}
	
	return result.String()
}

// ConversionOptions holds all conversion settings
type ConversionOptions struct {
	PreserveTiming   bool
	PreserveStyle    bool
	Encoding         string
	FrameRate        float64
	TimeOffset       float64
	RemoveFormatting bool
	TextCase         string
	FontFamily       string
	FontSize         int
	FontColor        string
	MarginLeft       int
	MarginRight      int
	MarginVertical   int
	StyleTemplate    string
}

// Helper functions for parsing strings
func parseFloat(s string, defaultVal float64) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return defaultVal
}

func parseInt(s string, defaultVal int) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultVal
}

// convertSubtitleFileAdvanced performs advanced subtitle conversion with all options
func convertSubtitleFileAdvanced(inputPath, inputFormat, outputFormat, outputDir string, 
	options ConversionOptions, progress *widget.ProgressBar, result *widget.Label) bool {
	
	// Determine output path
	var outputPath string
	if outputDir != "" {
		fileName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outputPath = filepath.Join(outputDir, fileName+"."+outputFormat)
	} else {
		outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "." + outputFormat
	}
	
	fyne.Do(func() {
		progress.SetValue(0.1)
		result.SetText(fmt.Sprintf("🔄 Converting %s to %s...", inputFormat, strings.ToUpper(outputFormat)))
	})
	
	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error reading input file: %v", err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(0.3)
		result.SetText("🔄 Parsing subtitle format...")
	})
	
	// Handle PGS format with OCR conversion
	var subtitles []SubtitleEntry
	if strings.ToUpper(inputFormat) == "PGS" {
		fyne.Do(func() {
			result.SetText("🔍 Converting PGS to SRT using OCR...")
		})
		
		// Convert PGS to SRT using the same method as Extract Subtitles tab
		tempSrtPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "_temp.srt"
		
		// Try pgsrip first if available
		var cmd *exec.Cmd
		var conversionMethod string
		
		// Ensure pgsrip is checked if path is empty
		if pgsripBinaryPath == "" {
			checkPgsrip() // This should set pgsripBinaryPath
		}
		
		if pgsripBinaryPath != "" {
			// Use pgsrip if available (same format as Extract Subtitles tab)
			cmd = exec.Command(pgsripBinaryPath, inputPath, tempSrtPath, "--verbose")
			conversionMethod = "pgsrip"
		} else if pgsToSrtScriptPath != "" && checkDeno() {
			// Fall back to Deno-based PGS to SRT script
			cmd = exec.Command("deno", "run", "--allow-read", "--allow-write", pgsToSrtScriptPath, inputPath, tempSrtPath)
			conversionMethod = "pgs-to-srt script"
		} else {
			fyne.Do(func() {
				result.SetText("❌ No PGS conversion tool available.\n\nPlease install either:\n1. pgsrip (recommended)\n2. PGS-to-SRT script with Deno\n\nCheck the Utilities tab for installation options.")
			})
			return false
		}
		
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("🔍 Converting PGS using %s...\nCommand: %s %s %s --verbose", conversionMethod, pgsripBinaryPath, inputPath, tempSrtPath))
		})
		
		err := cmd.Run()
		if err != nil {
			fyne.Do(func() {
				result.SetText(fmt.Sprintf("❌ Error converting PGS with %s: %v\n\nTry using the Extract Subtitles tab for PGS conversion, or check the Utilities tab for installation help.", conversionMethod, err))
			})
			return false
		}
		
		// Read the converted SRT file
		srtData, err := os.ReadFile(tempSrtPath)
		if err != nil {
			fyne.Do(func() {
				result.SetText(fmt.Sprintf("❌ Error reading converted SRT: %v", err))
			})
			return false
		}
		
		// Parse the SRT content
		subtitles, err = parseSRT(string(srtData))
		if err != nil {
			fyne.Do(func() {
				result.SetText(fmt.Sprintf("❌ Error parsing converted SRT: %v", err))
			})
			os.Remove(tempSrtPath) // Clean up
			return false
		}
		
		// Clean up temporary file
		os.Remove(tempSrtPath)
		
		fyne.Do(func() {
			result.SetText("✅ PGS successfully converted using OCR")
		})
	} else {
		// Parse input format normally
		var err error
		subtitles, err = parseSubtitleFile(string(inputData), inputFormat)
		if err != nil {
			fyne.Do(func() {
				result.SetText(fmt.Sprintf("❌ Error parsing %s format: %v", inputFormat, err))
			})
			return false
		}
	}
	
	fyne.Do(func() {
		progress.SetValue(0.5)
		result.SetText("🔄 Applying conversion options...")
	})
	
	// Apply conversion options
	subtitles = applyConversionOptions(subtitles, options)
	
	fyne.Do(func() {
		progress.SetValue(0.7)
		result.SetText("🔄 Converting to target format...")
	})
	
	// Convert to output format
	outputData, err := convertToFormatAdvanced(subtitles, outputFormat, options)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error converting to %s format: %v", outputFormat, err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(0.9)
		result.SetText("🔄 Writing output file...")
	})
	
	// Write output file
	err = os.WriteFile(outputPath, []byte(outputData), 0644)
	if err != nil {
		fyne.Do(func() {
			result.SetText(fmt.Sprintf("❌ Error writing output file: %v", err))
		})
		return false
	}
	
	fyne.Do(func() {
		progress.SetValue(1.0)
		result.SetText(fmt.Sprintf("✅ Successfully converted to: %s", outputPath))
	})
	
	return true
}

// applyConversionOptions applies various conversion options to subtitle entries
func applyConversionOptions(entries []SubtitleEntry, options ConversionOptions) []SubtitleEntry {
	var result []SubtitleEntry
	
	for _, entry := range entries {
		newEntry := entry
		
		// Apply time offset
		if options.TimeOffset != 0 {
			offset := time.Duration(options.TimeOffset * float64(time.Second))
			newEntry.StartTime += offset
			newEntry.EndTime += offset
			
			// Ensure times don't go negative
			if newEntry.StartTime < 0 {
				newEntry.StartTime = 0
			}
			if newEntry.EndTime < 0 {
				newEntry.EndTime = 0
			}
		}
		
		// Apply text processing
		text := newEntry.Text
		
		// Remove formatting tags if requested
		if options.RemoveFormatting {
			// Remove HTML tags
			text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
			// Remove ASS tags
			text = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(text, "")
		}
		
		// Apply case conversion
		switch options.TextCase {
		case "UPPERCASE":
			text = strings.ToUpper(text)
		case "lowercase":
			text = strings.ToLower(text)
		case "Title Case":
			text = strings.Title(strings.ToLower(text))
		}
		
		newEntry.Text = text
		result = append(result, newEntry)
	}
	
	return result
}

// convertToFormatAdvanced converts subtitles with advanced options
func convertToFormatAdvanced(entries []SubtitleEntry, format string, options ConversionOptions) (string, error) {
	switch strings.ToUpper(format) {
	case "SRT":
		return convertToSRT(entries), nil
	case "ASS":
		return convertToASSAdvanced(entries, options), nil
	case "SSA":
		return convertToSSAAdvanced(entries, options), nil
	case "VTT":
		return convertToVTT(entries), nil
	case "SUB":
		return convertToSUBAdvanced(entries, options), nil
	case "TXT":
		return convertToTXT(entries), nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

func createConvertSubtitlesTab(w fyne.Window) (*fyne.Container, func(string), func([]string)) {
	// Title
	convertTitle := widget.NewLabel("Convert Subtitles")
	convertTitle.TextStyle = fyne.TextStyle{Bold: true}
	convertTitle.Alignment = fyne.TextAlignCenter

	// Input file selection - support both single and batch
	var inputFile string
	var inputFormat string
	var convertFiles []string
	var convertBatchMode bool
	
	inputLabel := widget.NewLabel("No file selected")
	inputLabel.Wrapping = fyne.TextWrapWord
	
	// File list for batch mode (declare early)
	convertFileList := container.NewVBox()
	convertFileListScroll := container.NewScroll(convertFileList)
	convertFileListScroll.SetMinSize(fyne.NewSize(0, 150))
	
	// Function to update file list display (declare early)
	updateConvertFileList := func() {
		convertFileList.Objects = nil
		for i, filePath := range convertFiles {
			fileName := filepath.Base(filePath)
			format := detectSubtitleFormat(filePath)
			
			// Create remove button for each file
			removeBtn := widget.NewButton("Remove", nil)
			fileIndex := i // Capture index for closure
			removeBtn.OnTapped = func() {
				// Remove file from list
				if fileIndex < len(convertFiles) {
					convertFiles = append(convertFiles[:fileIndex], convertFiles[fileIndex+1:]...)
					// Refresh the list by clearing and rebuilding
					convertFileList.Objects = nil
					for j, filePath := range convertFiles {
						fileName := filepath.Base(filePath)
						format := detectSubtitleFormat(filePath)
						
						// Create new remove button
						newRemoveBtn := widget.NewButton("Remove", nil)
						newFileIndex := j
						newRemoveBtn.OnTapped = func() {
							// Simple removal without recursive calls
							if newFileIndex < len(convertFiles) {
								convertFiles = append(convertFiles[:newFileIndex], convertFiles[newFileIndex+1:]...)
								inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch conversion", len(convertFiles)))
							}
						}
						newRemoveBtn.Importance = widget.LowImportance
						
						fileRow := container.NewBorder(nil, nil, nil, newRemoveBtn,
							widget.NewLabel(fmt.Sprintf("%s (%s)", fileName, format)))
						convertFileList.Add(fileRow)
					}
					convertFileList.Refresh()
					inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch conversion", len(convertFiles)))
				}
			}
			removeBtn.Importance = widget.LowImportance
			
			fileRow := container.NewBorder(nil, nil, nil, removeBtn,
				widget.NewLabel(fmt.Sprintf("%s (%s)", fileName, format)))
			convertFileList.Add(fileRow)
		}
		convertFileList.Refresh()
	}
	
	// Single file selection button
	inputBtn := widget.NewButton("Select Subtitle File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			
			// Switch to single file mode
			convertBatchMode = false
			convertFiles = nil
			convertFileList.Objects = nil
			convertFileList.Refresh()
			
			inputFile = reader.URI().Path()
			fileName := filepath.Base(inputFile)
			inputFormat = detectSubtitleFormat(inputFile)
			
			if strings.ToUpper(inputFormat) == "PGS" {
				// Determine which conversion method will be used
				var conversionInfo string
				if pgsripBinaryPath != "" {
					conversionInfo = "This will be converted using OCR (pgsrip)."
				} else if pgsToSrtScriptPath != "" && checkDeno() {
					conversionInfo = "This will be converted using OCR (pgs-to-srt script with Deno)."
				} else {
					conversionInfo = "⚠️ No PGS conversion tool available. Please install pgsrip or PGS-to-SRT script."
				}
				
				inputLabel.SetText(fmt.Sprintf("File: %s\nDetected Format: %s\n\n🔍 PGS format detected! %s", fileName, inputFormat, conversionInfo))
			} else {
				inputLabel.SetText(fmt.Sprintf("File: %s\nDetected Format: %s", fileName, inputFormat))
			}
		}, w)
	})
	inputBtn.Importance = widget.MediumImportance
	
	// Batch file selection button
	batchBtn := widget.NewButton("Select Multiple Files (Batch)", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			
			// Switch to batch mode
			convertBatchMode = true
			inputFile = ""
			inputFormat = ""
			
			// Find all subtitle files in the directory
			convertFiles = []string{}
			supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt"}
			
			filepath.Walk(uri.Path(), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() {
					ext := strings.ToLower(filepath.Ext(path))
					for _, supportedExt := range supportedExts {
						if ext == supportedExt {
							convertFiles = append(convertFiles, path)
							break
						}
					}
				}
				return nil
			})
			
			// Update UI
			updateConvertFileList()
			inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch conversion", len(convertFiles)))
		}, w)
	})
	batchBtn.Importance = widget.MediumImportance
	
	// Clear files button
	clearBtn := widget.NewButton("Clear Files", func() {
		convertBatchMode = false
		convertFiles = nil
		inputFile = ""
		inputFormat = ""
		convertFileList.Objects = nil
		convertFileList.Refresh()
		inputLabel.SetText("No file selected")
	})
	clearBtn.Importance = widget.LowImportance

	// Format selection
	outputFormats := []string{"SRT", "ASS", "SSA", "VTT", "SUB", "TXT"}
	outputFormatSelect := widget.NewSelect(outputFormats, nil)
	outputFormatSelect.SetSelected("SRT") // Default to SRT

	// Output directory selection
	var outputDir string
	outputLabel := widget.NewLabel("Output: Same as input file")
	
	outputBtn := widget.NewButton("Choose Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDir = uri.Path()
			outputLabel.SetText(fmt.Sprintf("Output Directory: %s", outputDir))
		}, w)
	})

	// Basic conversion options
	preserveTimingCheck := widget.NewCheck("Preserve original timing", nil)
	preserveTimingCheck.SetChecked(true)
	
	preserveStyleCheck := widget.NewCheck("Preserve styling (when possible)", nil)
	preserveStyleCheck.SetChecked(true)
	
	encodingSelect := widget.NewSelect([]string{"UTF-8", "UTF-16", "Windows-1252", "ISO-8859-1"}, nil)
	encodingSelect.SetSelected("UTF-8")

	// Frame rate options (for SUB format)
	frameRateSelect := widget.NewSelect([]string{"23.976", "24", "25", "29.97", "30", "50", "59.94", "60"}, nil)
	frameRateSelect.SetSelected("25") // Default

	// Time offset options
	timeOffsetEntry := widget.NewEntry()
	timeOffsetEntry.SetPlaceHolder("0.0")
	timeOffsetEntry.SetText("0.0")

	// Text processing options
	removeFormattingCheck := widget.NewCheck("Remove formatting tags", nil)
	caseSelect := widget.NewSelect([]string{"Keep Original", "UPPERCASE", "lowercase", "Title Case"}, nil)
	caseSelect.SetSelected("Keep Original")

	// ASS/SSA specific options
	fontFamilySelect := widget.NewSelect([]string{
		"Arial", "Helvetica", "Times New Roman", "Georgia", "Verdana", 
		"Tahoma", "Trebuchet MS", "Comic Sans MS", "Impact", "Lucida Console",
		"Courier New", "Palatino", "Garamond", "Bookman", "Avant Garde",
		"Century Gothic", "Franklin Gothic", "Optima", "Futura", "Calibri",
		"Segoe UI", "Open Sans", "Roboto", "Lato", "Montserrat", "Custom...",
	}, nil)
	fontFamilySelect.SetSelected("Arial")
	
	// Set up custom font callback after creation
	fontFamilySelect.OnChanged = func(selected string) {
		// Handle custom font selection
		if selected == "Custom..." {
			// Show entry dialog for custom font
			customEntry := widget.NewEntry()
			customEntry.SetPlaceHolder("Enter custom font name")
			
			dialog.ShowForm("Custom Font", "OK", "Cancel", []*widget.FormItem{
				widget.NewFormItem("Font Name", customEntry),
			}, func(submitted bool) {
				if submitted && customEntry.Text != "" {
					// Add custom font to the list and select it
					options := fontFamilySelect.Options
					customFont := customEntry.Text
					// Remove "Custom..." temporarily
					options = options[:len(options)-1]
					// Add custom font and "Custom..." back
					options = append(options, customFont, "Custom...")
					fontFamilySelect.Options = options
					fontFamilySelect.SetSelected(customFont)
					fontFamilySelect.Refresh()
				}
			}, w)
		}
	}

	fontSizeEntry := widget.NewEntry()
	fontSizeEntry.SetPlaceHolder("20")
	fontSizeEntry.SetText("20")

	// Color picker for font color
	var selectedFontColor = "#FFFFFF"
	fontColorPreview := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	fontColorPreview.SetMinSize(fyne.NewSize(30, 25))
	
	fontColorLabel := widget.NewLabel("#FFFFFF")
	fontColorLabel.TextStyle = fyne.TextStyle{Monospace: true}
	
	fontColorButton := widget.NewButton("Choose Color", func() {
		// Parse current color
		currentColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		if col, ok := parseHexColorSimple(selectedFontColor); ok {
			currentColor = col
		}
		
		// Create color picker dialog
		colorPicker := NewColorPicker(currentColor, func(newColor color.NRGBA) {
			// Update preview and label
			fontColorPreview.FillColor = newColor
			fontColorPreview.Refresh()
			selectedFontColor = fmt.Sprintf("#%02X%02X%02X", newColor.R, newColor.G, newColor.B)
			fontColorLabel.SetText(selectedFontColor)
		})
		
		// Show color picker in dialog
		dialog.ShowCustom("Choose Font Color", "OK", colorPicker, w)
	})

	marginLeftEntry := widget.NewEntry()
	marginLeftEntry.SetPlaceHolder("10")
	marginLeftEntry.SetText("10")

	marginRightEntry := widget.NewEntry()
	marginRightEntry.SetPlaceHolder("10")
	marginRightEntry.SetText("10")

	marginVerticalEntry := widget.NewEntry()
	marginVerticalEntry.SetPlaceHolder("10")
	marginVerticalEntry.SetText("10")

	styleTemplateSelect := widget.NewSelect([]string{"Default", "Bold", "Italic", "Bold Italic", "Outline", "Shadow"}, nil)
	styleTemplateSelect.SetSelected("Default")

	// Progress bar and result
	convertProgress := widget.NewProgressBar()
	convertProgress.Hide()
	
	convertResult := widget.NewLabel("")
	convertResult.Wrapping = fyne.TextWrapWord
	convertResultScroll := container.NewScroll(convertResult)
	convertResultScroll.SetMinSize(fyne.NewSize(0, 100))

	// Convert button
	convertBtn := widget.NewButton("Convert Subtitle", func() {
		// Check if we have files to convert
		if !convertBatchMode && inputFile == "" {
			convertResult.SetText("❌ Please select an input file first")
			return
		}
		if convertBatchMode && len(convertFiles) == 0 {
			convertResult.SetText("❌ Please select files for batch conversion first")
			return
		}
		
		outputFormat := strings.ToLower(outputFormatSelect.Selected)
		if outputFormat == "" {
			convertResult.SetText("❌ Please select an output format")
			return
		}

		// Show progress
		convertProgress.Show()
		
		if convertBatchMode {
			convertResult.SetText(fmt.Sprintf("🔄 Converting %d subtitle files...", len(convertFiles)))
		} else {
			convertResult.SetText("🔄 Converting subtitle file...")
		}
		
		// Perform conversion in goroutine
		go func() {
			// Create conversion options struct
			options := ConversionOptions{
				PreserveTiming:    preserveTimingCheck.Checked,
				PreserveStyle:     preserveStyleCheck.Checked,
				Encoding:          encodingSelect.Selected,
				FrameRate:         parseFloat(frameRateSelect.Selected, 25.0),
				TimeOffset:        parseFloat(timeOffsetEntry.Text, 0.0),
				RemoveFormatting:  removeFormattingCheck.Checked,
				TextCase:          caseSelect.Selected,
				FontFamily:        fontFamilySelect.Selected,
				FontSize:          parseInt(fontSizeEntry.Text, 20),
				FontColor:         selectedFontColor,
				MarginLeft:        parseInt(marginLeftEntry.Text, 10),
				MarginRight:       parseInt(marginRightEntry.Text, 10),
				MarginVertical:    parseInt(marginVerticalEntry.Text, 10),
				StyleTemplate:     styleTemplateSelect.Selected,
			}
			
			if convertBatchMode {
				// Batch conversion
				successCount := 0
				totalFiles := len(convertFiles)
				
				for i, filePath := range convertFiles {
					fileFormat := detectSubtitleFormat(filePath)
					fileName := filepath.Base(filePath)
					
					fyne.Do(func() {
						convertProgress.SetValue(float64(i) / float64(totalFiles))
						convertResult.SetText(fmt.Sprintf("🔄 Converting file %d/%d: %s", i+1, totalFiles, fileName))
					})
					
					success := convertSubtitleFileAdvanced(filePath, fileFormat, outputFormat, outputDir, 
						options, convertProgress, convertResult)
					
					if success {
						successCount++
					}
				}
				
				fyne.Do(func() {
					convertProgress.SetValue(1.0)
					convertProgress.Hide()
					convertResult.SetText(fmt.Sprintf("✅ Batch conversion completed: %d/%d files successfully converted to %s format", 
						successCount, totalFiles, strings.ToUpper(outputFormat)))
				})
			} else {
				// Single file conversion
				success := convertSubtitleFileAdvanced(inputFile, inputFormat, outputFormat, outputDir, 
					options, convertProgress, convertResult)
				
				fyne.Do(func() {
					convertProgress.Hide()
					if success {
						convertResult.SetText(fmt.Sprintf("✅ Successfully converted %s to %s format", 
							filepath.Base(inputFile), strings.ToUpper(outputFormat)))
					}
				})
			}
		}()
	})
	convertBtn.Importance = widget.HighImportance

	// Layout
	fileSelectionGroup := widget.NewCard("Input Files", "", container.NewVBox(
		container.NewHBox(inputBtn, batchBtn, clearBtn),
		inputLabel,
		convertFileListScroll,
	))

	formatGroup := widget.NewCard("Output Format", "", container.NewVBox(
		widget.NewLabel("Convert to:"),
		outputFormatSelect,
		widget.NewSeparator(),
		outputBtn,
		outputLabel,
	))

	// Create organized option groups
	basicOptionsGroup := widget.NewCard("Basic Options", "", container.NewVBox(
		preserveTimingCheck,
		preserveStyleCheck,
		widget.NewSeparator(),
		widget.NewLabel("Output Encoding:"),
		encodingSelect,
	))

	timingOptionsGroup := widget.NewCard("Timing Options", "", container.NewVBox(
		widget.NewLabel("Frame Rate (for SUB format):"),
		frameRateSelect,
		widget.NewSeparator(),
		widget.NewLabel("Time Offset (seconds):"),
		timeOffsetEntry,
	))

	textOptionsGroup := widget.NewCard("Text Processing", "", container.NewVBox(
		removeFormattingCheck,
		widget.NewSeparator(),
		widget.NewLabel("Text Case:"),
		caseSelect,
	))

	styleOptionsGroup := widget.NewCard("ASS/SSA Style Options", "", container.NewVBox(
		widget.NewLabel("Font Family:"),
		fontFamilySelect,
		container.NewHBox(
			widget.NewLabel("Size:"), fontSizeEntry,
		),
		widget.NewLabel("Font Color:"),
		container.NewHBox(
			fontColorPreview,
			fontColorLabel,
			fontColorButton,
		),
		container.NewHBox(
			widget.NewLabel("L:"), marginLeftEntry,
			widget.NewLabel("R:"), marginRightEntry,
			widget.NewLabel("V:"), marginVerticalEntry,
		),
		widget.NewLabel("Style Template:"),
		styleTemplateSelect,
	))

	convertGroup := widget.NewCard("Convert", "", container.NewVBox(
		convertBtn,
		convertProgress,
	))

	resultsGroup := widget.NewCard("Results", "", convertResultScroll)

	// Create a function to handle drag & drop file loading
	loadDroppedFile := func(filePath string) {
		// Switch to single file mode
		convertBatchMode = false
		convertFiles = nil
		convertFileList.Objects = nil
		convertFileList.Refresh()
		
		inputFile = filePath
		fileName := filepath.Base(inputFile)
		inputFormat = detectSubtitleFormat(inputFile)
		
		if strings.ToUpper(inputFormat) == "PGS" {
			// Determine which conversion method will be used
			var conversionInfo string
			if pgsripBinaryPath != "" {
				conversionInfo = "This will be converted using OCR (pgsrip)."
			} else if pgsToSrtScriptPath != "" && checkDeno() {
				conversionInfo = "This will be converted using OCR (pgs-to-srt script with Deno)."
			} else {
				conversionInfo = "⚠️ No PGS conversion tool available. Please install pgsrip or PGS-to-SRT script."
			}
			
			inputLabel.SetText(fmt.Sprintf("File: %s\nDetected Format: %s\n\n🔍 PGS format detected! %s", fileName, inputFormat, conversionInfo))
			convertResult.SetText("📝 PGS (Presentation Graphics Stream) is a bitmap subtitle format.\n\nThis format will be automatically converted to text using OCR when you click 'Convert Subtitle'.\n\nThe same conversion tools used in the Extract Subtitles tab will be used here.")
		} else {
			inputLabel.SetText(fmt.Sprintf("File: %s\nDetected Format: %s", fileName, inputFormat))
			convertResult.SetText("File loaded successfully. Select output format and click 'Convert Subtitle' to begin.")
		}
	}
	
	// Create a function to handle multiple file drops for batch mode
	loadDroppedFiles := func(filePaths []string) {
		// Switch to batch mode
		convertBatchMode = true
		inputFile = ""
		inputFormat = ""
		
		// Filter for supported subtitle files
		convertFiles = []string{}
		supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt"}
		
		for _, filePath := range filePaths {
			ext := strings.ToLower(filepath.Ext(filePath))
			for _, supportedExt := range supportedExts {
				if ext == supportedExt {
					convertFiles = append(convertFiles, filePath)
					break
				}
			}
		}
		
		// Update UI
		updateConvertFileList()
		inputLabel.SetText(fmt.Sprintf("%d subtitle files selected for batch conversion", len(convertFiles)))
		convertResult.SetText("Multiple files loaded for batch conversion. Select output format and click 'Convert Subtitle' to begin.")
	}

	convertTabContent := container.NewVBox(
		container.NewPadded(convertTitle),
		fileSelectionGroup,
		formatGroup,
		basicOptionsGroup,
		timingOptionsGroup,
		textOptionsGroup,
		styleOptionsGroup,
		convertGroup,
		resultsGroup,
	)

	return convertTabContent, loadDroppedFile, loadDroppedFiles
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
			utilitiesResult.SetText(setLogMessage(LogInfo, "MKV File Selected", "Selected MKV file: "+filePath))
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
			utilitiesResult.SetText(setLogMessage(LogInfo, "SRT File Selected", "Selected SRT file: "+filePath))
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

		utilitiesResult.SetText(setLogMessage(LogInfo, "Getting MKV Information", "Getting MKV information...\n"))

		// Run mkvinfo command
		go func() {
			var cmd *exec.Cmd
			// Since mkvinfo is part of the same package as mkvextract and mkvmerge,
			// we can derive its path from the mkvextract path if available
			if mkvextractBinaryPath != "" {
				// Get the directory of mkvextract and use it for mkvinfo
				mkvToolsDir := filepath.Dir(mkvextractBinaryPath)
				mkvinfoPath := filepath.Join(mkvToolsDir, "mkvinfo")

				// Check if mkvinfo exists at the expected path
				if _, err := os.Stat(mkvinfoPath); err == nil {
					cmd = exec.Command(mkvinfoPath, mkvPath)
					fmt.Println("[DEBUG] Using derived mkvinfo path:", mkvinfoPath)
				} else {
					// Fallback to PATH lookup
					cmd = exec.Command("mkvinfo", mkvPath)
					fmt.Println("[DEBUG] Could not find mkvinfo at derived path, using default PATH lookup")
				}
			} else {
				// Fallback to PATH lookup
				cmd = exec.Command("mkvinfo", mkvPath)
				fmt.Println("[DEBUG] No stored mkvextract path to derive mkvinfo path from, using default PATH lookup")
			}
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText(setLogMessage(LogSuccess, "MKV Information", "MKV Information for: "+mkvPath+"\n\n"+string(output)))
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

		utilitiesResult.SetText(setLogMessage(LogInfo, "Extracting Chapters", "Extracting chapters to: "+outputPath+"\n"))

		// Run mkvextract command for chapters
		go func() {
			var cmd *exec.Cmd
			if mkvextractBinaryPath != "" {
				// Use the stored full path to mkvextract
				cmd = exec.Command(mkvextractBinaryPath, mkvPath, "chapters", outputPath)
				fmt.Println("[DEBUG] Using stored mkvextract path for chapters extraction:", mkvextractBinaryPath)
			} else {
				// Fallback to PATH lookup
				cmd = exec.Command("mkvextract", mkvPath, "chapters", outputPath)
				fmt.Println("[DEBUG] No stored mkvextract path for chapters extraction, using default PATH lookup")
			}
			output, err := cmd.CombinedOutput()

			fyne.Do(func() {
				if err != nil {
					utilitiesResult.SetText(utilitiesResult.Text + "\nError: " + err.Error())
					return
				}

				utilitiesResult.SetText(setLogMessage(LogSuccess, "Chapters Extracted", utilitiesResult.Text+"\nChapters extracted successfully to: "+outputPath+"\n"+string(output)))
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

		utilitiesResult.SetText(setLogMessage(LogInfo, "Fixing SRT Encoding", "Fixing SRT encoding...\n"))

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

				utilitiesResult.SetText(setLogMessage(LogSuccess, "SRT Encoding Fixed", utilitiesResult.Text+"\nSRT encoding fixed successfully.\nOriginal backup saved to: "+backupPath+"\n"+string(output)))
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
				utilitiesResult.SetText(setLogMessage(LogInfo, "Adjusting SRT Timing", "Adjusting SRT timing with offset: "+offset+" seconds...\n"))

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
						utilitiesResult.SetText(setLogMessage(LogSuccess, "SRT Timing Adjusted", utilitiesResult.Text+"\nSRT timing adjusted successfully.\nOriginal backup saved to: "+backupPath))
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

// loadTracksForFile loads tracks for a specific MKV file (for batch processing)
func loadTracksForFile(mkvPath string) bool {
	var cmd *exec.Cmd
	if mkvmergeBinaryPath != "" {
		cmd = exec.Command(mkvmergeBinaryPath, "-J", mkvPath)
	} else {
		cmd = exec.Command("mkvmerge", "-J", mkvPath)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	
	// Parse JSON output (simplified - just check if we have tracks)
	return len(output) > 0 && strings.Contains(string(output), "tracks")
}

// extractAllSubtitleTracks extracts all subtitle tracks from an MKV file (for batch processing)
func extractAllSubtitleTracks(mkvPath, outDir string) bool {
	// Get track info first
	var cmd *exec.Cmd
	if mkvmergeBinaryPath != "" {
		cmd = exec.Command(mkvmergeBinaryPath, "-J", mkvPath)
	} else {
		cmd = exec.Command("mkvmerge", "-J", mkvPath)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	
	// Parse JSON to find subtitle tracks
	var mkvInfo struct {
		Tracks []struct {
			ID         int    `json:"id"`
			Type       string `json:"type"`
			Codec      string `json:"codec"`
			Properties struct {
				Language string `json:"language"`
				TrackName string `json:"track_name"`
			} `json:"properties"`
		} `json:"tracks"`
	}
	
	err = json.Unmarshal(output, &mkvInfo)
	if err != nil {
		return false
	}
	
	// Extract subtitle tracks
	mkvBaseName := filepath.Base(mkvPath)
	mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))
	
	successCount := 0
	for _, track := range mkvInfo.Tracks {
		if track.Type == "subtitles" {
			// Determine file extension based on codec
			var ext string
			// Determine extension based on codec with comprehensive matching
			codecLower := strings.ToLower(track.Codec)
			switch {
			case strings.Contains(codecLower, "subrip") || strings.Contains(codecLower, "srt"):
				ext = "srt"
			case strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv"):
				ext = "sup"
			case strings.Contains(codecLower, "vobsub"):
				ext = "sub"
			case strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "advanced substation"):
				ext = "ass"
			case strings.Contains(codecLower, "ssa"):
				ext = "ssa"
			default:
				ext = "srt" // Default to SRT
			}
			
			lang := track.Properties.Language
			if lang == "" {
				lang = "und"
			}
			
			outFile := fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, track.ID, lang, ext)
			
			// Extract the track
			var extractCmd *exec.Cmd
			if mkvextractBinaryPath != "" {
				extractCmd = exec.Command(mkvextractBinaryPath, "tracks", mkvPath, fmt.Sprintf("%d:%s", track.ID, outFile))
			} else {
				extractCmd = exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", track.ID, outFile))
			}
			extractCmd.Dir = outDir
			
			_, err := extractCmd.CombinedOutput()
			if err == nil {
				successCount++
			}
		}
	}
	
	return successCount > 0
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

// Log types for styling the output
const (
	LogInfo    = "INFO"
	LogSuccess = "SUCCESS"
	LogError   = "ERROR"
	LogExtract = "EXTRACT"
	LogConvert = "CONVERT"
)

// setLogMessage formats a log message with an icon and title.
func setLogMessage(logType, title, message string) string {
	var icon string
	switch logType {
	case LogInfo:
		icon = "ℹ️"
	case LogSuccess:
		icon = "✅"
	case LogError:
		icon = "❌"
	case LogExtract:
		icon = "🎬"
	case LogConvert:
		icon = "🔄"
	default:
		icon = "➡️"
	}
	return fmt.Sprintf("%s %s\n%s", icon, title, message)
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

	// Global variables for batch processing
	var mkvFiles []string
	var batchMode bool

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

				result.SetText(setLogMessage(LogInfo, "MKV File Loaded", "MKV file dropped and loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file."))
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

	// Create file list widget for batch processing
	var fileList *widget.List
	fileList = widget.NewList(
		func() int { return len(mkvFiles) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(mkvFiles) {
				return
			}
			container := item.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			removeBtn := container.Objects[1].(*widget.Button)
			
			label.SetText(filepath.Base(mkvFiles[id]))
			removeBtn.OnTapped = func() {
				// Remove file from list
				mkvFiles = append(mkvFiles[:id], mkvFiles[id+1:]...)
				fileList.Refresh()
				if len(mkvFiles) == 0 {
					batchMode = false
					selectedFile.SetText("No files selected")
				} else {
					selectedFile.SetText(fmt.Sprintf("%d MKV files selected for batch processing", len(mkvFiles)))
				}
			}
		},
	)

	// Create file list container with scroll
	fileListContainer := container.NewBorder(
		widget.NewLabel("Selected Files:"),
		nil, nil, nil,
		container.NewScroll(fileList),
	)
	fileListContainer.Hide() // Initially hidden

	// Button to select single MKV file
	fileBtn := widget.NewButton("Select Single MKV File", func() {
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

			// Reset to single file mode
			batchMode = false
			mkvFiles = []string{}
			fileList.Refresh()
			fileListContainer.Hide()
			
			mkvPath = filePath
			selectedFile.SetText(mkvPath)

			// Set output directory to the same directory as the MKV file
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)

			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()

			result.SetText(setLogMessage(LogInfo, "MKV File Loaded", "MKV file loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file."))
		}, w)

		fd.SetFilter(filter)
		fd.Show()
	})

	// Button to select multiple MKV files for batch processing
	batchBtn := widget.NewButton("Select Multiple MKV Files (Batch)", func() {
		// Use folder selection for batch processing
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			folderPath := uri.Path()
			
			// Find all MKV files in the selected folder
			var foundFiles []string
			err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Continue walking
				}
				if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".mkv" {
					foundFiles = append(foundFiles, path)
				}
				return nil
			})

			if err != nil {
				dialog.ShowError(fmt.Errorf("Error scanning folder: %v", err), w)
				return
			}

			if len(foundFiles) == 0 {
				dialog.ShowInformation("No MKV Files", "No MKV files found in the selected folder.", w)
				return
			}

			// Set batch mode
			batchMode = true
			mkvFiles = foundFiles
			fileList.Refresh()
			fileListContainer.Show()

			// Set output directory to the selected folder
			outDir = folderPath
			selectedDir.SetText(outDir)

			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()

			selectedFile.SetText(fmt.Sprintf("%d MKV files selected for batch processing", len(mkvFiles)))
			result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Found %d MKV files. Click 'Start Extraction' to process all files.", len(mkvFiles))))
			
		}, w)
	})

	// Button to clear file selection
	clearBtn := widget.NewButton("Clear Selection", func() {
		batchMode = false
		mkvFiles = []string{}
		mkvPath = ""
		fileList.Refresh()
		fileListContainer.Hide()
		selectedFile.SetText("No files selected")
		selectedDir.SetText("No output directory selected")
		trackItems = []*TrackItem{}
		trackList.Objects = nil
		trackList.Refresh()
		result.SetText("Select MKV file(s) to begin.")
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
		if batchMode {
			// In batch mode, load tracks from all files for user selection
			if len(mkvFiles) == 0 {
				dialog.ShowError(fmt.Errorf("Please select MKV files for batch processing first."), w)
				return
			}
			
			// Load tracks from all MKV files
			go func() {
				fyne.Do(func() {
					result.SetText(setLogMessage(LogInfo, "Loading Batch Tracks", "Analyzing all MKV files for subtitle tracks..."))
					progress.Max = float64(len(mkvFiles))
					progress.SetValue(0)
				})
				
				// Clear previous tracks
				trackItems = []*TrackItem{}
				
				totalTracks := 0
				for fileIndex, mkvFile := range mkvFiles {
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Analyzing file %d/%d: %s", fileIndex+1, len(mkvFiles), filepath.Base(mkvFile)))
						progress.SetValue(float64(fileIndex))
					})
					
					// Get track info for this file
					var cmd *exec.Cmd
					if mkvmergeBinaryPath != "" {
						cmd = exec.Command(mkvmergeBinaryPath, "-J", mkvFile)
					} else {
						cmd = exec.Command("mkvmerge", "-J", mkvFile)
					}
					
					output, err := cmd.Output()
					if err != nil {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to analyze: %s", filepath.Base(mkvFile)))
						})
						continue
					}
					
					// Parse JSON output
					var mkvInfo struct {
						Tracks []struct {
							ID         int    `json:"id"`
							Type       string `json:"type"`
							Codec      string `json:"codec"`
							Properties struct {
								Language  string `json:"language"`
								TrackName string `json:"track_name"`
							} `json:"properties"`
						} `json:"tracks"`
					}
					
					err = json.Unmarshal(output, &mkvInfo)
					if err != nil {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to parse: %s", filepath.Base(mkvFile)))
						})
						continue
					}
					
					// Add subtitle tracks to the list
					for _, track := range mkvInfo.Tracks {
						if track.Type == "subtitles" {
							lang := track.Properties.Language
							if lang == "" {
								lang = "und"
							}
							
							trackName := track.Properties.TrackName
							if trackName == "" {
								trackName = "Untitled"
							}
							
							// Create track item with file info
							trackItem := &TrackItem{
								Num:      track.ID,
								Lang:     lang,
								Codec:    track.Codec,
								Name:     trackName,
								FilePath: mkvFile, // Store source file path
								Check:    widget.NewCheck("", nil),
								Status:   widget.NewLabel("Ready"),
							}
							
							// Handle PGS subtitles with OCR option
							if track.Codec == "hdmv_pgs_subtitle" {
								trackItem.ConvertOCR = widget.NewCheck("", nil)
							}
							
							trackItems = append(trackItems, trackItem)
							totalTracks++
						}
					}
				}
				
				// Update UI with all tracks
				fyne.Do(func() {
					progress.SetValue(float64(len(mkvFiles)))
					currentTrackLabel.SetText(fmt.Sprintf("Found %d subtitle tracks across %d files", totalTracks, len(mkvFiles)))
					
					// Update track list
					trackList.Objects = nil
					for _, tt := range trackItems {
						// Show file name + track info
						fileName := filepath.Base(tt.FilePath)
						trackInfo := widget.NewLabel(fmt.Sprintf("%s - Track %d: %s (%s) %s", fileName, tt.Num, tt.Lang, tt.Codec, tt.Name))
						
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
					
					result.SetText(setLogMessage(LogSuccess, "Batch Tracks Loaded", fmt.Sprintf("Found %d subtitle tracks across %d MKV files. Select the tracks you want to extract, then click 'Start Extraction'.", totalTracks, len(mkvFiles))))
				})
			}()
			return
		}
		
		// Single file mode
		if mkvPath == "" {
			dialog.ShowError(fmt.Errorf("Please select or drag & drop an MKV file first."), w)
			return
		}

		// Run mkvmerge to get track info
		var cmd *exec.Cmd
		if mkvmergeBinaryPath != "" {
			// Use the stored full path to mkvmerge
			cmd = exec.Command(mkvmergeBinaryPath, "-J", mkvPath)
			fmt.Println("[DEBUG] Using stored mkvmerge path:", mkvmergeBinaryPath)
		} else {
			// Fallback to PATH lookup (though this likely won't work if checkMkvmerge failed)
			cmd = exec.Command("mkvmerge", "-J", mkvPath)
			fmt.Println("[DEBUG] No stored mkvmerge path, using default PATH lookup")
		}

		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error running mkvmerge: %v (path: %s)", err, mkvmergeBinaryPath), w)
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

		result.SetText(setLogMessage(LogSuccess, "Tracks Loaded", "Tracks loaded. Select the tracks you want to extract, then click 'Start Extraction'"))
	})

	// Button to start extraction of selected tracks
	startExtractBtn := widget.NewButton("Start Extraction", func() {
		if batchMode {
			// Batch processing mode
			if len(mkvFiles) == 0 || outDir == "" {
				dialog.ShowError(fmt.Errorf("Please select MKV files and output directory for batch processing."), w)
				return
			}
			
			// Start batch processing
			go func() {
				totalFiles := len(mkvFiles)
				successCount := 0
				failureCount := 0
				
				fyne.Do(func() {
					result.SetText(setLogMessage(LogInfo, "Batch Processing Started", fmt.Sprintf("Processing %d MKV files...", totalFiles)))
					progress.Max = float64(totalFiles)
					progress.SetValue(0)
				})
				
				for fileIndex, currentMkvPath := range mkvFiles {
					mkvPath = currentMkvPath // Set current file for processing
					
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Processing file %d/%d: %s", fileIndex+1, totalFiles, filepath.Base(currentMkvPath)))
						progress.SetValue(float64(fileIndex))
					})
					
					// Load tracks for current file (simplified check)
					if !loadTracksForFile(currentMkvPath) {
						failureCount++
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Failed to load tracks for: %s", filepath.Base(currentMkvPath)))
						})
						continue
					}
					
					// Extract selected tracks for this file
					selectedTracksForFile := []*TrackItem{}
					for _, t := range trackItems {
						if t.Check.Checked && t.FilePath == currentMkvPath {
							selectedTracksForFile = append(selectedTracksForFile, t)
						}
					}
					
					if len(selectedTracksForFile) == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n⏭️ Skipped (no tracks selected): %s", filepath.Base(currentMkvPath)))
						})
						continue
					}
					
					// Extract selected tracks
					fileSuccess := true
					for _, track := range selectedTracksForFile {
						fyne.Do(func() {
							track.Status.SetText("Extracting...")
						})
						
						// Determine file extension based on codec with comprehensive matching
						var ext string
						codecLower := strings.ToLower(track.Codec)
						switch {
						case strings.Contains(codecLower, "subrip") || strings.Contains(codecLower, "srt"):
							ext = "srt"
						case strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv"):
							if track.ConvertOCR != nil && track.ConvertOCR.Checked {
								ext = "srt" // Convert PGS to SRT
							} else {
								ext = "sup"
							}
						case strings.Contains(codecLower, "vobsub"):
							ext = "sub"
						case strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "advanced substation"):
							ext = "ass"
						case strings.Contains(codecLower, "ssa"):
							ext = "ssa"
						default:
							ext = "srt"
						}
						
						mkvBaseName := filepath.Base(currentMkvPath)
						mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))
						outFile := fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, track.Num, track.Lang, ext)
						
						// Extract the track
						var extractCmd *exec.Cmd
						if mkvextractBinaryPath != "" {
							extractCmd = exec.Command(mkvextractBinaryPath, "tracks", currentMkvPath, fmt.Sprintf("%d:%s", track.Num, outFile))
						} else {
							extractCmd = exec.Command("mkvextract", "tracks", currentMkvPath, fmt.Sprintf("%d:%s", track.Num, outFile))
						}
						extractCmd.Dir = outDir
						
						_, err := extractCmd.CombinedOutput()
						if err != nil {
							fileSuccess = false
							fyne.Do(func() {
								track.Status.SetText("Failed")
							})
						} else {
							fyne.Do(func() {
								track.Status.SetText("Done")
							})
						}
					}
					
					if fileSuccess {
						successCount++
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n✅ Successfully processed: %s (%d tracks)", filepath.Base(currentMkvPath), len(selectedTracksForFile)))
						})
					} else {
						failureCount++
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n❌ Partially failed: %s", filepath.Base(currentMkvPath)))
						})
					}
				}
				
				// Final batch processing results
				fyne.Do(func() {
					progress.SetValue(float64(totalFiles))
					currentTrackLabel.SetText("Batch processing completed")
					result.SetText(result.Text + fmt.Sprintf("\n\n🎬 Batch Processing Complete\n✅ Success: %d files\n❌ Failed: %d files\n📁 Output: %s", successCount, failureCount, outDir))
				})
				
				// Show completion notification
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Batch Processing Complete",
					Content: fmt.Sprintf("Processed %d files. %d successful, %d failed.", totalFiles, successCount, failureCount),
				})
			}()
			return
		}
		
		// Single file processing mode
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
				result.SetText(setLogMessage(LogInfo, "Extraction Started", "Extracting selected tracks..."))
				progress.Max = float64(len(selected))
				progress.SetValue(0)
			})

			tracksDone := 0
			var output []byte
			var err error

			for i, t := range selected {
				// Update UI on main thread
				fyne.Do(func() {
					currentTrackLabel.SetText(setLogMessage(LogExtract, fmt.Sprintf("Extracting Track %d/%d", i+1, len(selected)), t.Name))
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
						result.SetText(setLogMessage(LogConvert, "PGS to SRT Conversion", "Starting PGS extraction process..."))
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
					var cmd *exec.Cmd
					if mkvextractBinaryPath != "" {
						// Use the stored full path to mkvextract
						cmd = exec.Command(mkvextractBinaryPath, "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
						fmt.Println("[DEBUG] Using stored mkvextract path for PGS track extraction:", mkvextractBinaryPath)
					} else {
						// Fallback to PATH lookup
						cmd = exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
						fmt.Println("[DEBUG] No stored mkvextract path for PGS track extraction, using default PATH lookup")
					}
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

						// Try to use pgsrip first, fall back to pgs-to-srt if not available
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

						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempPgsFile)
						absOutputPath := filepath.Join(outDir, outFile)
						
						// Check if pgsrip is available and use it if possible
						pgsripAvailable := checkPgsrip()
						if pgsripAvailable {
							fyne.Do(func() {
								result.SetText(result.Text + "\n\n=== Using pgsrip for conversion ===\n")
								statusLabel.SetText("Starting pgsrip conversion...")
							})
							
							// Set up conversion settings - simplified
							convSettings := PgsConversionSettings{
								Verbose: true, // Enable verbose logging
							}
							
							// Call our pgsrip conversion function
							err = convertPgsWithPgsrip(absInputPath, absOutputPath, langCode, result, statusLabel, conversionProgress, convSettings)
							if err == nil {
								fyne.Do(func() {
									result.SetText(result.Text + "\n\n✅ PGS to SRT conversion with pgsrip completed successfully!")
									statusLabel.SetText("Conversion complete!")
									conversionProgress.SetValue(100)
								})
								return
							} else {
								fyne.Do(func() {
									result.SetText(result.Text + "\n⚠️ pgsrip conversion failed: " + err.Error() + "\nFalling back to pgs-to-srt...")
								})
								// Fall back to pgs-to-srt
							}
						}
						
						// Fall back to pgs-to-srt if pgsrip not available or failed
						// Use the configured PGS-to-SRT script with Deno
						pgsToSrtScript := pgsToSrtScriptPath
						
						// Define the path to the trained data file with the selected language
						trainedDataPath := filepath.Join(filepath.Dir(pgsToSrtScript), "tessdata_fast", langCode+".traineddata")
						
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
						var denoCmd string
						if denoBinaryPath != "" {
							denoCmd = fmt.Sprintf("%s run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"",
								denoBinaryPath, pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
							logger.Printf("Using stored Deno path: %s\n", denoBinaryPath)
						} else {
							denoCmd = fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"",
								pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
							logger.Printf("No stored Deno path, using default 'deno' command\n")
						}
						cmd = exec.Command("sh", "-c", denoCmd)

						// Set the working directory to ensure relative paths work correctly
						cmd.Dir = filepath.Dir(pgsToSrtScript)

						// Print the environment and command for debugging
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Environment ===\n")
							result.SetText(result.Text + fmt.Sprintf("Working directory: %s\n", cmd.Dir))
							result.SetText(result.Text + fmt.Sprintf("PATH: %s\n", os.Getenv("PATH")))
							result.SetText(result.Text + "\n=== Command ===\n")
							if denoBinaryPath != "" {
								result.SetText(result.Text + fmt.Sprintf("%s run --allow-read --allow-write %s %s %s > %s\n",
									denoBinaryPath, pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
							} else {
								result.SetText(result.Text + fmt.Sprintf("deno run --allow-read --allow-write %s %s %s > %s\n",
									pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
							}
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
						result.SetText(setLogMessage(LogConvert, "ASS/SSA to SRT Conversion", "Starting ASS/SSA to SRT conversion process..."))
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
					var cmd *exec.Cmd
					if mkvextractBinaryPath != "" {
						// Use the stored full path to mkvextract
						cmd = exec.Command(mkvextractBinaryPath, "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
						fmt.Println("[DEBUG] Using stored mkvextract path for ASS track extraction:", mkvextractBinaryPath)
					} else {
						// Fallback to PATH lookup
						cmd = exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
						fmt.Println("[DEBUG] No stored mkvextract path for ASS track extraction, using default PATH lookup")
					}
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
						result.SetText(setLogMessage(LogConvert, "VobSub to SRT Conversion", "Starting VobSub to SRT conversion process..."))
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
					var cmd *exec.Cmd
					if mkvextractBinaryPath != "" {
						// Use the stored full path to mkvextract
						cmd = exec.Command(mkvextractBinaryPath, "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
						fmt.Println("[DEBUG] Using stored mkvextract path for VobSub track extraction:", mkvextractBinaryPath)
					} else {
						// Fallback to PATH lookup
						cmd = exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
						fmt.Println("[DEBUG] No stored mkvextract path for VobSub track extraction, using default PATH lookup")
					}
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

					// Determine extension based on codec with comprehensive matching
					codecLower := strings.ToLower(t.Codec)
					if strings.Contains(codecLower, "subrip") || strings.Contains(codecLower, "srt") {
						fileExt = "srt"
						fyne.Do(func() {
							result.SetText(result.Text + "\nDetected SRT format, using .srt extension")
						})
					} else if strings.Contains(codecLower, "pgs") || strings.Contains(codecLower, "hdmv") {
						fileExt = "sup"
					} else if strings.Contains(codecLower, "ass") || strings.Contains(codecLower, "substation") || strings.Contains(codecLower, "advanced substation") {
						fileExt = "ass"
					} else if strings.Contains(codecLower, "ssa") {
						fileExt = "ssa"
					} else if strings.Contains(codecLower, "vobsub") {
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
					var cmd *exec.Cmd
					if mkvextractBinaryPath != "" {
						// Use the stored full path to mkvextract
						cmd = exec.Command(mkvextractBinaryPath, "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))
						fmt.Println("[DEBUG] Using stored mkvextract path for generic track extraction:", mkvextractBinaryPath)
					} else {
						// Fallback to PATH lookup
						cmd = exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))
						fmt.Println("[DEBUG] No stored mkvextract path for generic track extraction, using default PATH lookup")
					}

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
						t.Status.SetText(setLogMessage(LogError, fmt.Sprintf("Error Extracting Track %s", t.Name), err.Error()))
						if t.ConvertOCR != nil && t.ConvertOCR.Checked {
							result.SetText(result.Text + setLogMessage(LogError, "Conversion Failed", err.Error()))
						} else {
							result.SetText(result.Text + setLogMessage(LogError, "Extraction Failed", err.Error()))
						}
					} else {
						t.State = "Done"
						if t.ConvertOCR != nil && t.ConvertOCR.Checked {
							t.Status.SetText("Converted")
							result.SetText(result.Text + setLogMessage(LogSuccess, "Conversion Complete", fmt.Sprintf("Successfully converted %s to SRT.", t.Name)))
						} else {
							t.Status.SetText("Extracted")
							result.SetText(result.Text + setLogMessage(LogSuccess, "Track Extracted", t.Name))
						}
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
					result.SetText(setLogMessage(LogSuccess, "Extraction Complete", "All selected tracks have been processed."))
					progress.SetValue(progress.Max)
				} else {
					result.SetText(fmt.Sprintf("Extraction stopped after %d of %d tracks", tracksDone, len(selected)))
				}
			})
		}()
	})

	// Create Support button with improved UX
	supportBtn := widget.NewButton("Donate ", func() {
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
	titleLabel := widget.NewLabel(fmt.Sprintf("Subtitle Forge %s", AppVersion))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create file selection button row
	fileButtonRow := container.NewHBox(
		fileBtn,
		batchBtn,
		clearBtn,
	)

	topContent := container.NewVBox(
		titleLabel,
		fileButtonRow,
		selectedFile,
		fileListContainer,
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
	insertSubtitleFileLabel := widget.NewLabel("No subtitle file selected")

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

	selectInsertSubtitleBtn := widget.NewButton("Select Subtitle File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			filePath := reader.URI().Path()
			// Check if it's a supported subtitle format
			supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt"}
			fileExt := strings.ToLower(filepath.Ext(filePath))
			isSupported := false
			for _, ext := range supportedExts {
				if fileExt == ext {
					isSupported = true
					break
				}
			}
			
			if !isSupported {
				dialog.ShowInformation("Invalid File", "Please select a subtitle file (.srt, .ass, .ssa, .vtt, .sub, .sup, .txt)", w)
				return
			}

			insertSubtitleFileLabel.SetText(filePath)
	}, w)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt", ".smi", ".mpl", ".tmp"}))
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
		subtitlePath := insertSubtitleFileLabel.Text

		if mkvPath == "No MKV file selected" || subtitlePath == "No subtitle file selected" {
			dialog.ShowInformation("Missing Files", "Please select both MKV and subtitle files", w)
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

		// Add subtitle file at the end
		mkvmergeArgs = append(mkvmergeArgs, subtitlePath)

		// Run mkvmerge command to add subtitle
		go func() {
			var cmd *exec.Cmd
			if mkvmergeBinaryPath != "" {
				// Use the stored full path to mkvmerge
				cmd = exec.Command(mkvmergeBinaryPath, mkvmergeArgs...)
				fmt.Println("[DEBUG] Using stored mkvmerge path for subtitle addition:", mkvmergeBinaryPath)
			} else {
				// Fallback to PATH lookup
				cmd = exec.Command("mkvmerge", mkvmergeArgs...)
				fmt.Println("[DEBUG] No stored mkvmerge path for subtitle addition, using default PATH lookup")
			}

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

	subtitleDropArea := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 100})
	subtitleDropLabel := widget.NewLabelWithStyle("Drop Subtitle File Here", fyne.TextAlignCenter, fyne.TextStyle{})
	subtitleDropContainer := container.NewStack(
		subtitleDropArea,
		subtitleDropLabel,
	)
	subtitleDropContainer.Resize(fyne.NewSize(300, 60))

	// Group file selection
	fileSelectionGroup := widget.NewCard("File Selection", "", container.NewVBox(
		container.NewHBox(selectInsertMkvBtn, insertMkvFileLabel),
		mkvDropContainer,
		container.NewHBox(selectInsertSubtitleBtn, insertSubtitleFileLabel),
		subtitleDropContainer,
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
		container.NewPadded(languageSection),       // Using our new language section with title and separator
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
		&widget.TextSegment{Text: "Subtitle Forge " + AppVersion + "\n", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
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

	// Create convert subtitles tab content
	convertTabContent, loadConvertFile, loadConvertFiles := createConvertSubtitlesTab(w)

	// Wrap each tab content in a scroll container to ensure proper resizability
	extractScroll := container.NewScroll(extractTabContent)
	insertScroll := container.NewScroll(insertTabContent)
	convertScroll := container.NewScroll(convertTabContent)
	settingsScroll := container.NewScroll(settingsTabContent)

	// Create tabs with scrollable content
	tabs := container.NewAppTabs(
		container.NewTabItem("Extract Subtitles", extractScroll),
		container.NewTabItem("Insert Subtitles", insertScroll),
		container.NewTabItem("Convert Subtitles", convertScroll),
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
					} else {
						// Check if it's a supported subtitle format
						supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".sup", ".txt"}
						isSupported := false
						for _, ext := range supportedExts {
							if fileExt == ext {
								isSupported = true
								break
							}
						}
						
						if isSupported {
							// Handle subtitle file drop
							insertSubtitleFileLabel.SetText(filePath)
							subtitleDropLabel.SetText(filepath.Base(filePath))
							subtitleDropArea.FillColor = color.NRGBA{R: 100, G: 200, B: 100, A: 100}
							subtitleDropArea.Refresh()
							a.SendNotification(&fyne.Notification{
								Title:   "File Dropped",
								Content: "Subtitle file loaded: " + filepath.Base(filePath),
							})
						} else {
							a.SendNotification(&fyne.Notification{
								Title:   "Invalid File",
								Content: "Please drop an MKV or subtitle file (.srt, .ass, .ssa, .vtt, .sub, .sup, .txt).",
							})
						}
					}
				}
			})
		} else if tab.Text == "Convert Subtitles" {
			// Enhanced drag and drop for Convert Subtitles tab (supports batch processing)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}
				
				// Check if it's a single file or multiple files
				if len(uris) == 1 {
					// Single file - use existing single file logic
					filePath := uris[0].Path()
					fileExt := strings.ToLower(filepath.Ext(filePath))
					
					// Check if it's a supported subtitle format
					supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".idx", ".sup", ".txt"}
					isSupported := false
					for _, ext := range supportedExts {
						if fileExt == ext {
							isSupported = true
							break
						}
					}
					
					if isSupported {
						// Load the file into the Convert tab
						loadConvertFile(filePath)
						a.SendNotification(&fyne.Notification{
							Title:   "Subtitle File Loaded",
							Content: "Loaded " + strings.ToUpper(fileExt[1:]) + " file: " + filepath.Base(filePath),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "Invalid File",
							Content: "Please drop a subtitle file (.srt, .ass, .vtt, .sub, etc.)",
						})
					}
				} else {
					// Multiple files - use batch mode
					var subtitleFiles []string
					supportedExts := []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".idx", ".sup", ".txt"}
					
					for _, uri := range uris {
						filePath := uri.Path()
						fileExt := strings.ToLower(filepath.Ext(filePath))
						for _, ext := range supportedExts {
							if fileExt == ext {
								subtitleFiles = append(subtitleFiles, filePath)
								break
							}
						}
					}
					
					if len(subtitleFiles) > 0 {
						// Load multiple files for batch conversion
						loadConvertFiles(subtitleFiles)
						a.SendNotification(&fyne.Notification{
							Title:   "Batch Mode Enabled",
							Content: fmt.Sprintf("Loaded %d subtitle files for batch conversion", len(subtitleFiles)),
						})
					} else {
						a.SendNotification(&fyne.Notification{
							Title:   "No Valid Files",
							Content: "Please drop subtitle files (.srt, .ass, .vtt, .sub, etc.)",
						})
					}
				}
			})
		} else if tab.Text == "Extract Subtitles" {
			// Enhanced drag and drop for Extract Subtitles tab (supports batch processing)
			w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
				if len(uris) == 0 {
					return
				}

				// Filter for MKV files only
				var mkvUris []fyne.URI
				for _, uri := range uris {
					filePath := uri.Path()
					fileExt := strings.ToLower(filepath.Ext(filePath))
					if fileExt == ".mkv" {
						mkvUris = append(mkvUris, uri)
					}
				}

				if len(mkvUris) == 0 {
					a.SendNotification(&fyne.Notification{
						Title:   "Invalid Files",
						Content: "Please drop MKV files only.",
					})
					return
				}

				if len(mkvUris) == 1 {
					// Single file mode
					filePath := mkvUris[0].Path()
					batchMode = false
					mkvFiles = []string{}
					fileList.Refresh()
					fileListContainer.Hide()

					mkvPath = filePath
					a.SendNotification(&fyne.Notification{
						Title:   "Single MKV File Dropped",
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

					result.SetText(setLogMessage(LogInfo, "MKV File Loaded", "MKV file dropped and loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file."))
				} else {
					// Multiple files - batch mode
					batchMode = true
					mkvFiles = []string{}
					for _, uri := range mkvUris {
						mkvFiles = append(mkvFiles, uri.Path())
					}
					fileList.Refresh()
					fileListContainer.Show()

					a.SendNotification(&fyne.Notification{
						Title:   "Multiple MKV Files Dropped",
						Content: fmt.Sprintf("%d MKV files loaded for batch processing", len(mkvFiles)),
					})

					// Set output directory to the directory of the first file
					if len(mkvFiles) > 0 {
						outDir = filepath.Dir(mkvFiles[0])
						selectedDir.SetText(outDir)
					}

					// Clear previous tracks
					trackItems = []*TrackItem{}
					trackList.Objects = nil
					trackList.Refresh()

					selectedFile.SetText(fmt.Sprintf("%d MKV files selected for batch processing", len(mkvFiles)))
					result.SetText(setLogMessage(LogInfo, "Batch Mode Enabled", fmt.Sprintf("Dropped %d MKV files. Click 'Load Tracks' to analyze all files and select which tracks to extract.", len(mkvFiles))))
				}
			})
		}
	}
	w.SetContent(tabs)

	// Trigger the OnChanged handler for the initial tab
	tabs.OnChanged(tabs.Selected())

	w.ShowAndRun()
}
