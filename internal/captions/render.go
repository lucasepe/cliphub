package captions

import (
	"fmt"

	"github.com/lucasepe/x/image/gg"
	imageio "github.com/lucasepe/x/image/io"
)

// renderTextPNG draws wrapped text onto a transparent image saved at outputPath.
func renderTextPNG(cfg Config, overlay Overlay, outputPath string) error {
	ctx := gg.NewContext(cfg.Width, cfg.Height)
	ctx.SetRGBA(0, 0, 0, 0)
	ctx.Clear()

	fontPath := cfg.FontPath
	fontSize := *overlay.FontSize
	face, err := loadFontFace(fontPath, fontSize)
	if err != nil {
		return fmt.Errorf("load font %q: %w", fontPath, err)
	}
	ctx.SetFontFace(face)

	lines, err := layoutLines(ctx, cfg, overlay)
	if err != nil {
		return err
	}
	lineHeight := fontSize * 1.25
	blockHeight := lineHeight * float64(len(lines))
	y := textY(overlay.Gravity, float64(cfg.Height), blockHeight, overlay)

	if *overlay.Box {
		drawTextBackground(ctx, lines, float64(cfg.Width)/2, y, lineHeight, cfg, overlay)
	}
	drawTextBlock(ctx, lines, float64(cfg.Width)/2, y, lineHeight, fontSize)

	if err := imageio.WriteToFile(ctx.Image(), outputPath, imageio.PNG); err != nil {
		return fmt.Errorf("save overlay PNG: %w", err)
	}
	return nil
}

// textY calculates the first baseline Y coordinate for the requested gravity.
func textY(gravity string, imageHeight, blockHeight float64, overlay Overlay) float64 {
	fontSize := *overlay.FontSize
	padding := *overlay.Padding
	topInset := fontSize * 0.85
	if *overlay.Box {
		topInset += effectiveBoxPadding(overlay)
	}

	switch gravity {
	case "top":
		return padding + topInset
	case "center":
		return (imageHeight - blockHeight) / 2
	case "bottom":
		return imageHeight - blockHeight - padding
	default:
		return padding
	}
}
