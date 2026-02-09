package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Supported video container extensions
var videoExtensions = []string{".mkv", ".mp4", ".m4v"}

// IsVideoFile returns true if the file has a supported video container extension.
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range videoExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// IsMKVFile returns true if the file is an MKV container.
func IsMKVFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".mkv"
}

// IsMP4File returns true if the file is an MP4/M4V container.
func IsMP4File(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4" || ext == ".m4v"
}

// VideoFileExtensions returns the list of supported video file extensions (with dots).
func VideoFileExtensions() []string {
	return videoExtensions
}

// ---------- ffprobe / ffmpeg helpers ----------

// findFFmpegPath returns the best available path for ffmpeg.
func findFFmpegPath() string {
	// Check PATH first
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	// Homebrew (Apple Silicon)
	if _, err := os.Stat("/opt/homebrew/bin/ffmpeg"); err == nil {
		return "/opt/homebrew/bin/ffmpeg"
	}
	// Homebrew (Intel)
	if _, err := os.Stat("/usr/local/bin/ffmpeg"); err == nil {
		return "/usr/local/bin/ffmpeg"
	}
	// Miniconda
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "miniconda3", "bin", "ffmpeg")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "ffmpeg" // fallback
}

// findFFprobePath returns the best available path for ffprobe.
func findFFprobePath() string {
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	if _, err := os.Stat("/opt/homebrew/bin/ffprobe"); err == nil {
		return "/opt/homebrew/bin/ffprobe"
	}
	if _, err := os.Stat("/usr/local/bin/ffprobe"); err == nil {
		return "/usr/local/bin/ffprobe"
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "miniconda3", "bin", "ffprobe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "ffprobe" // fallback
}

// NewFFmpegCmd creates an exec.Cmd for ffmpeg.
func NewFFmpegCmd(args ...string) *exec.Cmd {
	return exec.Command(findFFmpegPath(), args...)
}

// NewFFprobeCmd creates an exec.Cmd for ffprobe.
func NewFFprobeCmd(args ...string) *exec.Cmd {
	return exec.Command(findFFprobePath(), args...)
}

// ---------- ffprobe track info ----------

// FFprobeStream represents a single stream from ffprobe JSON output.
type FFprobeStream struct {
	Index     int               `json:"index"`
	CodecName string            `json:"codec_name"`
	CodecType string            `json:"codec_type"`
	Tags      map[string]string `json:"tags"`
}

// FFprobeResult represents the top-level ffprobe JSON output.
type FFprobeResult struct {
	Streams []FFprobeStream `json:"streams"`
}

// MP4SubtitleTrack holds parsed subtitle track info from an MP4 file.
type MP4SubtitleTrack struct {
	Index     int    // Stream index in the file
	SubIndex  int    // Subtitle-only index (0-based among subtitle streams)
	Codec     string // e.g. "mov_text", "subrip", "ass"
	Language  string // ISO 639 language tag
	TrackName string // Title tag if present
}

// LoadMP4SubtitleTracks uses ffprobe to list subtitle tracks in an MP4/M4V file.
func LoadMP4SubtitleTracks(videoPath string) ([]MP4SubtitleTrack, error) {
	cmd := NewFFprobeCmd(
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result FFprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var tracks []MP4SubtitleTrack
	subIdx := 0
	for _, s := range result.Streams {
		if s.CodecType != "subtitle" {
			continue
		}

		lang := s.Tags["language"]
		if lang == "" {
			lang = "und"
		}
		name := s.Tags["title"]
		if name == "" {
			name = s.Tags["handler_name"]
		}
		if name == "" || name == "SubtitleHandler" {
			name = "Untitled"
		}

		tracks = append(tracks, MP4SubtitleTrack{
			Index:     s.Index,
			SubIndex:  subIdx,
			Codec:     s.CodecName,
			Language:  lang,
			TrackName: name,
		})
		subIdx++
	}

	return tracks, nil
}

// MP4CodecDisplayName returns a human-readable name for an ffprobe subtitle codec.
func MP4CodecDisplayName(codec string) string {
	switch strings.ToLower(codec) {
	case "mov_text", "tx3g":
		return "SRT (mov_text)"
	case "subrip", "srt":
		return "SubRip (SRT)"
	case "ass":
		return "ASS"
	case "ssa":
		return "SSA"
	case "webvtt":
		return "WebVTT"
	case "hdmv_pgs_subtitle":
		return "HDMV PGS"
	case "dvd_subtitle", "dvdsub":
		return "VobSub"
	default:
		return codec
	}
}

// ---------- ffmpeg extraction ----------

// MP4CodecToExtension maps an ffprobe codec name to a subtitle file extension.
func MP4CodecToExtension(codec string) string {
	switch strings.ToLower(codec) {
	case "subrip", "srt":
		return "srt"
	case "ass", "ssa":
		return "ass"
	case "mov_text", "tx3g":
		return "srt" // mov_text is best extracted as SRT
	case "webvtt":
		return "vtt"
	case "hdmv_pgs_subtitle":
		return "sup"
	case "dvd_subtitle", "dvdsub":
		return "sub"
	default:
		return "srt"
	}
}

// ExtractMP4Subtitle extracts a single subtitle track from an MP4/M4V file using ffmpeg.
// streamIndex is the absolute stream index, outPath is the full output file path.
func ExtractMP4Subtitle(videoPath string, streamIndex int, outPath string) ([]byte, error) {
	AppLog("EXTRACT", "ffmpeg extracting stream %d from %s -> %s", streamIndex, filepath.Base(videoPath), filepath.Base(outPath))

	// Determine the best extraction strategy based on output extension.
	// Text-based subtitle formats (srt, ass, vtt) need explicit format specification
	// because ffmpeg can't always infer the muxer from the extension when copying mov_text.
	ext := strings.ToLower(filepath.Ext(outPath))

	var cmd *exec.Cmd
	switch ext {
	case ".srt":
		// Force SRT output format — handles mov_text/tx3g conversion
		cmd = NewFFmpegCmd(
			"-y",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", streamIndex),
			"-f", "srt",
			outPath,
		)
	case ".ass", ".ssa":
		cmd = NewFFmpegCmd(
			"-y",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", streamIndex),
			"-f", "ass",
			outPath,
		)
	case ".vtt":
		cmd = NewFFmpegCmd(
			"-y",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", streamIndex),
			"-f", "webvtt",
			outPath,
		)
	default:
		// Binary formats (sup, sub) — use codec copy
		cmd = NewFFmpegCmd(
			"-y",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", streamIndex),
			"-c:s", "copy",
			outPath,
		)
	}

	output, err := cmd.CombinedOutput()
	AppLogCmd(cmd, output, err)
	return output, err
}

// ---------- ffmpeg insertion ----------

// InsertMP4Subtitle inserts a subtitle file into an MP4 using ffmpeg.
// lang is the ISO 639 language code, trackName is the track title.
func InsertMP4Subtitle(videoPath, subtitlePath, outputPath, lang, trackName string, removeExisting bool) ([]byte, error) {
	args := []string{"-y", "-i", videoPath, "-i", subtitlePath}

	if removeExisting {
		// Copy video+audio, ignore existing subs, add new sub
		args = append(args, "-map", "0:v", "-map", "0:a?", "-map", "1:0")
	} else {
		// Copy everything from input, add new sub
		args = append(args, "-map", "0", "-map", "1:0")
	}

	args = append(args, "-c", "copy", "-c:s", "mov_text")

	// Set language metadata on the new subtitle stream
	if lang != "" {
		// The new subtitle stream is the last one
		args = append(args, "-metadata:s:s", fmt.Sprintf("language=%s", lang))
	}
	if trackName != "" {
		args = append(args, "-metadata:s:s", fmt.Sprintf("title=%s", trackName))
	}

	args = append(args, outputPath)

	cmd := NewFFmpegCmd(args...)
	AppLog("INSERT", "ffmpeg inserting subtitle into MP4: %s + %s -> %s", filepath.Base(videoPath), filepath.Base(subtitlePath), filepath.Base(outputPath))

	output, err := cmd.CombinedOutput()
	AppLogCmd(cmd, output, err)
	return output, err
}
