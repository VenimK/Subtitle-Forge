package main

import (
	"fmt"
	"strings"
	"testing"
)

// srtSnippet wraps dialogue lines in the shape ffmpeg produces, including the
// markup the detector has to ignore.
func srtSnippet(lines ...string) string {
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%d\n00:00:%02d,000 --> 00:00:%02d,500\n<i>%s</i>\n\n", i+1, i, i+1, line)
	}
	return b.String()
}

func TestDetectSubtitleLanguage(t *testing.T) {
	cases := []struct {
		name string
		want string
		text string
	}{
		{
			name: "dutch",
			want: "nl",
			text: srtSnippet(
				"Heerlijk om weer in Newport te zijn.",
				"Hoort deze halte bij je verrassing? - Ja.",
				"Is dat alles wat je te zeggen hebt?",
				"Ik heb zo meteen van alles te zeggen.",
				"Wat vind je van de buurt? - Het ziet er leuk uit.",
				"Maar het huis is nog niet klaar.",
				"Weet je wat ik bedoel? - Nee, ik weet het niet.",
				"Dit is voor jou, niet voor mij.",
			),
		},
		{
			name: "english",
			want: "en",
			text: srtSnippet(
				"I do love being back in Newport.",
				"What do you want from me?",
				"Don't you think it's time we talked?",
				"That was not what I meant at all.",
				"You have no idea what I'm going through.",
				"Was it really that simple for you?",
				"The thing is, I don't have the answers.",
			),
		},
		{
			name: "german",
			want: "de",
			text: srtSnippet(
				"Ich bin froh, wieder in Newport zu sein.",
				"Und? Was willst du von mir?",
				"Das ist nicht mein Problem, ich habe es nicht gesagt.",
				"Wir müssen darüber sprechen, aber ich weiß nicht wie.",
				"Mein Gott, ich habe dich doch gewarnt.",
				"Auch das ist nicht der Rede wert.",
			),
		},
		{
			name: "french",
			want: "fr",
			text: srtSnippet(
				"Je suis heureuse d'être de retour à Newport.",
				"Vous ne pouvez pas me faire ça.",
				"C'est la dernière fois que je vous parle.",
				"Les gens d'ici sont très différents de nous.",
				"Je n'ai pas le choix, mais je veux partir.",
				"Pourquoi dans ce quartier? - Parce que c'est chez moi.",
			),
		},
		{
			name: "spanish",
			want: "es",
			text: srtSnippet(
				"Me encanta estar de vuelta en Newport.",
				"No es lo que yo quiero, es lo que ella quiere.",
				"¿Pero qué vas a hacer con toda esta gente?",
				"Para mí es muy difícil, la verdad.",
				"Los vecinos dicen que es una buena idea.",
				"Del pasado no hablamos, nada más.",
			),
		},
		{
			name: "portuguese",
			want: "pt",
			text: srtSnippet(
				"Não acredito que estou de volta a Newport.",
				"Você não pode fazer isso comigo, não pode.",
				"Isso é tudo o que você tem a dizer?",
				"Ela não está aqui, mas eu estou.",
				"Muito bonito, mas não é para mim.",
				"Uma festa para todos nós.",
			),
		},
		{
			name: "italian",
			want: "it",
			text: srtSnippet(
				"Sono felice di essere di nuovo a Newport.",
				"Non è quello che voglio, è quello che vuole lei.",
				"Che cosa hai detto? - Non ho detto niente.",
				"Per me è molto difficile, anche se sembra facile.",
				"Questo non è il mio mondo, ma il tuo.",
				"Come facciamo adesso?",
			),
		},
		{
			name: "dutch-vs-finnish-edge",
			want: "fi",
			text: srtSnippet(
				"Onpa hienoa olla taas Newportissa.",
				"En tiedä, mitä sanoin. - Mitä sinä sanoit?",
				"Minä olen täällä, hän on siellä.",
				"Se ei ole minun vikani, vai onko?",
				"Kun minä sanoin, ettet tule.",
				"Vaikea kysymys, mutta vastaus on ei.",
			),
		},
		{
			name: "estonian",
			want: "et",
			text: srtSnippet(
				"Mulle meeldib taas Newportis olla.",
				"Ma ei teada, mida öelda. - Kas sa ütlesid midagi?",
				"See pole minu süü, või on?",
				"Minu arvates on see väga hea mõte.",
				"Ma olen siin, tema on seal.",
				"Kas sa näed seda? - Ei, ma ei näe.",
			),
		},
		{
			name: "lithuanian",
			want: "lt",
			text: srtSnippet(
				"Kaip gera vėl būti Niuporte.",
				"Aš nežinau, ką pasakyti. - Ar tu ką nors pasakei?",
				"Tai nėra mano kaltė, ar yra?",
				"Man mano, kad tai labai gera idėja.",
				"Aš esu čia, jis yra ten.",
				"Kaip tu tai padarai? - Noriu pabandyti.",
			),
		},
		{
			name: "vietnamese",
			want: "vi",
			text: srtSnippet(
				"THỜI ĐẠI VÀNG SON",
				"Thật tuyệt khi được quay lại Newport.",
				"Tôi không biết phải nói gì. - Anh có nói gì không?",
				"Đó không phải lỗi của tôi, đúng không?",
				"Tôi nghĩ đây là một ý tưởng rất hay.",
				"Tôi ở đây, còn ông ấy ở kia.",
				"Anh làm cách nào? - Tôi muốn thử.",
			),
		},
		{
			name: "polish",
			want: "pl",
			text: srtSnippet(
				"Cieszę się, że znowu jestem w Newport.",
				"Nie wiem, co powiedzieć. - Czy pan coś powiedział?",
				"To nie moja wina, ale może jednak.",
				"Myślę, że to bardzo dobry pomysł.",
				"Chyba jestem już zmęczona tą rozmową.",
				"Jak ty to robisz? - Bardzo prosto.",
			),
		},
		{
			name: "russian",
			want: "ru",
			text: srtSnippet(
				"Как хорошо снова быть в Ньюпорте.",
				"Я не знаю, что сказать. - А вы что-нибудь сказали?",
				"Это не моя вина, но, возможно.",
				"Думаю, это очень хорошая идея.",
				"Я здесь, а он там.",
				"Как ты это делаешь? - Мне хочется попробовать.",
			),
		},
		{
			name: "ukrainian",
			want: "uk",
			text: srtSnippet(
				"Як добре знову бути в Ньюпорті.",
				"Я не знаю, що сказати. - А ви щось сказали?",
				"Це не моя провина, але, можливо.",
				"Здається, це так добра ідея.",
				"Я тут, а він там. - Та ні, він тут.",
				"Мені треба спробувати, ви ж розумієте.",
			),
		},
		{
			name: "turkish",
			want: "tr",
			text: srtSnippet(
				"Newport'ta yeniden olmak çok güzel.",
				"Ne diyeceğimi bilmiyorum. - Sen bir şey söyledin mi?",
				"Bu benim suçum değil, ama belki.",
				"Bence bu çok iyi bir fikir.",
				"Ben buradayım, o ise orada.",
				"Bunu nasıl yaptın? - Denemek istedim.",
			),
		},
		{
			name: "japanese",
			want: "ja",
			text: srtSnippet(
				"ニューポートに戻れてうれしい。",
				"何て言ったらいいかわからない。 - 何か言ったの？",
				"わたしのせいじゃない、maybe?",
				"すごくいい考えだと思うわ。",
				"私はここにいる、あの人あそこにいる。",
			),
		},
		{
			name: "korean",
			want: "ko",
			text: srtSnippet(
				"뉴포트에 돌아오니 정말 좋아요.",
				"무슨 말 해야 할지 모르겠어요. - 뭐라고 말했어요?",
				"내 잘못이 아니잖아요. - 그런 것 같기도 해요.",
				"정말 좋은 생각인 것 같아요.",
				"나는 여기 있고, 그는 저기 있어요.",
			),
		},
		{
			name: "chinese",
			want: "zh",
			text: srtSnippet(
				"能回到纽波特真好。",
				"我不知道该说什么。 - 你刚才说了什么吗？",
				"这不是我的错，也许是。",
				"我觉得这是个好主意。",
				"我在这里，他在那里。",
				"你怎么做到的？我想试试。",
			),
		},
		{
			name: "greek",
			want: "el",
			text: srtSnippet(
				"Είναι ωραίο που ξαναείμαι στο Νιούπορτ.",
				"Δεν ξέρω τι να πω. - Εσύ κάτι είπες;",
				"Δεν είναι δικό μου λάθος, αλλά ίσως.",
				"Νομίζω ότι είναι πολύ καλή ιδέα.",
				"Εγώ είμαι εδώ, αυτός είναι εκεί.",
			),
		},
		{
			name: "arabic",
			want: "ar",
			text: srtSnippet(
				"من الجميل أن أعود إلى نيوبورت.",
				"لا أعرف ماذا أقول. - هل قلت شيئاً؟",
				"هذا ليس خطئي، لكن ربما.",
				"أعتقد أنها فكرة جيدة جداً.",
				"أنا هنا، وهو هناك.",
			),
		},
		{
			name: "hebrew",
			want: "he",
			text: srtSnippet(
				"זה נהדר לחזור לניופורט.",
				"אני לא יודע מה לומר. - אמרת משהו?",
				"זה לא האשם שלי, אבל אולי.",
				"אני חושב שזה רעיון טוב מאוד.",
				"אני כאן, והוא שם.",
			),
		},
		{
			name: "thai",
			want: "th",
			text: srtSnippet(
				"ดีจริงที่ได้กลับมานิวพอร์ต",
				"ไม่รู้ว่าจะพูดอะไร - เธอพูดอะไรรึยัง",
				"ไม่ใช่ความผิดของฉัน แต่ก็อาจใช่",
				"ฉันคิดว่ามันเป็นไอเดียที่ดีมาก",
				"ฉันอยู่ที่นี่ เขาอยู่โน่น",
			),
		},
		{
			name: "swedish",
			want: "sv",
			text: srtSnippet(
				"Det är skönt att vara tillbaka i Newport.",
				"Jag vet inte vad jag ska säga. - Sa du något?",
				"Det är inte mitt fel, men kanske.",
				"Jag tycker att det är en bra idé.",
				"Jag är här, du är där.",
				"Hur gör du det? - Jag vill prova.",
			),
		},
		{
			name: "romanian",
			want: "ro",
			text: srtSnippet(
				"E frumos să fiu din nou în Newport.",
				"Nu știu ce să spun. - Ai spus ceva?",
				"Nu e vina mea, dar poate.",
				"Cred că e o idee foarte bună.",
				"Eu sunt aici, el este acolo.",
				"Cum faci asta? - Vreau să încerc.",
			),
		},
		{
			name: "hungarian",
			want: "hu",
			text: srtSnippet(
				"Jó újra Newportban lenni.",
				"Nem tudom, mit mondjak. - Mondtál valamit?",
				"Nem az én hibám, de talán.",
				"Úgy gondolom, ez egy nagyon jó ötlet.",
				"Én itt vagyok, ő ott van.",
				"Hogyan csináltad? - Meg akarom próbálni.",
			),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetectSubtitleLanguage(c.text)
			if got != c.want {
				t.Errorf("DetectSubtitleLanguage() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDetectSubtitleLanguageTooShort(t *testing.T) {
	// A stream with only timestamps and numbering carries no language evidence.
	if got := DetectSubtitleLanguage("1\n00:00:12,000 --> 00:00:15,000\n\n2\n00:00:15,000 --> 00:00:18,000\n\n"); got != "" {
		t.Errorf("DetectSubtitleLanguage() = %q, want empty for a textless snippet", got)
	}
	if got := DetectSubtitleLanguage(""); got != "" {
		t.Errorf("DetectSubtitleLanguage() = %q, want empty for empty input", got)
	}
}

func TestDetectSubtitleLanguageLeavesUnknownAlone(t *testing.T) {
	// A mostly proper-noun snippet must not be forced into a language.
	text := srtSnippet("Newport.", "James.", "Scott.", "Bertha.", "Russell.", "Ott.", "Ada.")
	if got := DetectSubtitleLanguage(text); got != "" {
		t.Errorf("DetectSubtitleLanguage() = %q, want empty for name-only snippet", got)
	}
}

func TestNeedsLangDetection(t *testing.T) {
	cases := []struct {
		name  string
		track *TrackItem
		want  bool
	}{
		{"untagged text track", &TrackItem{Lang: "und", Codec: "SubRip/SRT"}, true},
		{"missing language", &TrackItem{Lang: "", Codec: "subrip"}, true},
		{"already tagged", &TrackItem{Lang: "dut", Codec: "SubRip/SRT"}, false},
		{"pgs has no text to sample", &TrackItem{Lang: "und", Codec: "hdmv_pgs_subtitle"}, false},
		{"vobsub has no text to sample", &TrackItem{Lang: "und", Codec: "vobsub"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NeedsLangDetection(c.track); got != c.want {
				t.Errorf("NeedsLangDetection() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestTrackLangDisplay(t *testing.T) {
	tagged := &TrackItem{Lang: "dut"}
	if got := trackLangDisplay(tagged); got != "dut" {
		t.Errorf("trackLangDisplay() = %q, want the container tag unchanged", got)
	}
	guessed := &TrackItem{Lang: "nl", LangDetected: true}
	if got := trackLangDisplay(guessed); got != "nl (detected)" {
		t.Errorf("trackLangDisplay() = %q, want %q", got, "nl (detected)")
	}
}

func TestDetectTrackLanguagesFillsUntaggedTracks(t *testing.T) {
	// No video file is involved: tracks that do not need sampling must be left
	// untouched and the helper must report zero detections.
	tracks := []*TrackItem{
		{Num: 1, Lang: "eng", Codec: "SubRip/SRT"},
		{Num: 2, Lang: "und", Codec: "hdmv_pgs_subtitle"},
	}
	if got := detectTrackLanguages("", tracks, nil); got != 0 {
		t.Errorf("detectTrackLanguages() = %d, want 0", got)
	}
	if tracks[0].Lang != "eng" || tracks[0].LangDetected {
		t.Errorf("tagged track was modified: lang=%q detected=%v", tracks[0].Lang, tracks[0].LangDetected)
	}
	if tracks[1].Lang != "und" || tracks[1].LangDetected {
		t.Errorf("bitmap track was modified: lang=%q detected=%v", tracks[1].Lang, tracks[1].LangDetected)
	}
}

// TestDetectSubtitleLanguageNeighbourGroups covers languages that share their
// most common function words, so the marker tier is what has to decide.
func TestDetectSubtitleLanguageNeighbourGroups(t *testing.T) {
	cases := []struct {
		name string
		want string
		text string
	}{
		{
			name: "czech",
			want: "cs",
			text: srtSnippet(
				"Jsem tady. Co se děje? - To nic není.",
				"Ano, vím to. Ještě chvíli počkám.",
				"Který rok? - Velmi dlouho.",
				"Proč se ptáš? - Co je tam?",
			),
		},
		{
			name: "slovak",
			want: "sk",
			text: srtSnippet(
				"Nie som chorý. Čo sa stalo? - Teraz idem von.",
				"Sme tu. Všetko je v poriadku.",
				"Tiež prídem aj ja. - To je dobré.",
				"Nie je to ťažké.",
			),
		},
		{
			name: "slovenian",
			want: "sl",
			text: srtSnippet(
				"Nisem bolan. Kaj se dogaja? - Vse je v redu.",
				"Zakaj vprašuješ? - Tudi jaz vem.",
				"Kjer sem bil, tam ni nič.",
				"Večkrat pridem domov pozno.",
			),
		},
		{
			name: "croatian",
			want: "hr",
			text: srtSnippet(
				"Nisam bolestan. Što je s tobom? - Ali to nije moje.",
				"Kao uvijek. Ništa nije važno.",
				"Gdje si sada? - Nešto ću reći.",
				"Nešto se promijenilo. - Što to znači?",
			),
		},
		{
			name: "serbian-cyrillic",
			want: "sr",
			text: srtSnippet(
				"Шта је то? - Нисам болестан.",
				"Нећу да кажем. Али то је све.",
				"Као увек. То је ништа за мене.",
				"Свако ко зна прича о томе.",
			),
		},
		{
			name: "macedonian",
			want: "mk",
			text: srtSnippet(
				"Не сум болен. - Што ти е?",
				"Ќе одам дома сега. - Ова е ништо.",
				"Ние не сакаме тоа. - Што да правам?",
				"Многу благодарам. - Сум тука.",
			),
		},
		{
			name: "mongolian",
			want: "mn",
			text: srtSnippet(
				"Би өвдөөгүй. - Юм биш.",
				"Энэ зүйл бол сайн. - Гэж хэлэв.",
				"Ноён тэр ирэх үү? - Байсан болов уу.",
				"Зүйл биш. - Би явлаа.",
			),
		},
		{
			name: "danish",
			want: "da",
			text: srtSnippet(
				"Jeg er ikke syg. Hvad kan jeg gøre? - Det nok er lidt koldt.",
				"Snart kommer han. - Måske aldrig.",
				"Jeg gjorde det i går. - Ikke igen.",
				"Det er lidt slemt.",
			),
		},
		{
			name: "norwegian",
			want: "no",
			text: srtSnippet(
				"Jeg er ikke syk. Noen kan hjelpe. - Veldig snilt av deg.",
				"Derfor kommer jeg. - Hva sier du?",
				"Jeg har ikke noe valg. - Nei, det er sant.",
				"Veldig kjapt. - Noen ganger.",
			),
		},
		{
			name: "indonesian",
			want: "id",
			text: srtSnippet(
				"Aku tidak sakit. Apa yang bisa kamu lakukan?",
				"Saya hanya ingin tidur, tidak lebih.",
				"Ibu ada di rumah. - Tidak apa-apa.",
				"Saya sangat lelah. - Baiklah.",
			),
		},
		{
			name: "malay",
			want: "ms",
			text: srtSnippet(
				"Saya tidak sakit. Apa yang awak boleh buat?",
				"Cuba fikir dulu. - Ramai orang di sana.",
				"Awak pandai sangat. - Macam biasa.",
				"Kerana itu saya tinggal. - Awak yakin?",
			),
		},
		{
			name: "portuguese-vs-spanish",
			want: "pt",
			text: srtSnippet(
				"Não estou doente. - Não é verdade.",
				"O que você tem a dizer? - Isso é tudo.",
				"Muito bonito, mas não é para mim.",
				"Uma festa para todos nós. - Estou aqui.",
			),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectSubtitleLanguage(c.text); got != c.want {
				t.Errorf("DetectSubtitleLanguage() = %q, want %q", got, c.want)
			}
		})
	}
}
