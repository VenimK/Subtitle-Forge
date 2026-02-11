//go:build !darwin

package main

// GetWindowPosition returns (-1, -1) on non-macOS platforms (not supported).
func GetWindowPosition() (float64, float64) {
	return -1, -1
}

// SetWindowPosition is a no-op on non-macOS platforms.
func SetWindowPosition(x, y float64) {
	// Not supported on this platform
}
