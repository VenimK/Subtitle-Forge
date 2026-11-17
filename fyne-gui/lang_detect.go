package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

// Subtitle tracks are frequently muxed without a language tag, which shows up as
// "und" in mkvmerge/ffprobe. In that case the track text itself is the only
// language evidence available, so a short snippet is sampled and identified from
// function-word frequencies. No third-party dependency: the profiles below cover
// the languages the OCR dropdown offers plus the common neighbours of those.
const (
	langWindowSpan      = 75 * time.Second
	langSnippetMaxBytes = 64 << 10
	langSnippetTimeout  = 25 * time.Second
	langDurationTimeout = 10 * time.Second
	langDetectWorkers   = 4
	langMinHits         = 3
	langMinDistinct     = 2
	// langMarkerWeight is per occurrence and per distinct marker word; high enough
	// that "što" beats a pile of "da"/"je" shared across the whole Slavic group.
	langMarkerWeight = 5
)

// langWindows are the points across a track's timeline that get sampled, as a
// fraction of the runtime. The head of an episode is often a title card plus a
// handful of lines, which is not enough function words to identify a language, so
// three spread-out windows are concatenated instead.
var langWindows = []float64{0.10, 0.40, 0.70}

// langMarkers are words that belong to one profile only. Czech, Slovak,
// Slovenian and Croatian all use "je", "ne", "da", "to", so those function words
// cannot separate the languages on their own; exclusive ones ("što" vs "kar" vs
// "jsem" vs "som") can, which is why markers are weighted well above words.
var langMarkers = map[string][]string{
	"sv": {"att", "för", "ska", "något", "hade", "ville"},
	"da": {"lidt", "nok", "gjorde", "snart", "måske", "aldrig"},
	"no": {"noen", "noe", "veldig", "derfor"},
	"fi": {"minä", "hän", "että", "mitä", "vai", "sinä"},
	"et": {"pole", "minu", "kas", "väga", "kui", "üldse"},
	"pl": {"nic", "nikt", "chce", "będę", "może", "niego"},
	"cs": {"jsem", "ano", "protože", "ještě", "který", "proč", "velmi", "není"},
	"sk": {"som", "čo", "teraz", "sme", "sú", "tiež", "veľa", "tento"},
	"sl": {"kar", "nisem", "vse", "več", "zakaj", "tudi", "kjer"},
	"hr": {"što", "šta", "nisam", "kao", "ali", "nije", "ništa", "gdje", "nešto", "uvijek"},
	"is": {"að", "það", "já", "mér", "ekki"},
	"ru": {"это", "очень", "сейчас", "тебя", "может", "него"},
	"uk": {"ні", "щось", "таке", "мені"},
	"bg": {"ще", "като", "много", "нея"},
	"sr": {"што", "као", "али", "нијесам", "нећу", "сваки"},
	"mk": {"сум", "ќе", "ова", "што", "ние"},
	"mn": {"би", "нь", "дээ", "юм", "энэ", "зүйл", "гэж", "байна"},
	// Portuguese subtitles always carry accents, which the ASCII word list cannot
	// match, so the accented forms have to live here as markers.
	"pt": {"não", "você", "também", "estou", "então", "isso", "mais", "muito", "com", "fazer", "uma", "dos", "ela", "pode"},
	"es": {"muy", "pero", "porque", "tengo", "tiene", "usted", "esto", "ese", "esa", "voy"},
	"en": {"dont", "cant", "doesnt", "isnt", "wont", "youre", "thats", "theres", "would", "could"},
	// Indonesian and Malay share almost every function word, so only the words
	// each one keeps to itself ("tidak"/"bisa" vs "awak"/"boleh") separate them.
	"id": {"tidak", "bisa", "sangat", "pak", "ibu", "saja", "dengan", "untuk"},
	"ms": {"awak", "boleh", "tak", "macam", "kerana", "cuba", "sana", "pandai", "ramai"},
}

type langRule struct {
	code   string
	words  []string
	letter string // letters that are near-exclusive to this language
}

var latinProfiles = []langRule{
	{"en", []string{"the", "you", "what", "dont", "its", "was", "have", "were", "because", "cant"}, ""},
	{"nl", []string{"het", "van", "niet", "je", "wat", "met", "voor", "een", "ben", "weet", "ik", "maar", "ook", "nog"}, ""},
	{"de", []string{"und", "ich", "nicht", "ist", "der", "die", "das", "mich", "dich", "wir", "auch", "mein", "aber", "wie", "mir"}, "ß"},
	{"fr", []string{"je", "pas", "vous", "moi", "cest", "les", "des", "pour", "dans", "tres", "mais", "sur", "avec"}, ""},
	{"es", []string{"no", "que", "el", "una", "por", "para", "pero", "esta", "los", "del", "muy", "nada"}, "ñ"},
	{"pt", []string{"nao", "voce", "esta", "mas", "isso", "ela", "por", "para", "uma", "tambem", "muito", "dos"}, "ãçõ"},
	{"it", []string{"non", "che", "sono", "per", "piu", "questo", "hai", "come", "alla", "anche", "pero"}, ""},
	{"sv", []string{"och", "att", "inte", "jag", "med", "för", "har", "du", "ska", "bra", "dig"}, ""},
	{"da", []string{"og", "at", "ikke", "jeg", "med", "for", "har", "du", "ger", "noget", "dig"}, "æø"},
	{"no", []string{"og", "ikke", "jeg", "deg", "for", "med", "har", "du", "skal", "noe"}, "æø"},
	{"fi", []string{"ja", "on", "ei", "minä", "hän", "että", "olen", "vai", "mitä", "kun"}, "äö"},
	{"et", []string{"ja", "on", "ei", "pole", "olen", "minu", "see", "kas", "väga", "näen", "äri"}, "õ"},
	{"pl", []string{"nie", "jest", "się", "że", "pan", "ale", "jak", "tego", "chyba", "cię", "bardzo", "jestem"}, "żśąćęłń"},
	{"cs", []string{"ne", "je", "se", "jak", "ale", "ten", "pro", "jsem", "ze", "moc", "byl"}, "ěřůň"},
	{"sk", []string{"nie", "som", "ako", "ale", "je", "to", "moc", "bol", "že"}, "ä"},
	{"tr", []string{"ve", "bir", "bu", "için", "değil", "ama", "var", "ben", "çok", "diye", "gibi"}, "ığşâ"},
	{"hu", []string{"nem", "és", "hogy", "van", "igen", "ez", "szia", "nagyon", "csak"}, "őű"},
	{"ro", []string{"nu", "este", "că", "și", "cu", "să", "da", "mult", "aici"}, "ășț"},
	{"lt", []string{"ir", "yra", "kad", "aš", "tik", "su", "kaip", "mano", "labai", "noriu"}, "ėįū"},
	{"lv", []string{"un", "ir", "es", "ka", "nav", "ar", "to", "būtu"}, "āēņļ"},
	{"sl", []string{"ne", "je", "za", "sem", "da", "in", "kar", "bi", "tako"}, ""},
	{"hr", []string{"ne", "je", "sam", "ali", "što", "kao", "za", "tako", "bi"}, ""},
	{"id", []string{"yang", "tidak", "dan", "aku", "itu", "saya", "untuk", "dengan", "saja"}, ""},
	{"ms", []string{"yang", "tidak", "dan", "aku", "itu", "saya", "untuk", "dengan", "saja"}, ""},
	{"vi", []string{"không", "là", "và", "tôi", "để", "đó", "có", "được", "mình", "này", "với", "phải"}, "đệữưị"},
	{"is", []string{"og", "að", "ekki", "ég", "það", "já", "mér"}, "þð"},
}

// cyrillicProfiles are scored only when the text is mostly Cyrillic; without that
// split Russian would absorb every Cyrillic track, including Ukrainian and Bulgarian.
var cyrillicProfiles = []langRule{
	{"ru", []string{"не", "на", "что", "это", "как", "вы", "но", "да", "его", "будет", "тебя", "очень", "сейчас"}, ""},
	{"uk", []string{"як", "це", "ні", "та", "вони", "мені", "так", "але", "того"}, ""},
	{"bg", []string{"не", "че", "като", "ще", "да", "аз", "той", "много", "се"}, ""},
	{"sr", []string{"не", "је", "сам", "али", "што", "као", "за", "тако", "би"}, ""},
	{"mk", []string{"не", "да", "се", "во", "на", "што", "го", "ние", "тој", "много"}, ""},
	{"mn", []string{"би", "нь", "энэ", "юм", "дээ", "бол", "гэж", "байна", "зүйл", "ноён"}, ""},
}

// DetectSubtitleLanguage returns a 2-letter code for the dominant language of the
// given subtitle text, or "" when the text is too short or too mixed to be sure.
func DetectSubtitleLanguage(text string) string {
	cleaned := cleanSubtitleText(text)
	if cleaned == "" {
		return ""
	}
	if hint := scriptHint(cleaned); hint != "" {
		if hint == "cyrillic" {
			if code := scoreProfiles(cleaned, cyrillicProfiles); code != "" {
				return code
			}
			return "ru"
		}
		return hint
	}
	return scoreProfiles(cleaned, latinProfiles)
}

// cleanSubtitleText strips container markup (ASS override codes, HTML tags) and
// lowercases the text, so function words can be matched token by token.
func cleanSubtitleText(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch == '{' {
			if end := strings.Index(text[i:], "}"); end >= 0 {
				i += end
				continue
			}
		}
		if ch == '<' {
			if end := strings.Index(text[i:], ">"); end >= 0 {
				i += end
				continue
			}
		}
		b.WriteByte(ch)
	}
	return strings.ToLower(strings.ReplaceAll(b.String(), "'", ""))
}

// tokenize splits subtitle text into lowercase word tokens, dropping timestamps and
// punctuation-only fragments. Apostrophes are removed by cleanSubtitleText, so
// "don't" and "c'est" become "dont" and "cest".
func tokenize(text string) map[string]int {
	counts := map[string]int{}
	for _, raw := range strings.Fields(text) {
		tok := strings.TrimFunc(raw, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		if len([]rune(tok)) < 2 {
			continue
		}
		counts[tok]++
	}
	return counts
}

func scoreProfiles(text string, profiles []langRule) string {
	counts := tokenize(text)
	runes := []rune(text)
	totalLetters := 0
	for _, r := range runes {
		if unicode.IsLetter(r) {
			totalLetters++
		}
	}
	if totalLetters == 0 {
		return ""
	}

	best, bestScore := "", 0.0
	for _, p := range profiles {
		if len(p.words) == 0 {
			continue
		}
		hits, distinct := 0, 0
		for _, w := range p.words {
			if c := counts[w]; c > 0 {
				hits += c
				distinct++
			}
		}
		markerHits, markerDistinct := 0, 0
		for _, w := range langMarkers[p.code] {
			if c := counts[w]; c > 0 {
				markerHits += c
				markerDistinct++
			}
		}
		// Two distinct exclusive markers are stronger evidence than a handful of
		// function words that neighbouring languages share, so either kind of
		// evidence is enough before a profile is allowed to score.
		wordEvidence := hits >= langMinHits && distinct >= langMinDistinct
		markerEvidence := markerHits >= langMinHits && markerDistinct >= 2
		if !wordEvidence && !markerEvidence {
			continue
		}
		special := 0
		if p.letter != "" {
			for _, r := range runes {
				if strings.ContainsRune(p.letter, r) {
					special++
				}
			}
		}
		score := float64(hits) + 3*float64(distinct) +
			float64(langMarkerWeight)*(float64(markerHits)+float64(markerDistinct)) +
			30*float64(special)/float64(totalLetters)
		if score > bestScore {
			best, bestScore = p.code, score
		}
	}
	return best
}

// scriptHint labels a script that identifies the language on its own. Cyrillic is
// returned as a marker rather than a code because several languages share it.
func scriptHint(text string) string {
	var letters, kana, hangul, han, cyrillic, greek, arabic, hebrew, thai, devanagari int
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		switch {
		case r >= 0x3040 && r <= 0x30ff:
			kana++
		case r >= 0xac00 && r <= 0xd7af:
			hangul++
		case r >= 0x4e00 && r <= 0x9fff:
			han++
		case r >= 0x0400 && r <= 0x04ff:
			cyrillic++
		case r >= 0x0370 && r <= 0x03ff:
			greek++
		case r >= 0x0600 && r <= 0x06ff:
			arabic++
		case r >= 0x0590 && r <= 0x05ff:
			hebrew++
		case r >= 0x0e00 && r <= 0x0e7f:
			thai++
		case r >= 0x0900 && r <= 0x097f:
			devanagari++
		}
	}
	if letters == 0 {
		return ""
	}
	frac := func(n int) float64 { return float64(n) / float64(letters) }
	switch {
	case frac(kana) >= 0.12:
		return "ja"
	case frac(hangul) >= 0.30:
		return "ko"
	case frac(han) >= 0.50:
		return "zh"
	case frac(cyrillic) >= 0.50:
		return "cyrillic"
	case frac(greek) >= 0.40:
		return "el"
	case frac(arabic) >= 0.40:
		return "ar"
	case frac(hebrew) >= 0.40:
		return "he"
	case frac(thai) >= 0.40:
		return "th"
	case frac(devanagari) >= 0.40:
		return "hi"
	}
	return ""
}

// NeedsLangDetection reports whether a track has no usable language tag and holds
// text that can actually be inspected. Bitmap tracks (PGS, VobSub) have no text to
// sample, so they keep whatever tag the container carries.
func NeedsLangDetection(t *TrackItem) bool {
	if !isPreviewableCodec(t.Codec) {
		return false
	}
	return t.Lang == "" || strings.EqualFold(t.Lang, "und")
}

// videoDuration returns a container's runtime so the sample windows can be spread
// across it. A zero duration means the caller samples from the beginning only.
func videoDuration(videoPath string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), langDurationTimeout)
	defer cancel()

	cmd := NewFFprobeCmdContext(ctx,
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to read duration: %w", err)
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || secs <= 0 {
		return 0, fmt.Errorf("could not parse duration %q", strings.TrimSpace(string(out)))
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// sampleSubtitleWindow extracts one SRT window of a subtitle stream. Only the
// window is decoded, so this stays cheap even for large files.
func sampleSubtitleWindow(videoPath string, streamIndex int, start time.Duration) (string, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("sf_lang_%d_%d_%d.srt", os.Getpid(), streamIndex, start))
	defer os.Remove(tmpFile)

	ctx, cancel := context.WithTimeout(context.Background(), langSnippetTimeout)
	defer cancel()

	args := []string{"-v", "error", "-y"}
	if start > 0 {
		args = append(args, "-ss", strconv.FormatFloat(start.Seconds(), 'f', 3, 64))
	}
	args = append(args,
		"-i", videoPath,
		"-map", fmt.Sprintf("0:%d", streamIndex),
		"-t", strconv.Itoa(int(langWindowSpan.Seconds())),
		"-f", "srt",
		tmpFile,
	)

	output, err := NewFFmpegCmdContext(ctx, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to sample stream %d: %s (%s)", streamIndex, err, strings.TrimSpace(string(output)))
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to read sampled subtitle: %w", err)
	}
	defer f.Close()

	buf := make([]byte, langSnippetMaxBytes)
	n, _ := io.ReadFull(f, buf)
	return string(buf[:n]), nil
}

// sampleTrackText concatenates the sample windows of a track, skipping windows
// that fall outside the runtime.
func sampleTrackText(videoPath string, streamIndex int, duration time.Duration) (string, error) {
	if duration <= 0 {
		return sampleSubtitleWindow(videoPath, streamIndex, 0)
	}

	var b strings.Builder
	var lastErr error
	for _, fraction := range langWindows {
		text, err := sampleSubtitleWindow(videoPath, streamIndex, time.Duration(fraction*float64(duration)))
		if err != nil {
			lastErr = err
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return "", lastErr
	}
	return b.String(), nil
}

// detectTrackLanguages samples every untagged text track and fills in Lang.
// onProgress receives the number of sampled tracks and the number that needed
// sampling, so the caller can show progress while this blocks.
// It returns how many tracks got a language.
func detectTrackLanguages(videoPath string, tracks []*TrackItem, onProgress func(done, total int)) int {
	var pending []*TrackItem
	for _, t := range tracks {
		if NeedsLangDetection(t) {
			pending = append(pending, t)
		}
	}
	if len(pending) == 0 {
		return 0
	}

	total := len(pending)
	var wg sync.WaitGroup
	workers := make(chan struct{}, langDetectWorkers)
	var done, detected int64

	// One duration probe per source file, reused by every track of that file.
	var durMu sync.Mutex
	durations := map[string]time.Duration{}
	durationOf := func(path string) time.Duration {
		durMu.Lock()
		defer durMu.Unlock()
		if d, ok := durations[path]; ok {
			return d
		}
		d, err := videoDuration(path)
		if err != nil {
			AppLog("DETECT", "Duration probe failed for %s: %v (sampling from the beginning)", filepath.Base(path), err)
			d = 0
		}
		durations[path] = d
		return d
	}

	run := func(videoPath string, t *TrackItem) {
		defer wg.Done()
		defer func() { <-workers }()

		source := videoPath
		if t.FilePath != "" {
			source = t.FilePath
		}

		text, err := sampleTrackText(source, t.Num, durationOf(source))
		if err != nil {
			AppLog("DETECT", "Language sampling failed for track %d in %s: %v", t.Num, filepath.Base(source), err)
		} else if code := DetectSubtitleLanguage(text); code != "" {
			t.Lang = code
			t.LangDetected = true
			atomic.AddInt64(&detected, 1)
			AppLog("DETECT", "Track %d in %s detected as %s", t.Num, filepath.Base(source), code)
		}

		n := int(atomic.AddInt64(&done, 1))
		if onProgress != nil {
			onProgress(n, total)
		}
	}

	for _, t := range pending {
		wg.Add(1)
		workers <- struct{}{}
		go run(videoPath, t)
	}
	wg.Wait()

	return int(atomic.LoadInt64(&detected))
}

// trackLangDisplay renders a track's language, marking values that were guessed
// from the subtitle text rather than read from the container.
func trackLangDisplay(t *TrackItem) string {
	if t.LangDetected {
		return t.Lang + " (detected)"
	}
	return t.Lang
}
