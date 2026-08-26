package transcribe

import "testing"

func TestParseWhisperTimeAcceptsCommaMilliseconds(t *testing.T) {
	got, err := parseWhisperTime("00:00:04,560")
	if err != nil {
		t.Fatalf("parseWhisperTime() error = %v", err)
	}
	if got != 4.56 {
		t.Fatalf("parseWhisperTime() = %v, want 4.56", got)
	}
}

func TestParseWhisperTimeAcceptsDotMilliseconds(t *testing.T) {
	got, err := parseWhisperTime("00:01:02.340")
	if err != nil {
		t.Fatalf("parseWhisperTime() error = %v", err)
	}
	if got != 62.34 {
		t.Fatalf("parseWhisperTime() = %v, want 62.34", got)
	}
}

func TestWhisperSegmentsToOverlaysCanEmitOneOverlayPerWord(t *testing.T) {
	cfg := TranscribeConfig{
		Gravity:    "bottom",
		FontSize:   defaultFontSize,
		Padding:    defaultPadding,
		MaxChars:   defaultMaxChars,
		Box:        true,
		BoxAlpha:   defaultBoxAlpha,
		BoxPadding: 0,
		BoxRadius:  0,
		Words:      true,
	}
	segments := []WhisperSegment{
		{
			Text: "ciao mondo bello",
			Timestamps: WhisperTimestamps{
				From: "00:00:03,000",
				To:   "00:00:06,000",
			},
		},
	}

	overlays, err := whisperSegmentsToOverlays(segments, cfg)
	if err != nil {
		t.Fatalf("whisperSegmentsToOverlays() error = %v", err)
	}
	if len(overlays) != 3 {
		t.Fatalf("len(overlays) = %d, want 3", len(overlays))
	}
	if overlays[0].Text != "ciao" || overlays[0].Start != 3 || overlays[0].End != 4 {
		t.Fatalf("overlays[0] = %+v, want ciao from 3 to 4", overlays[0])
	}
	if overlays[2].Text != "bello" || overlays[2].Start != 5 || overlays[2].End != 6 {
		t.Fatalf("overlays[2] = %+v, want bello from 5 to 6", overlays[2])
	}
}

func TestTokenWordOverlaysUsesTokenTimestamps(t *testing.T) {
	cfg := TranscribeConfig{
		Gravity:    "bottom",
		FontSize:   defaultFontSize,
		Padding:    defaultPadding,
		MaxChars:   defaultMaxChars,
		Box:        true,
		BoxAlpha:   defaultBoxAlpha,
		BoxPadding: 0,
		BoxRadius:  0,
	}
	tokens := []WhisperToken{
		{Text: "[_BEG_]", Offsets: WhisperOffsets{From: 0, To: 0}},
		{Text: " Il", Offsets: WhisperOffsets{From: 0, To: 40}},
		{Text: " dir", Offsets: WhisperOffsets{From: 100, To: 270}},
		{Text: "let", Offsets: WhisperOffsets{From: 360, To: 430}},
		{Text: "ta", Offsets: WhisperOffsets{From: 430, To: 540}},
		{Text: " è", Offsets: WhisperOffsets{From: 750, To: 750}},
		{Text: " sem", Offsets: WhisperOffsets{From: 770, To: 890}},
		{Text: "plic", Offsets: WhisperOffsets{From: 910, To: 1130}},
		{Text: "emente", Offsets: WhisperOffsets{From: 1130, To: 1460}},
		{Text: " meditazione[_TT_228]", Offsets: WhisperOffsets{From: 1500, To: 2100}},
		{Text: " l'asfalto,", Offsets: WhisperOffsets{From: 2200, To: 2600}},
	}

	overlays := tokenWordOverlays(tokens, cfg)
	if len(overlays) != 6 {
		t.Fatalf("len(overlays) = %d, want 6", len(overlays))
	}
	if overlays[0].Text != "Il" || overlays[0].Start != 0 || overlays[0].End != 0.04 {
		t.Fatalf("overlays[0] = %+v, want Il from 0 to 0.04", overlays[0])
	}
	if overlays[1].Text != "dirletta" || overlays[1].Start != 0.1 || overlays[1].End != 0.54 {
		t.Fatalf("overlays[1] = %+v, want dirletta from 0.1 to 0.54", overlays[1])
	}
	if overlays[2].Text != "è" || overlays[2].Start != 0.75 || overlays[2].End != 0.77 {
		t.Fatalf("overlays[2] = %+v, want è from 0.75 to 0.77", overlays[2])
	}
	if overlays[3].Text != "semplicemente" || overlays[3].Start != 0.77 || overlays[3].End != 1.46 {
		t.Fatalf("overlays[3] = %+v, want semplicemente from 0.77 to 1.46", overlays[3])
	}
	if overlays[4].Text != "meditazione" || overlays[4].Start != 1.5 || overlays[4].End != 2.1 {
		t.Fatalf("overlays[4] = %+v, want meditazione from 1.5 to 2.1", overlays[4])
	}
	if overlays[5].Text != "l'asfalto" || overlays[5].Start != 2.2 || overlays[5].End != 2.6 {
		t.Fatalf("overlays[5] = %+v, want l'asfalto from 2.2 to 2.6", overlays[5])
	}
}

func TestCleanCaptionWordRemovesEdgePunctuation(t *testing.T) {
	tests := map[string]string{
		"trasporto,": "trasporto",
		"fine.":      "fine",
		"\"ciao\"":   "ciao",
		"l'asfalto":  "l'asfalto",
	}
	for input, want := range tests {
		if got := cleanCaptionWord(input); got != want {
			t.Fatalf("cleanCaptionWord(%q) = %q, want %q", input, got, want)
		}
	}
}
