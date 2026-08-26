package captions

import (
	"testing"

	"github.com/lucasepe/x/image/gg"
)

func TestTextPartFallsBackToTextWhenEmojiAssetIsMissing(t *testing.T) {
	ctx := gg.NewContext(defaultWidth, defaultHeight)
	face, err := loadFontFace("", defaultFontSize)
	if err != nil {
		t.Fatalf("loadFontFace() error = %v", err)
	}
	ctx.SetFontFace(face)

	part, err := textPart(ctx, "♪", Config{EmojiCacheDir: t.TempDir()}, defaultFontSize)
	if err != nil {
		t.Fatalf("textPart() error = %v", err)
	}
	if part.Value != "♪" {
		t.Fatalf("part.Value = %q, want ♪", part.Value)
	}
	if part.Emoji != nil {
		t.Fatal("part.Emoji is not nil, want text fallback")
	}
	if part.Width <= 0 {
		t.Fatalf("part.Width = %v, want positive width", part.Width)
	}
}
