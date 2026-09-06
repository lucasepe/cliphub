package captions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// processVideo creates a temporary text overlay image and invokes ffmpeg to produce the final video.
func processVideo(cfg Config) error {
	overlays, err := loadOverlays(cfg)
	if err != nil {
		return err
	}
	if cfg.DryRun {
		return printRenderDryRun(cfg, overlays)
	}

	jobs, cleanup, err := createOverlayImages(cfg, overlays)
	if err != nil {
		return err
	}
	defer cleanup()

	return runFFmpeg(cfg, jobs)
}

// printRenderDryRun prints the ffmpeg command using placeholder overlay PNG paths.
func printRenderDryRun(cfg Config, overlays []Overlay) error {
	jobs := make([]RenderJob, 0, len(overlays))
	for i, overlay := range overlays {
		jobs = append(jobs, RenderJob{
			Overlay: overlay,
			Path:    fmt.Sprintf("<overlay_%03d.png>", i+1),
		})
	}
	printRenderCommands(cfg, jobs)
	return nil
}

// loadOverlays reads timed overlays from JSON, returning none when the file is absent.
func loadOverlays(cfg Config) ([]Overlay, error) {
	overlays, found, err := readOverlayFile(cfg.Overlays)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return applyOverlayDefaults(overlays)
}

// readOverlayFile returns overlays from path and reports whether the file existed.
func readOverlayFile(path string) ([]Overlay, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read overlays file %q: %w", path, err)
	}

	var overlays []Overlay
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&overlays); err != nil {
		if field := unknownJSONField(err); field != "" {
			return nil, false, fmt.Errorf("parse overlays file %q: unknown property %q", path, field)
		}
		return nil, false, fmt.Errorf("parse overlays file %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, false, fmt.Errorf("parse overlays file %q: unexpected content after overlay list", path)
	}
	return overlays, true, nil
}

func unknownJSONField(err error) string {
	const prefix = "json: unknown field \""
	message := err.Error()
	if !strings.HasPrefix(message, prefix) || !strings.HasSuffix(message, "\"") {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(message, prefix), "\"")
}

// applyOverlayDefaults fills omitted overlay fields from built-in defaults and validates each overlay.
func applyOverlayDefaults(overlays []Overlay) ([]Overlay, error) {
	for i := range overlays {
		if overlays[i].Gravity == "" {
			overlays[i].Gravity = "bottom"
		}
		if overlays[i].Box == nil {
			defaultBox := false
			overlays[i].Box = &defaultBox
		}
		if overlays[i].BoxAlpha == nil {
			defaultAlpha := defaultBoxAlpha
			overlays[i].BoxAlpha = &defaultAlpha
		}
		if overlays[i].BoxPadding == nil {
			defaultBoxPadding := 0.0
			overlays[i].BoxPadding = &defaultBoxPadding
		}
		if overlays[i].BoxRadius == nil {
			defaultBoxRadius := 0.0
			overlays[i].BoxRadius = &defaultBoxRadius
		}
		if overlays[i].FontSize == nil {
			fontSize := float64(defaultFontSize)
			overlays[i].FontSize = &fontSize
		}
		applyPaddingDefaults(&overlays[i])
		if overlays[i].MaxChars == nil {
			maxChars := defaultMaxChars
			overlays[i].MaxChars = &maxChars
		}
		if err := validateOverlay(overlays[i], i); err != nil {
			return nil, err
		}
	}
	return overlays, nil
}

func applyPaddingDefaults(overlay *Overlay) {
	top, bottom := float64(defaultPadding), float64(defaultPadding)
	left, right := float64(defaultPadding), float64(defaultPadding)
	if overlay.SafeArea == "instagram-reel" {
		top, bottom = instagramTop, instagramBottom
	}
	if overlay.PaddingTop == nil {
		overlay.PaddingTop = &top
	}
	if overlay.PaddingBottom == nil {
		overlay.PaddingBottom = &bottom
	}
	if overlay.PaddingLeft == nil {
		overlay.PaddingLeft = &left
	}
	if overlay.PaddingRight == nil {
		overlay.PaddingRight = &right
	}
}

// validateOverlay checks required text, timing, gravity, and optional styling values for one overlay.
func validateOverlay(overlay Overlay, index int) error {
	prefix := fmt.Sprintf("overlay %d", index)
	if overlay.Text == "" {
		return fmt.Errorf("%s: missing text", prefix)
	}
	if overlay.Start < 0 {
		return fmt.Errorf("%s: start must be zero or greater", prefix)
	}
	if overlay.End <= overlay.Start {
		return fmt.Errorf("%s: end must be greater than start", prefix)
	}
	switch overlay.Gravity {
	case "top", "center", "bottom":
	default:
		return fmt.Errorf("%s: gravity must be one of: top, center, bottom", prefix)
	}
	if overlay.SafeArea != "" && overlay.SafeArea != "instagram-reel" {
		return fmt.Errorf("%s: safe_area must be one of: instagram-reel", prefix)
	}
	if overlay.BoxAlpha != nil && (*overlay.BoxAlpha < 0 || *overlay.BoxAlpha > 1) {
		return fmt.Errorf("%s: box_alpha must be between 0 and 1", prefix)
	}
	if overlay.BoxPadding != nil && *overlay.BoxPadding < 0 {
		return fmt.Errorf("%s: box_padding must be zero or greater", prefix)
	}
	if overlay.BoxRadius != nil && *overlay.BoxRadius < 0 {
		return fmt.Errorf("%s: box_radius must be zero or greater", prefix)
	}
	if overlay.FontSize != nil && *overlay.FontSize <= 0 {
		return fmt.Errorf("%s: font_size must be positive", prefix)
	}
	for name, value := range map[string]*float64{
		"padding_top": overlay.PaddingTop, "padding_bottom": overlay.PaddingBottom,
		"padding_left": overlay.PaddingLeft, "padding_right": overlay.PaddingRight,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s: %s must be zero or greater", prefix, name)
		}
	}
	if overlay.MaxChars != nil && *overlay.MaxChars <= 0 {
		return fmt.Errorf("%s: max_chars must be positive", prefix)
	}
	return nil
}

// createOverlayImages renders every configured overlay into a temporary PNG used as an ffmpeg input.
func createOverlayImages(cfg Config, overlays []Overlay) ([]RenderJob, func(), error) {
	var jobs []RenderJob
	var paths []string
	cleanup := func() {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}

	for _, overlay := range overlays {
		path, err := createTempOverlay(cfg, overlay)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		paths = append(paths, path)
		jobs = append(jobs, RenderJob{Overlay: overlay, Path: path})
	}
	return jobs, cleanup, nil
}

// createTempOverlay renders one configured overlay into a transparent PNG and returns its path.
func createTempOverlay(cfg Config, overlay Overlay) (string, error) {
	file, err := os.CreateTemp("", "cliphub-overlay-*.png")
	if err != nil {
		return "", fmt.Errorf("create temp overlay: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close temp overlay: %w", err)
	}

	if err := renderTextPNG(cfg, overlay, path); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}
