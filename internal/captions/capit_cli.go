package captions

import (
	"errors"
	"fmt"

	"github.com/lucasepe/cliphub/internal/shared"
)

// DefaultOverlaysPath returns the default overlay JSON path beside the input video.
func DefaultOverlaysPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_overlays", ".json")
}

// defaultCaptionedPath returns the default rendered video path beside the input video.
func defaultCaptionedPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_captioned", "")
}

// validateConfig checks required paths, timing, dimensions, and supported gravity values before work begins.
func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return errors.New("-width and -height must be positive")
	}
	if isKnownColorEmojiFont(cfg.FontPath) {
		return fmt.Errorf("%q is a color emoji font, but the text renderer cannot draw color emoji glyphs; use a normal text font for now", cfg.FontPath)
	}
	return nil
}
