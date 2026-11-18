package main

import (
	"strings"
	"testing"
)

func TestAdjustSRTTiming(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,500\nHello\n\n2\n01:59:59,999 --> 02:00:01,000\nWorld\n"

	got := adjustSRTTiming(content, 1.5)
	want := "1\n00:00:02,500 --> 00:00:04,000\nHello\n\n2\n02:00:01,499 --> 02:00:02,500\nWorld\n"
	if got != want {
		t.Errorf("forward shift\n got %q\nwant %q", got, want)
	}

	// A negative offset clamps at zero instead of wrapping.
	got = adjustSRTTiming(content, -5)
	if !strings.Contains(got, "00:00:00,000 --> 00:00:00,000") {
		t.Errorf("negative shift should clamp to zero, got %q", got)
	}

	// Text lines are untouched.
	if strings.Contains(adjustSRTTiming(content, 1), "Hello, 1") {
		t.Error("text lines must not change")
	}
}
