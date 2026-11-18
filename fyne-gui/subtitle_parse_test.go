package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// oldPunctuationSpacing is the original four-pass implementation, kept here so
// the compiled-regex version in convert_tab.go cannot drift from it.
func oldPunctuationSpacing(text string) string {
	text = regexp.MustCompile(`,(\S)`).ReplaceAllString(text, ", $1")
	text = regexp.MustCompile(`\.(\S)`).ReplaceAllString(text, ". $1")
	text = regexp.MustCompile(`\?(\S)`).ReplaceAllString(text, "? $1")
	text = regexp.MustCompile(`!(\S)`).ReplaceAllString(text, "! $1")
	return text
}

func TestPunctuationSpacingMatchesOriginal(t *testing.T) {
	samples := []string{
		"Hello,world", "Hello.World", "Really?Yes!", "!.", "!.",
		"..", "...!", "a,b.c?d!e", "no punctuation",
		"1,2.3?4!5", "  spaced  ,now  ", "\tabc.def",
		"subtitle with (parentheses), and commas,ok",
		"Unicode: café,voilà!", "emoji 😀,next",
		"multi\nline,here", "trailing,", "-dash-case",
		"a,.b", ",.,.", "?!", "",
	}
	for _, s := range samples {
		if got := addPunctuationSpaces(s); got != oldPunctuationSpacing(s) {
			t.Errorf("addPunctuationSpaces(%q) = %q, want %q", s, got, oldPunctuationSpacing(s))
		}
	}
}

func TestParseSRTAppliesPunctuationSpacing(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,500\nHello,world\nsecond line\n\n2\n00:00:03,000 --> 00:00:04,000\nBye.now\n\n"
	entries, err := parseSRT(content)
	if err != nil {
		t.Fatalf("parseSRT: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Text != "Hello, world\nsecond line" {
		t.Errorf("entry 1 text = %q", entries[0].Text)
	}
	if entries[0].StartTime != time.Second || entries[0].EndTime != 2500*time.Millisecond {
		t.Errorf("entry 1 times = %v-%v", entries[0].StartTime, entries[0].EndTime)
	}
	if entries[1].Text != "Bye. now" {
		t.Errorf("entry 2 text = %q", entries[1].Text)
	}
}

func TestParseASSStripsTagsAndSpacesPunctuation(t *testing.T) {
	content := "[Script Info]\n[Events]\nDialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,{\\b1}Hi,there{\\b0} {\\pos(0,0)}\n"
	entries, err := parseASS(content)
	if err != nil {
		t.Fatalf("parseASS: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Text != "<b>Hi, there</b>" {
		t.Errorf("text = %q", entries[0].Text)
	}
	if entries[0].StartTime != time.Second || entries[0].EndTime != 2*time.Second {
		t.Errorf("times = %v-%v", entries[0].StartTime, entries[0].EndTime)
	}
}

// The digit fast path must agree with the regex fallback, including on the
// shapes the fast path rejects (odd spacing, more or fewer hour digits).
func TestTimeParsersAgreeWithRegex(t *testing.T) {
	srt := []string{
		"00:01:02,003", "23:59:59,999", "00:00:00,000",
		"  00:01:02,003  ", "0:01:02,003", "00:01:02,0035", "00:01:02.003",
	}
	for _, s := range srt {
		want := srtTimeRe.FindStringSubmatch(strings.TrimSpace(s))
		got, err := parseSRTTime(strings.TrimSpace(s))
		if want == nil {
			if err == nil {
				t.Errorf("parseSRTTime(%q) should fail like the regex does", s)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseSRTTime(%q): %v", s, err)
		}
		wantDur := mustSRTDuration(t, want)
		if got != wantDur {
			t.Errorf("parseSRTTime(%q) = %v, want %v", s, got, wantDur)
		}
	}

	vtt := []string{
		"00:01:02.003", "00:00:00.000", "01:02.003", "59:59.999",
		"1:02:03.004", "0:01:02.003", "00:01:02,003",
	}
	for _, s := range vtt {
		want := vttTimeRe.FindStringSubmatch(s)
		got, err := parseVTTTime(s)
		if want == nil {
			if err == nil {
				t.Errorf("parseVTTTime(%q) should fail like the regex does", s)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseVTTTime(%q): %v", s, err)
		}
		h := 0
		if want[1] != "" {
			h, _ = strconv.Atoi(want[1])
		}
		m, _ := strconv.Atoi(want[2])
		s2, _ := strconv.Atoi(want[3])
		ms, _ := strconv.Atoi(want[4])
		wantDur := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
			time.Duration(s2)*time.Second + time.Duration(ms)*time.Millisecond
		if got != wantDur {
			t.Errorf("parseVTTTime(%q) = %v, want %v", s, got, wantDur)
		}
	}
}

func mustSRTDuration(t *testing.T, matches []string) time.Duration {
	t.Helper()
	h, _ := strconv.Atoi(matches[1])
	m, _ := strconv.Atoi(matches[2])
	s, _ := strconv.Atoi(matches[3])
	ms, _ := strconv.Atoi(matches[4])
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond
}

func TestParseVTTTimes(t *testing.T) {
	content := "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nHello,world\n\n0:00:05.500 --> 0:00:06.000\nShort form\n\n"
	entries, err := parseVTT(content)
	if err != nil {
		t.Fatalf("parseVTT: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[1].StartTime != 5500*time.Millisecond {
		t.Errorf("entry 2 start = %v", entries[1].StartTime)
	}
}

// The live progress line is replaced in place instead of appended per tick, so
// the log keeps one progress line at the bottom.
func TestAppendProgressKeepsSingleBottomLine(t *testing.T) {
	oldLines, oldIdx, oldArea := aiResultsLines, aiProgressLineIdx, resultsArea
	defer func() {
		aiResultsLines, aiProgressLineIdx, resultsArea = oldLines, oldIdx, oldArea
	}()

	resultsArea = nil // no widget, so the coalesced redraw is not armed
	aiResultsLines = []string{"Processing: a.srt"}
	aiProgressLineIdx = -1

	appendProgressToResults("Translating: 10%")
	appendProgressToResults("Translating: 20%")
	appendToResults("done")

	if len(aiResultsLines) != 3 {
		t.Fatalf("lines = %q, want 3 entries", aiResultsLines)
	}
	if aiResultsLines[1] != "Translating: 20%" {
		t.Errorf("progress line = %q, want the latest tick", aiResultsLines[1])
	}
	if aiProgressLineIdx != 1 {
		t.Errorf("aiProgressLineIdx = %d, want 1", aiProgressLineIdx)
	}
}
