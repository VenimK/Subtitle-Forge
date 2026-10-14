package main

import "os"

// Helper function to check if a file exists and is executable
func fileExistsAndExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if file is executable
	return info.Mode()&0111 != 0
}
