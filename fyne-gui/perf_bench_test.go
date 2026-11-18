package main

import (
	"fmt"
	"strings"
	"testing"
)

func benchSRT(entries int) string {
	var sb strings.Builder
	for i := 1; i <= entries; i++ {
		sb.WriteString(fmt.Sprintf("%d\n00:0%d:%02d,%03d --> 00:0%d:%02d,%03d\nLine one of subtitle %d\nand a second text line\n\n",
			i, i%60, i%60, i%1000, i%60, i%60, (i+1)%1000, i))
	}
	return sb.String()
}

func benchASS(entries int) string {
	var sb strings.Builder
	sb.WriteString("[Script Info]\n[Events]\n")
	for i := 1; i <= entries; i++ {
		sb.WriteString(fmt.Sprintf("Dialogue: 0,0:0%d:%02d.%02d,0:0%d:%02d.%02d,Default,,0,0,0,,{\\b1}subtitle %d,text{\\b0}\n",
			i%60, i%60, i%100, i%60, i%60, i%100, i))
	}
	return sb.String()
}

func benchVTT(entries int) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	for i := 1; i <= entries; i++ {
		sb.WriteString(fmt.Sprintf("00:0%d:%02d.%03d --> 00:0%d:%02d.%03d\nsubtitle %d,text\n\n",
			i%60, i%60, i%1000, i%60, i%60, (i+1)%1000, i))
	}
	return sb.String()
}

func BenchmarkParseSRT(b *testing.B) {
	content := benchSRT(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parseSRT(content); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSRTBlocks(b *testing.B) {
	content := benchSRT(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if n := len(parseSRTBlocks(content)); n != 2000 {
			b.Fatalf("got %d blocks", n)
		}
	}
}

func BenchmarkParseASS(b *testing.B) {
	content := benchASS(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parseASS(content); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseVTT(b *testing.B) {
	content := benchVTT(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parseVTT(content); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdjustSRTTiming(b *testing.B) {
	content := benchSRT(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adjustSRTTiming(content, 2.5)
	}
}

// BenchmarkGSTLine parses one line of GST output the way the stream reader does:
// strip the escapes, then look for the percentage. This runs many times a second.
func BenchmarkGSTLine(b *testing.B) {
	line := "\x1b[F\x1b[2KTranslating:  45%|████▌     | 900/2000 [00:10<00:12, 85.0it/s]\x1b[B"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleaned := stripANSI(strings.ReplaceAll(line, "\r", ""))
		if _, ok := parseGSTProgress(cleaned); !ok {
			b.Fatal("progress not parsed")
		}
	}
}

// BenchmarkCheckDependencies measures the Settings-tab sweep: nine tools, each
// one `--version`, now run concurrently so the total is the slowest tool.
func BenchmarkCheckDependencies(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if len(checkDependencies()) == 0 {
			b.Fatal("no dependency checks ran")
		}
	}
}
