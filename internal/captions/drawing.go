package captions

import (
	"image"
	"image/color"

	"github.com/lucasepe/x/image/gg"
)

// drawTextBackground paints an optional rounded rectangle behind the entire laid-out text block.
func drawTextBackground(ctx *gg.Context, lines []TextLine, centerX, topY, lineHeight float64, cfg Config, overlay Overlay) {
	fontSize := *overlay.FontSize
	boxPad := effectiveBoxPadding(overlay)
	boxRadius := *overlay.BoxRadius
	if boxRadius == 0 {
		boxRadius = fontSize * 0.25
	}

	boxWidth := widestLine(lines) + boxPad*2
	boxHeight := lineHeight*float64(len(lines)) + boxPad*2
	boxX := centerX - boxWidth/2
	boxY := topY - fontSize*0.85 - boxPad

	ctx.SetRGBA(0, 0, 0, *overlay.BoxAlpha)
	ctx.DrawRoundedRectangle(boxX, boxY, boxWidth, boxHeight, boxRadius)
	ctx.Fill()
}

// effectiveBoxPadding returns the configured caption box padding or a font-scaled default.
func effectiveBoxPadding(overlay Overlay) float64 {
	boxPad := *overlay.BoxPadding
	if boxPad == 0 {
		boxPad = *overlay.FontSize * 0.45
	}
	return boxPad
}

// drawTextBlock renders centered lines with text shadows and emoji images aligned to the text baseline.
func drawTextBlock(ctx *gg.Context, lines []TextLine, centerX, topY, lineHeight, fontSize float64) {
	for i, line := range lines {
		x := centerX - line.Width/2
		baseline := topY + float64(i)*lineHeight
		previousWasEmoji := false

		for _, part := range line.Parts {
			if previousWasEmoji && part.Emoji != nil {
				x += emojiGap(fontSize)
			}

			if part.Emoji != nil {
				drawEmoji(ctx, part.Emoji, x, baseline-fontSize*0.82, fontSize)
			} else {
				ctx.SetColor(color.RGBA{R: 0, G: 0, B: 0, A: 170})
				ctx.DrawString(part.Value, x+3, baseline+3)
				ctx.SetColor(color.White)
				ctx.DrawString(part.Value, x, baseline)
			}

			x += part.Width
			previousWasEmoji = part.Emoji != nil
		}
	}
}

// drawEmoji paints one emoji image scaled to a square box at x/y.
func drawEmoji(ctx *gg.Context, img image.Image, x, y, size float64) {
	ctx.Push()
	ctx.Translate(x, y)
	ctx.Scale(size/float64(img.Bounds().Dx()), size/float64(img.Bounds().Dy()))
	ctx.DrawImage(img, 0, 0)
	ctx.Pop()
}
