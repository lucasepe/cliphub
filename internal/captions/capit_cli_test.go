package captions

import (
	"path/filepath"
	"testing"
)

func TestDefaultCaptionedPath(t *testing.T) {
	got := defaultCaptionedPath(filepath.Join("testdata", "D01_clip_4.MP4"))
	want := filepath.Join("testdata", "D01_clip_4_captioned.MP4")
	if got != want {
		t.Fatalf("defaultCaptionedPath() = %q, want %q", got, want)
	}
}

func TestDefaultOverlaysPath(t *testing.T) {
	got := DefaultOverlaysPath(filepath.Join("testdata", "D01_clip_4.MP4"))
	want := filepath.Join("testdata", "D01_clip_4_overlays.json")
	if got != want {
		t.Fatalf("defaultOverlaysPath() = %q, want %q", got, want)
	}
}
