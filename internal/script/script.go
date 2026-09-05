package script

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/lucasepe/cliphub/internal/captions"
)

func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in text path")
	}

	if cfg.Output == "" {
		return errors.New("missing required -out overlays path")
	}

	if cfg.Duration <= 0 {
		return errors.New("-duration must be positive")
	}

	return nil
}

// generate reads a plain-text script and distributes its items over the requested duration.
func generate(cfg Config) ([]captions.Overlay, int, error) {
	data, err := os.ReadFile(cfg.Input)
	if err != nil {
		return nil, 0, fmt.Errorf("read text file %q: %w", cfg.Input, err)
	}

	items, wordCount := textItems(string(data), cfg.Words)
	if len(items) == 0 {
		return nil, 0, errors.New("input text contains no words")
	}

	return timedOverlays(items, cfg.Duration), wordCount, nil
}

// textItems uses non-empty lines as caption blocks, or individual words when requested.
func textItems(text string, words bool) ([]string, int) {
	fields := strings.Fields(text)
	if words {
		return fields, len(fields)
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	items := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			items = append(items, line)
		}
	}

	return items, len(fields)
}

// timedOverlays allocates more screen time to longer text and punctuation pauses.
func timedOverlays(items []string, duration float64) []captions.Overlay {
	weights := make([]float64, len(items))
	var totalWeight float64

	for i, item := range items {
		weights[i] = readingWeight(item)
		totalWeight += weights[i]
	}

	fontSize := float64(defaultFontSize)
	overlays := make([]captions.Overlay, 0, len(items))

	start := 0.0
	for i, item := range items {
		end := start + duration*weights[i]/totalWeight
		if i == len(items)-1 {
			end = duration
		}

		overlays = append(overlays, captions.Overlay{
			Text:     item,
			Gravity:  defaultGravity,
			Start:    start,
			End:      end,
			FontSize: &fontSize,
		})
		start = end
	}

	return overlays
}

func readingWeight(text string) float64 {
	var weight float64
	for _, word := range strings.Fields(text) {
		// Five characters approximate one average word; short words still get a full beat.
		weight += math.Max(1,
			float64(utf8.RuneCountInString(strings.Trim(word, ".,;:!?")))/5)

		if strings.HasSuffix(word, ".") ||
			strings.HasSuffix(word, "!") ||
			strings.HasSuffix(word, "?") {
			weight += 0.5
		} else if strings.HasSuffix(word, ",") ||
			strings.HasSuffix(word, ";") ||
			strings.HasSuffix(word, ":") {
			weight += 0.25
		}
	}

	return math.Max(1, weight)
}

func writeOverlays(path string, overlays []captions.Overlay) error {
	data, err := json.MarshalIndent(overlays, "", "  ")
	if err != nil {
		return fmt.Errorf("encode overlays: %w", err)
	}

	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write overlays %q: %w", path, err)
	}

	return nil
}

func wordsPerMinute(words int, duration float64) float64 {
	return float64(words) * 60 / duration
}
