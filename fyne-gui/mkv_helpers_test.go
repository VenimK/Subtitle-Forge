package main

import (
	"testing"
)

func TestCodecToExtension(t *testing.T) {
	tests := []struct {
		name       string
		codec      string
		convertOCR bool
		want       string
	}{
		{"SubRip", "S_TEXT/UTF8", false, "srt"},
		{"SubRip explicit", "SubRip/SRT", false, "srt"},
		{"PGS no convert", "HDMV PGS", false, "sup"},
		{"PGS with convert", "HDMV PGS", true, "srt"},
		{"PGS lowercase", "hdmv_pgs_subtitle", false, "sup"},
		{"PGS lowercase convert", "hdmv_pgs_subtitle", true, "srt"},
		{"VobSub", "VobSub", false, "sub"},
		{"VobSub lowercase", "vobsub", false, "sub"},
		{"ASS", "S_TEXT/ASS", false, "ass"},
		{"SubStation Alpha", "SubStation Alpha", false, "ass"},
		{"Advanced SubStation", "Advanced SubStation Alpha", false, "ass"},
		{"SSA", "S_TEXT/SSA", false, "ssa"},
		{"Unknown defaults to srt", "UnknownCodec", false, "srt"},
		{"Empty defaults to srt", "", false, "srt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodecToExtension(tt.codec, tt.convertOCR)
			if got != tt.want {
				t.Errorf("CodecToExtension(%q, %v) = %q, want %q", tt.codec, tt.convertOCR, got, tt.want)
			}
		})
	}
}

func TestCodecToExtensionForExtract(t *testing.T) {
	tests := []struct {
		name  string
		codec string
		want  string
	}{
		{"SubRip", "SubRip/SRT", "srt"},
		{"PGS", "HDMV PGS", "sup"},
		{"ASS", "S_TEXT/ASS", "ass"},
		{"SSA", "S_TEXT/SSA", "ssa"},
		{"VobSub returns idx", "VobSub", "idx"},
		{"Unknown uses cleaned codec", "S_TEXT/UTF8", "s_text_utf8"},
		{"Slash in codec cleaned", "Some/Weird/Codec", "some_weird_codec"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodecToExtensionForExtract(tt.codec)
			if got != tt.want {
				t.Errorf("CodecToExtensionForExtract(%q) = %q, want %q", tt.codec, got, tt.want)
			}
		})
	}
}

func TestIsSubtitleFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/path/to/file.srt", true},
		{"/path/to/file.SRT", true},
		{"/path/to/file.ass", true},
		{"/path/to/file.ssa", true},
		{"/path/to/file.vtt", true},
		{"/path/to/file.sub", true},
		{"/path/to/file.sup", true},
		{"/path/to/file.txt", true},
		{"/path/to/file.mkv", false},
		{"/path/to/file.mp4", false},
		{"/path/to/file.idx", false}, // idx not in basic list
		{"/path/to/file", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsSubtitleFile(tt.path)
			if got != tt.want {
				t.Errorf("IsSubtitleFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsSubtitleFileWithIdx(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/path/to/file.idx", true},
		{"/path/to/file.srt", true},
		{"/path/to/file.mkv", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsSubtitleFileWithIdx(tt.path)
			if got != tt.want {
				t.Errorf("IsSubtitleFileWithIdx(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsConvertibleCodec(t *testing.T) {
	tests := []struct {
		codec string
		want  bool
	}{
		{"hdmv_pgs_subtitle", true},
		{"HDMV PGS", true},
		{"S_TEXT/ASS", true},
		{"S_TEXT/SSA", true},
		{"SubStation Alpha", true},
		{"Sub Station Alpha", true},
		{"vobsub", true},
		{"VobSub", true},
		{"SubRip/SRT", false},
		{"S_TEXT/UTF8", false},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			got := IsConvertibleCodec(tt.codec)
			if got != tt.want {
				t.Errorf("IsConvertibleCodec(%q) = %v, want %v", tt.codec, got, tt.want)
			}
		})
	}
}

func TestIsPGSCodec(t *testing.T) {
	tests := []struct {
		codec string
		want  bool
	}{
		{"HDMV PGS", true},
		{"hdmv_pgs_subtitle", true},
		{"pgs", true},
		{"SubRip/SRT", false},
		{"VobSub", false},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			got := IsPGSCodec(tt.codec)
			if got != tt.want {
				t.Errorf("IsPGSCodec(%q) = %v, want %v", tt.codec, got, tt.want)
			}
		})
	}
}

func TestIsASSCodec(t *testing.T) {
	tests := []struct {
		codec string
		want  bool
	}{
		{"S_TEXT/ASS", true},
		{"S_TEXT/SSA", true},
		{"SubStation Alpha", true},
		{"Sub Station Alpha", true},
		{"SubRip/SRT", false},
		{"HDMV PGS", false},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			got := IsASSCodec(tt.codec)
			if got != tt.want {
				t.Errorf("IsASSCodec(%q) = %v, want %v", tt.codec, got, tt.want)
			}
		})
	}
}

func TestIsVobSubCodec(t *testing.T) {
	tests := []struct {
		codec string
		want  bool
	}{
		{"VobSub", true},
		{"vobsub", true},
		{"HDMV PGS", false},
		{"SubRip/SRT", false},
	}

	for _, tt := range tests {
		t.Run(tt.codec, func(t *testing.T) {
			got := IsVobSubCodec(tt.codec)
			if got != tt.want {
				t.Errorf("IsVobSubCodec(%q) = %v, want %v", tt.codec, got, tt.want)
			}
		})
	}
}

func TestNewMkvextractCmd(t *testing.T) {
	// Reset global state for test
	origPath := mkvextractBinaryPath
	defer func() { mkvextractBinaryPath = origPath }()

	// Test with empty path (should use "mkvextract")
	mkvextractBinaryPath = ""
	cmd := NewMkvextractCmd("tracks", "test.mkv", "0:out.srt")
	if cmd.Path == "" {
		t.Error("NewMkvextractCmd returned cmd with empty Path")
	}
	if len(cmd.Args) != 4 {
		t.Errorf("NewMkvextractCmd args count = %d, want 4", len(cmd.Args))
	}

	// Test with custom path
	mkvextractBinaryPath = "/usr/local/bin/mkvextract"
	cmd = NewMkvextractCmd("tracks", "test.mkv", "0:out.srt")
	if cmd.Path != "/usr/local/bin/mkvextract" {
		t.Errorf("NewMkvextractCmd Path = %q, want /usr/local/bin/mkvextract", cmd.Path)
	}
}

func TestNewMkvmergeCmd(t *testing.T) {
	// Reset global state for test
	origPath := mkvmergeBinaryPath
	defer func() { mkvmergeBinaryPath = origPath }()

	// Test with empty path (should use "mkvmerge")
	mkvmergeBinaryPath = ""
	cmd := NewMkvmergeCmd("-J", "test.mkv")
	if cmd.Path == "" {
		t.Error("NewMkvmergeCmd returned cmd with empty Path")
	}
	if len(cmd.Args) != 3 {
		t.Errorf("NewMkvmergeCmd args count = %d, want 3", len(cmd.Args))
	}

	// Test with custom path
	mkvmergeBinaryPath = "/opt/homebrew/bin/mkvmerge"
	cmd = NewMkvmergeCmd("-J", "test.mkv")
	if cmd.Path != "/opt/homebrew/bin/mkvmerge" {
		t.Errorf("NewMkvmergeCmd Path = %q, want /opt/homebrew/bin/mkvmerge", cmd.Path)
	}
}
