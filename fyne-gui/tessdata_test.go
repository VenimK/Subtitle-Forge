package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTessdataDetection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TESSDATA_PREFIX", "")
	if !isReadableFile("/opt/homebrew/share/tessdata/eng.traineddata") {
		t.Skip("Homebrew tessdata not installed")
	}
	if !isTessdataConfigured() {
		t.Fatal("installed Homebrew tessdata was not detected")
	}
}

func TestTessdataPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := t.TempDir()
	if err := os.WriteFile(filepath.Join(custom, "eng.traineddata"), []byte("model"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESSDATA_PREFIX", custom)
	if got := tessdataPath(); got != custom {
		t.Fatalf("tessdataPath() = %q, want %q", got, custom)
	}
	t.Setenv("TESSDATA_PREFIX", filepath.Join(home, "missing"))
	local := filepath.Join(home, "tessdata_best")
	if err := os.Mkdir(local, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "eng.traineddata"), []byte("model"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := tessdataPath(); got != local {
		t.Fatalf("tessdataPath() = %q, want %q", got, local)
	}
}
