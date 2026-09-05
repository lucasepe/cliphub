package script

import (
	"testing"
)

func TestTextItemsUsesLinesOrWords(t *testing.T) {
	text := "Prima riga.\n\nSeconda riga lunga!\n"
	lines, count := textItems(text, false)
	if count != 5 || len(lines) != 2 || lines[1] != "Seconda riga lunga!" {
		t.Fatalf("textItems(lines) = %#v, %d", lines, count)
	}
	words, count := textItems(text, true)
	if count != 5 || len(words) != 5 || words[2] != "Seconda" {
		t.Fatalf("textItems(words) = %#v, %d", words, count)
	}
}

func TestTimedOverlaysCoverDurationAndUseDefaults(t *testing.T) {
	overlays := timedOverlays([]string{"Ciao", "una frase decisamente più lunga."}, 10)
	if len(overlays) != 2 {
		t.Fatalf("len(overlays) = %d", len(overlays))
	}
	if overlays[0].Start != 0 || overlays[1].End != 10 || overlays[0].End != overlays[1].Start {
		t.Fatalf("unexpected timeline: %+v", overlays)
	}
	if overlays[0].Gravity != "bottom" || overlays[0].FontSize == nil || *overlays[0].FontSize != 80 {
		t.Fatalf("unexpected defaults: %+v", overlays[0])
	}
	if overlays[0].End >= 5 {
		t.Fatalf("short item should receive less than half the duration: %+v", overlays)
	}
}

func TestWordsPerMinute(t *testing.T) {
	if got := wordsPerMinute(100, 30); got != 200 {
		t.Fatalf("wordsPerMinute = %v, want 200", got)
	}
}
