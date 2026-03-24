package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Current application version - injected at build time
var AppVersion = "V2.4.3" // Default value for development

// ReleaseInfo stores information about a GitHub release
type ReleaseInfo struct {
	TagName    string `json:"tag_name"`
	HtmlUrl    string `json:"html_url"`
	TarballUrl string `json:"tarball_url"`
	ZipballUrl string `json:"zipball_url"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func normalizeVersionString(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimLeft(version, "vV")
	return version
}

func parseVersionParts(version string) []int {
	normalized := normalizeVersionString(version)
	if normalized == "" {
		return []int{0}
	}

	rawParts := strings.Split(normalized, ".")
	parts := make([]int, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			parts = append(parts, 0)
			continue
		}

		digits := strings.Builder{}
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			digits.WriteRune(r)
		}

		if digits.Len() == 0 {
			parts = append(parts, 0)
			continue
		}

		value, err := strconv.Atoi(digits.String())
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, value)
	}

	return parts
}

func isNewerVersion(latestVersion, currentVersion string) bool {
	latestParts := parseVersionParts(latestVersion)
	currentParts := parseVersionParts(currentVersion)

	maxLen := len(latestParts)
	if len(currentParts) > maxLen {
		maxLen = len(currentParts)
	}

	for i := 0; i < maxLen; i++ {
		latestPart := 0
		if i < len(latestParts) {
			latestPart = latestParts[i]
		}

		currentPart := 0
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}

		if latestPart > currentPart {
			return true
		}
		if latestPart < currentPart {
			return false
		}
	}

	return false
}

// openURL opens a URL in the default browser
func openURL(urlStr string) {
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Println("Error parsing URL:", err)
		return
	}
	fyne.CurrentApp().OpenURL(u)
}

// CheckForUpdates checks for new releases on GitHub and shows a notification
func CheckForUpdates(w fyne.Window) {
	go func() {
		// GitHub API URL for the latest release
		apiURL := "https://api.github.com/repos/venimk/gmmmkvsubsextract/releases/latest"

		// Create a new HTTP client with a timeout
		client := &http.Client{Timeout: 15 * time.Second}

		// Create a new HTTP request
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			log.Printf("Update check: failed to create request: %v", err)
			return
		}

		// Set headers
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Update check: failed to fetch release info: %v", err)
			return
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Update check: failed to read response body: %v", err)
			return
		}

		// Parse the JSON response
		var releaseInfo ReleaseInfo
		if err := json.Unmarshal(body, &releaseInfo); err != nil {
			log.Printf("Update check: failed to parse JSON: %v", err)
			return
		}

		// Compare normalized semantic-ish version parts instead of raw strings
		latestVersion := releaseInfo.TagName
		if isNewerVersion(latestVersion, AppVersion) {
			// Show notification on the main UI thread
			fyne.Do(func() {
				dialog.ShowConfirm(
					"Update Available",
					fmt.Sprintf("A new version (%s) is available!\nYou are using version %s.", latestVersion, AppVersion),
					func(b bool) {
						if b {
							downloadAndInstallUpdate(releaseInfo, w)
						}
					},
					w,
				)
			})
		}
	}()
}

// downloadAndInstallUpdate downloads and installs the update
func downloadAndInstallUpdate(releaseInfo ReleaseInfo, w fyne.Window) {
	// Find the appropriate asset for the current platform
	var downloadURL string
	var assetName string

	// Look for platform-specific assets
	switch runtime.GOOS {
	case "darwin": // macOS
		var expectedName string
		var archHint string
		if runtime.GOARCH == "arm64" {
			expectedName = "Subtitle-Forge-macOS-ARM64.zip"
			archHint = "arm64"
		} else {
			expectedName = "Subtitle-Forge-macOS-Intel.zip"
			archHint = "intel"
		}

		// Pass 1: exact match
		for _, asset := range releaseInfo.Assets {
			if asset.Name == expectedName {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				goto FOUND_ASSET
			}
		}

		// Pass 2: architecture hint in filename
		for _, asset := range releaseInfo.Assets {
			lower := strings.ToLower(asset.Name)
			if strings.Contains(lower, "macos") || strings.Contains(lower, "darwin") {
				if strings.Contains(lower, archHint) {
					downloadURL = asset.BrowserDownloadURL
					assetName = asset.Name
					goto FOUND_ASSET
				}
			}
		}

		// Pass 3: universal binary (backward compatibility)
		for _, asset := range releaseInfo.Assets {
			if asset.Name == "Subtitle-Forge-macOS.zip" {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				goto FOUND_ASSET
			}
		}

		// Pass 4: any macOS-related file
		for _, asset := range releaseInfo.Assets {
			lower := strings.ToLower(asset.Name)
			if strings.Contains(lower, "macos") || strings.Contains(lower, "darwin") {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				goto FOUND_ASSET
			}
		}
	case "windows":
		for _, asset := range releaseInfo.Assets {
			if strings.Contains(strings.ToLower(asset.Name), "windows") {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break
			}
		}
	case "linux":
		for _, asset := range releaseInfo.Assets {
			if strings.Contains(strings.ToLower(asset.Name), "linux") {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break
			}
		}
	}

	// If no appropriate asset was found, use the tarball or zipball URL
FOUND_ASSET:
	if downloadURL == "" {
		if runtime.GOOS == "windows" && releaseInfo.ZipballUrl != "" {
			downloadURL = releaseInfo.ZipballUrl
			assetName = "source-code.zip"
		} else if releaseInfo.TarballUrl != "" {
			downloadURL = releaseInfo.TarballUrl
			assetName = "source-code.tar.gz"
		} else {
			fyne.Do(func() {
				dialog.ShowInformation("Download Failed",
					"Couldn't find the appropriate download for your platform.\nOpening the releases page in your browser instead.", w)
			})
			openURL(releaseInfo.HtmlUrl)
			return
		}
	}

	// Create a progress dialog
	var progressDialog *dialog.CustomDialog
	var progress *widget.ProgressBar

	fyne.Do(func() {
		progress = widget.NewProgressBar()
		progress.Min = 0
		progress.Max = 100
		progress.SetValue(0)

		progressDialog = dialog.NewCustom("Downloading Update", "Cancel", progress, w)
		progressDialog.SetOnClosed(func() {
			// Handle cancel operation if needed
		})
		progressDialog.Show()
	})

	// Download in a goroutine
	go func() {
		// Create a temporary directory for the download
		tempDir, err := os.MkdirTemp("", "subtitle-forge-update")
		if err != nil {
			log.Printf("Failed to create temp directory: %v", err)
			showDownloadError(err, progressDialog, w)
			return
		}

		// Create the file path
		filePath := filepath.Join(tempDir, assetName)

		// Create the file
		outFile, err := os.Create(filePath)
		if err != nil {
			log.Printf("Failed to create file: %v", err)
			showDownloadError(err, progressDialog, w)
			return
		}
		defer outFile.Close()

		// Download the file
		resp, err := http.Get(downloadURL)
		if err != nil {
			log.Printf("Failed to download update: %v", err)
			showDownloadError(err, progressDialog, w)
			return
		}
		defer resp.Body.Close()

		// Check server response
		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("bad status: %s", resp.Status)
			log.Printf("Failed to download update: %v", err)
			showDownloadError(err, progressDialog, w)
			return
		}

		// Get the total size for progress reporting
		totalSize := resp.ContentLength

		// Create a buffer to store downloaded data
		buffer := make([]byte, 32*1024) // 32KB buffer
		var downloaded int64 = 0

		// Download the file and update progress
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				// Write to file
				_, writeErr := outFile.Write(buffer[:n])
				if writeErr != nil {
					log.Printf("Error writing to file: %v", writeErr)
					showDownloadError(writeErr, progressDialog, w)
					return
				}

				// Update progress
				downloaded += int64(n)
				if totalSize > 0 {
					progressValue := float64(downloaded) / float64(totalSize) * 100
					fyne.Do(func() {
						if progress != nil {
							progress.SetValue(progressValue)
						}
					})
				}
			}

			if err != nil {
				if err == io.EOF {
					break // Download complete
				}
				log.Printf("Error downloading file: %v", err)
				showDownloadError(err, progressDialog, w)
				return
			}
		}

		// Close the progress dialog
		fyne.Do(func() {
			if progressDialog != nil {
				progressDialog.Hide()
			}
		})

		// Install the update
		installUpdate(filePath, tempDir, w)
	}()
}

// showDownloadError displays an error message when download fails
func showDownloadError(err error, progressDialog *dialog.CustomDialog, w fyne.Window) {
	fyne.Do(func() {
		if progressDialog != nil {
			progressDialog.Hide()
		}
		dialog.ShowError(err, w)
	})
}

// installUpdate handles the installation of the downloaded update
func installUpdate(filePath string, tempDir string, w fyne.Window) {
	// Show installation instructions based on platform
	instructions := ""

	switch runtime.GOOS {
	case "darwin": // macOS
		instructions = "The update has been downloaded to:\n" + tempDir + "\n\n" +
			"To install the update:\n" +
			"1. Extract the downloaded archive\n" +
			"2. Replace the current application with the new version\n" +
			"3. Restart the application"
	case "windows":
		instructions = "The update has been downloaded to:\n" + tempDir + "\n\n" +
			"To install the update:\n" +
			"1. Extract the downloaded zip file\n" +
			"2. Replace the current executable with the new version\n" +
			"3. Restart the application"
	case "linux":
		instructions = "The update has been downloaded to:\n" + tempDir + "\n\n" +
			"To install the update:\n" +
			"1. Extract the downloaded archive\n" +
			"2. Replace the current binary with the new version\n" +
			"3. Restart the application"
	}

	fyne.Do(func() {
		dialog.ShowInformation("Update Downloaded", instructions, w)

		// Open the folder containing the downloaded file
		switch runtime.GOOS {
		case "darwin":
			exec.Command("open", tempDir).Start()
		case "windows":
			exec.Command("explorer", tempDir).Start()
		case "linux":
			exec.Command("xdg-open", tempDir).Start()
		}
	})
}
