package main

import (
	"bufio"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

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
			// Join text lines with proper spacing for GST compatibility
			text := strings.Join(textLines, "\n")
			// Add explicit spaces after commas to help GST
			text = regexp.MustCompile(`,(\S)`).ReplaceAllString(text, ", $1")
			// Add explicit spaces after periods to help GST
			text = regexp.MustCompile(`\.(\S)`).ReplaceAllString(text, ". $1")
			// Add explicit spaces after question marks and exclamation marks
			text = regexp.MustCompile(`\?(\S)`).ReplaceAllString(text, "? $1")
			text = regexp.MustCompile(`!(\S)`).ReplaceAllString(text, "! $1")

			entries = append(entries, SubtitleEntry{
				Index:     index,
				StartTime: startTime,
				EndTime:   endTime,
				Text:      text,
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

// convertASSTagsToSRT converts ASS formatting tags to SRT-compatible HTML tags
func convertASSTagsToSRT(text string) string {
	// Convert line breaks
	text = strings.ReplaceAll(text, "\\N", "\n")
	text = strings.ReplaceAll(text, "\\n", "\n")

	// Convert italic tags
	text = strings.ReplaceAll(text, "{\\i1}", "<i>")
	text = strings.ReplaceAll(text, "{\\i0}", "</i>")

	// Convert bold tags
	text = strings.ReplaceAll(text, "{\\b1}", "<b>")
	text = strings.ReplaceAll(text, "{\\b0}", "</b>")

	// Convert underline tags
	text = strings.ReplaceAll(text, "{\\u1}", "<u>")
	text = strings.ReplaceAll(text, "{\\u0}", "</u>")

	// Remove other ASS tags (position, alignment, color, etc.)
	re := regexp.MustCompile(`\{[^}]*\}`)
	text = re.ReplaceAllString(text, "")

	return strings.TrimSpace(text)
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

				// Convert ASS tags to SRT-compatible HTML tags
				text = convertASSTagsToSRT(text)

				// Ensure proper spacing after commas and punctuation for better GST translation
				text = regexp.MustCompile(`,(\S)`).ReplaceAllString(text, ", $1")
				text = regexp.MustCompile(`\.(\S)`).ReplaceAllString(text, ". $1")

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

	AppLog("CONVERT", "Starting conversion: %s (%s) -> %s", filepath.Base(inputPath), inputFormat, outputFormat)

	// Determine output path
	var outputPath string
	if outputDir != "" {
		fileName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outputPath = filepath.Join(outputDir, fileName+"."+outputFormat)
	} else {
		outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "." + outputFormat
	}

	AppLog("CONVERT", "Output path: %s", outputPath)

	fyne.Do(func() {
		progress.SetValue(0.1)
		result.SetText(fmt.Sprintf("🔄 Converting %s to %s...", inputFormat, strings.ToUpper(outputFormat)))
	})

	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		AppLog("ERROR", "Convert: Failed to read input file %s: %v", filepath.Base(inputPath), err)
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

		AppLog("CONVERT", "PGS OCR conversion using %s", conversionMethod)
		AppLogCmd(cmd, nil, nil)
		err := cmd.Run()
		if err != nil {
			AppLog("ERROR", "Convert: PGS OCR failed with %s: %v", conversionMethod, err)
			fyne.Do(func() {
				result.SetText(fmt.Sprintf("❌ Error converting PGS with %s: %v\n\nTry using the Extract Subtitles tab for PGS conversion, or check the Utilities tab for installation help.", conversionMethod, err))
			})
			return false
		}
		AppLog("SUCCESS", "PGS OCR conversion completed")

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
	convertTitle := widget.NewLabel(T("convert.title"))
	convertTitle.TextStyle = fyne.TextStyle{Bold: true}
	convertTitle.Alignment = fyne.TextAlignCenter

	// Input file selection - support both single and batch
	var inputFile string
	var inputFormat string
	var convertFiles []string
	var convertBatchMode bool

	inputLabel := widget.NewLabel(T("convert.no_file"))
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
	inputBtn := widget.NewButton(T("convert.select_file"), func() {
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
	batchBtn := widget.NewButton(T("ai.select_batch"), func() {
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

			filepath.Walk(uri.Path(), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && IsSubtitleFile(path) {
					convertFiles = append(convertFiles, path)
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
	clearBtn := widget.NewButton(T("extract.clear_files"), func() {
		convertBatchMode = false
		convertFiles = nil
		inputFile = ""
		inputFormat = ""
		convertFileList.Objects = nil
		convertFileList.Refresh()
		inputLabel.SetText(T("convert.no_file"))
	})
	clearBtn.Importance = widget.LowImportance

	// Format selection
	outputFormats := []string{"SRT", "ASS", "SSA", "VTT", "SUB", "TXT"}
	outputFormatSelect := widget.NewSelect(outputFormats, nil)
	outputFormatSelect.SetSelected("SRT") // Default to SRT

	// Output directory selection
	var outputDir string
	outputLabel := widget.NewLabel(T("convert.output_dir"))

	outputBtn := widget.NewButton(T("convert.select_output"), func() {
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
	convertBtn := widget.NewButton(T("convert.start"), func() {
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
				PreserveTiming:   preserveTimingCheck.Checked,
				PreserveStyle:    preserveStyleCheck.Checked,
				Encoding:         encodingSelect.Selected,
				FrameRate:        parseFloat(frameRateSelect.Selected, 25.0),
				TimeOffset:       parseFloat(timeOffsetEntry.Text, 0.0),
				RemoveFormatting: removeFormattingCheck.Checked,
				TextCase:         caseSelect.Selected,
				FontFamily:       fontFamilySelect.Selected,
				FontSize:         parseInt(fontSizeEntry.Text, 20),
				FontColor:        selectedFontColor,
				MarginLeft:       parseInt(marginLeftEntry.Text, 10),
				MarginRight:      parseInt(marginRightEntry.Text, 10),
				MarginVertical:   parseInt(marginVerticalEntry.Text, 10),
				StyleTemplate:    styleTemplateSelect.Selected,
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

				AppLog("CONVERT", "Batch conversion completed: %d/%d files to %s", successCount, totalFiles, outputFormat)
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
						AppLog("SUCCESS", "Single file conversion completed: %s to %s", filepath.Base(inputFile), outputFormat)
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

		for _, filePath := range filePaths {
			if IsSubtitleFile(filePath) {
				convertFiles = append(convertFiles, filePath)
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
