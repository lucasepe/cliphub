package captions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOverlayFileRejectsUnknownProperty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overlays.json")
	data := `[{"text":"hello","start":0,"end":1,"padding":250}]`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := readOverlayFile(path)
	if err == nil || !strings.Contains(err.Error(), `unknown property "padding"`) {
		t.Fatalf("readOverlayFile() error = %v, want clear unknown property error", err)
	}
}

func TestInstagramReelSafeAreaCanBeOverridden(t *testing.T) {
	left := 120.0
	overlays, err := applyOverlayDefaults([]Overlay{{
		Text: "hello", Start: 0, End: 1, SafeArea: "instagram-reel", PaddingLeft: &left,
	}})
	if err != nil {
		t.Fatal(err)
	}
	overlay := overlays[0]
	if *overlay.PaddingTop != instagramTop || *overlay.PaddingBottom != instagramBottom {
		t.Fatalf("preset vertical padding = %v/%v", *overlay.PaddingTop, *overlay.PaddingBottom)
	}
	if *overlay.PaddingLeft != left || *overlay.PaddingRight != defaultPadding {
		t.Fatalf("horizontal padding = %v/%v", *overlay.PaddingLeft, *overlay.PaddingRight)
	}
}
