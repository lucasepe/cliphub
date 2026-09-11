package captions

import (
	"slices"
	"strings"
	"testing"

	"github.com/lucasepe/x/image/gg"
)

func TestLayoutLinesWrapsBeforeWholeWord(t *testing.T) {
	ctx := gg.NewContext(2000, 500)
	face, err := loadFontFace("", defaultFontSize)
	if err != nil {
		t.Fatalf("loadFontFace() error = %v", err)
	}
	ctx.SetFontFace(face)

	fontSize := float64(defaultFontSize)
	padding := 0.0
	maxChars := 8
	lines, err := layoutLines(ctx, Config{Width: 2000}, Overlay{
		Text:         "hello button",
		FontSize:     &fontSize,
		PaddingLeft:  &padding,
		PaddingRight: &padding,
		MaxChars:     &maxChars,
	})
	if err != nil {
		t.Fatalf("layoutLines() error = %v", err)
	}
	if got, want := lineText(lines), []string{"hello", "button"}; !slices.Equal(got, want) {
		t.Fatalf("layoutLines() text = %#v, want %#v", got, want)
	}
}

func TestLayoutLinesWrapsWholeWordByRenderedWidth(t *testing.T) {
	ctx := gg.NewContext(2000, 500)
	face, err := loadFontFace("", defaultFontSize)
	if err != nil {
		t.Fatalf("loadFontFace() error = %v", err)
	}
	ctx.SetFontFace(face)

	helloWidth, _ := ctx.MeasureString("hello")
	buttonWidth, _ := ctx.MeasureString("button")
	maxWidth := max(helloWidth, buttonWidth) + 1
	fontSize := float64(defaultFontSize)
	padding := 0.0
	maxChars := 100
	lines, err := layoutLines(ctx, Config{Width: int(maxWidth)}, Overlay{
		Text:         "hello button",
		FontSize:     &fontSize,
		PaddingLeft:  &padding,
		PaddingRight: &padding,
		MaxChars:     &maxChars,
	})
	if err != nil {
		t.Fatalf("layoutLines() error = %v", err)
	}
	if got, want := lineText(lines), []string{"hello", "button"}; !slices.Equal(got, want) {
		t.Fatalf("layoutLines() text = %#v, want %#v", got, want)
	}
}

func lineText(lines []TextLine) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		var value strings.Builder
		for _, part := range line.Parts {
			value.WriteString(part.Value)
		}
		result[i] = value.String()
	}
	return result
}

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
