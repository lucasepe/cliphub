package captions

import (
	"errors"
	"image"
	"unicode"

	"github.com/lucasepe/x/image/gg"
	"github.com/rivo/uniseg"
)

// TextPart represents one drawable item in a laid-out line.
type TextPart struct {
	Value string
	Emoji image.Image
	Width float64
	Chars int
}

// TextLine contains text and emoji parts that fit within one rendered row.
type TextLine struct {
	Parts []TextPart
	Width float64
	Chars int
}

type textToken struct {
	Parts []TextPart
	Space bool
}

// layoutLines keeps words intact while wrapping by rendered width and character count.
func layoutLines(ctx *gg.Context, cfg Config, overlay Overlay) ([]TextLine, error) {
	fontSize := *overlay.FontSize
	maxWidth := float64(cfg.Width) - *overlay.PaddingLeft - *overlay.PaddingRight
	maxChars := *overlay.MaxChars
	tokens, err := textTokens(ctx, cfg, overlay.Text, fontSize)
	if err != nil {
		return nil, err
	}
	var lines []TextLine
	var current TextLine
	var spaces []TextPart

	for _, token := range tokens {
		if token.Space {
			if len(current.Parts) > 0 {
				spaces = append(spaces, token.Parts...)
			}
			continue
		}

		candidate := appendParts(current, spaces, fontSize)
		candidate = appendParts(candidate, token.Parts, fontSize)
		if len(current.Parts) > 0 && !lineFits(candidate, maxWidth, maxChars) {
			lines = append(lines, trimLine(current, fontSize))
			current = TextLine{}
			spaces = nil
		}

		word := appendParts(TextLine{}, token.Parts, fontSize)
		if len(current.Parts) == 0 && !lineFits(word, maxWidth, maxChars) {
			for _, part := range token.Parts {
				withPart := appendParts(current, []TextPart{part}, fontSize)
				if len(current.Parts) > 0 && !lineFits(withPart, maxWidth, maxChars) {
					lines = append(lines, trimLine(current, fontSize))
					current = TextLine{}
				}
				current = appendParts(current, []TextPart{part}, fontSize)
			}
			continue
		}

		current = appendParts(current, spaces, fontSize)
		current = appendParts(current, token.Parts, fontSize)
		spaces = nil
	}
	if len(current.Parts) > 0 {
		lines = append(lines, trimLine(current, fontSize))
	}
	if len(lines) == 0 {
		return []TextLine{{Parts: []TextPart{{Value: "", Chars: 0}}}}, nil
	}
	return lines, nil
}

// textTokens groups grapheme clusters into whitespace and non-whitespace runs.
func textTokens(ctx *gg.Context, cfg Config, value string, fontSize float64) ([]textToken, error) {
	var tokens []textToken
	graphemes := uniseg.NewGraphemes(value)
	for graphemes.Next() {
		cluster := graphemes.Str()
		part, err := textPart(ctx, cluster, cfg, fontSize)
		if err != nil {
			return nil, err
		}
		space := isSpace(cluster)
		if len(tokens) == 0 || tokens[len(tokens)-1].Space != space {
			tokens = append(tokens, textToken{Space: space})
		}
		tokens[len(tokens)-1].Parts = append(tokens[len(tokens)-1].Parts, part)
	}
	return tokens, nil
}

func appendParts(line TextLine, parts []TextPart, fontSize float64) TextLine {
	line.Parts = append(line.Parts, parts...)
	line.Width = lineWidth(line.Parts, fontSize)
	line.Chars = lineChars(line.Parts)
	return line
}

func lineFits(line TextLine, maxWidth float64, maxChars int) bool {
	return line.Width <= maxWidth && line.Chars <= maxChars
}

// textPart returns a drawable part for a grapheme cluster, using Twemoji PNGs for emoji clusters.
func textPart(ctx *gg.Context, cluster string, cfg Config, fontSize float64) (TextPart, error) {
	if isEmojiCluster(cluster) {
		img, err := loadEmojiImage(cluster, cfg.EmojiCacheDir)
		if err != nil {
			if !errors.Is(err, errEmojiAssetNotFound) {
				return TextPart{}, err
			}
			width, _ := ctx.MeasureString(cluster)
			return TextPart{Value: cluster, Width: width, Chars: len([]rune(cluster))}, nil
		}
		return TextPart{Value: cluster, Emoji: img, Width: fontSize, Chars: 1}, nil
	}
	width, _ := ctx.MeasureString(cluster)
	return TextPart{Value: cluster, Width: width, Chars: len([]rune(cluster))}, nil
}

// emojiGap returns extra spacing between adjacent emoji, scaled from the configured font size.
func emojiGap(fontSize float64) float64 {
	return fontSize * emojiGapRatio
}

// isEmojiCluster reports whether any rune in a grapheme cluster is in common emoji Unicode ranges.
func isEmojiCluster(cluster string) bool {
	if containsVariationSelector(cluster) {
		return true
	}
	for _, r := range cluster {
		if r >= 0x2669 && r <= 0x266F {
			return false
		}
		switch {
		case r >= 0x2190 && r <= 0x21FF:
			return true
		case r >= 0x2300 && r <= 0x23FF:
			return true
		case r >= 0x2460 && r <= 0x24FF:
			return true
		case r >= 0x25A0 && r <= 0x25FF:
			return true
		case r >= 0x1F000 && r <= 0x1FAFF:
			return true
		case r >= 0x2600 && r <= 0x27BF:
			return true
		case r >= 0x2900 && r <= 0x297F:
			return true
		case r >= 0x2B00 && r <= 0x2BFF:
			return true
		case r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299:
			return true
		}
	}
	return false
}

// containsVariationSelector reports whether a cluster requests emoji presentation.
func containsVariationSelector(cluster string) bool {
	for _, r := range cluster {
		if r == 0xFE0F {
			return true
		}
	}
	return false
}

// isSpace reports whether a grapheme cluster contains only Unicode whitespace.
func isSpace(cluster string) bool {
	for _, r := range cluster {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return cluster != ""
}

// trimLine removes leading and trailing whitespace parts and recomputes the line width.
func trimLine(line TextLine, fontSize float64) TextLine {
	start := 0
	end := len(line.Parts)
	for start < end && isSpace(line.Parts[start].Value) {
		start++
	}
	for end > start && isSpace(line.Parts[end-1].Value) {
		end--
	}
	line.Parts = line.Parts[start:end]
	line.Width = lineWidth(line.Parts, fontSize)
	line.Chars = lineChars(line.Parts)
	return line
}

// lineWidth sums part widths and includes extra spacing for adjacent emoji pairs.
func lineWidth(parts []TextPart, fontSize float64) float64 {
	var width float64
	previousWasEmoji := false
	for _, part := range parts {
		if previousWasEmoji && part.Emoji != nil {
			width += emojiGap(fontSize)
		}
		width += part.Width
		previousWasEmoji = part.Emoji != nil
	}
	return width
}

// lineChars sums the character counts of drawable text parts after whitespace trimming.
func lineChars(parts []TextPart) int {
	var chars int
	for _, part := range parts {
		chars += part.Chars
	}
	return chars
}

// widestLine returns the largest measured line width in a laid-out text block.
func widestLine(lines []TextLine) float64 {
	var width float64
	for _, line := range lines {
		if line.Width > width {
			width = line.Width
		}
	}
	return width
}
