package transcribe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var whisperSpecialTokenPattern = regexp.MustCompile(`\[_[^\]]+\]`)

// validateTranscribeConfig checks required paths and style defaults for speech-to-overlay generation.
func validateTranscribeConfig(cfg TranscribeConfig) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out overlays path")
	}
	if cfg.Model == "" {
		return errors.New("missing required -model whisper model path")
	}
	switch cfg.Gravity {
	case "top", "center", "bottom":
	default:
		return errors.New("-gravity must be one of: top, center, bottom")
	}
	if cfg.FontSize <= 0 {
		return errors.New("-font-size must be positive")
	}
	if cfg.PaddingTop < 0 || cfg.PaddingBottom < 0 || cfg.PaddingLeft < 0 || cfg.PaddingRight < 0 {
		return errors.New("directional padding values must be zero or greater")
	}
	if cfg.MaxChars <= 0 {
		return errors.New("-max-chars must be positive")
	}
	if cfg.BoxAlpha < 0 || cfg.BoxAlpha > 1 {
		return errors.New("-box-alpha must be between 0 and 1")
	}
	if cfg.BoxPadding < 0 {
		return errors.New("-box-padding must be zero or greater")
	}
	if cfg.BoxRadius < 0 {
		return errors.New("-box-radius must be zero or greater")
	}
	return nil
}

// transcribeVideo extracts audio, runs whisper.cpp, converts segments, and writes overlays JSON.
func transcribeVideo(cfg TranscribeConfig) error {
	tempDir, err := os.MkdirTemp("", "cliphub-transcribe-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	audioPath := filepath.Join(tempDir, "audio.wav")
	if err := extractAudio(cfg.Input, audioPath); err != nil {
		return err
	}

	whisperPrefix := filepath.Join(tempDir, "whisper")
	whisperJSON := whisperPrefix + ".json"
	if err := runWhisper(cfg, audioPath, whisperPrefix); err != nil {
		return err
	}

	segments, err := readWhisperSegments(whisperJSON)
	if err != nil {
		return err
	}
	overlays, err := whisperSegmentsToOverlays(segments, cfg)
	if err != nil {
		return err
	}
	return writeOverlays(cfg.Output, overlays)
}

// extractAudio converts the input video audio stream into the mono 16 kHz WAV format expected by Whisper.
func extractAudio(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-vn", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract audio: %w", err)
	}
	return nil
}

// runWhisper invokes whisper.cpp and asks it to write JSON output at outputPrefix+".json".
func runWhisper(cfg TranscribeConfig, audioPath, outputPrefix string) error {
	jsonFlag := "-oj"
	if cfg.Words {
		jsonFlag = "-ojf"
	}
	args := []string{"-m", cfg.Model, "-f", audioPath, "-l", cfg.Language, jsonFlag, "-of", outputPrefix}
	if cfg.NoGPU {
		args = append(args, "--no-gpu")
	}
	cmd := exec.Command(cfg.Whisper, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run whisper: %w", err)
	}
	return nil
}

// readWhisperSegments parses the JSON file produced by whisper.cpp.
func readWhisperSegments(path string) ([]WhisperSegment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read whisper JSON %q: %w", path, err)
	}
	var output WhisperOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse whisper JSON %q: %w", path, err)
	}
	return output.Transcription, nil
}

// whisperSegmentsToOverlays converts Whisper segments into timed render overlays with shared styling.
func whisperSegmentsToOverlays(segments []WhisperSegment, cfg TranscribeConfig) ([]Overlay, error) {
	var overlays []Overlay
	for i, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		start, err := parseWhisperTime(segment.Timestamps.From)
		if err != nil {
			return nil, fmt.Errorf("segment %d start: %w", i, err)
		}
		end, err := parseWhisperTime(segment.Timestamps.To)
		if err != nil {
			return nil, fmt.Errorf("segment %d end: %w", i, err)
		}
		if cfg.Words {
			wordItems := tokenWordOverlays(segment.Tokens, cfg)
			if len(wordItems) == 0 {
				wordItems = estimatedWordOverlays(text, start, end, cfg)
			}
			overlays = append(overlays, wordItems...)
			continue
		}
		overlays = append(overlays, captionOverlay(text, start, end, cfg))
	}
	return overlays, nil
}

// captionOverlay builds one styled overlay for the provided text and time range.
func captionOverlay(text string, start, end float64, cfg TranscribeConfig) Overlay {
	return Overlay{
		Text:          text,
		Gravity:       cfg.Gravity,
		Start:         start,
		End:           end,
		Box:           &cfg.Box,
		BoxAlpha:      &cfg.BoxAlpha,
		BoxPadding:    &cfg.BoxPadding,
		BoxRadius:     &cfg.BoxRadius,
		FontSize:      &cfg.FontSize,
		PaddingTop:    &cfg.PaddingTop,
		PaddingBottom: &cfg.PaddingBottom,
		PaddingLeft:   &cfg.PaddingLeft,
		PaddingRight:  &cfg.PaddingRight,
		MaxChars:      &cfg.MaxChars,
	}
}

// tokenWordOverlays joins Whisper tokens into words while preserving token-derived timestamps.
func tokenWordOverlays(tokens []WhisperToken, cfg TranscribeConfig) []Overlay {
	type wordRange struct {
		text  string
		start float64
		end   float64
	}

	var words []wordRange
	current := wordRange{}
	for _, token := range tokens {
		if isSpecialWhisperToken(token.Text) {
			continue
		}
		text := cleanWhisperTokenText(token.Text)
		if text == "" {
			continue
		}
		start := float64(token.Offsets.From) / 1000
		end := float64(token.Offsets.To) / 1000
		if strings.HasPrefix(token.Text, " ") && current.text != "" {
			words = append(words, current)
			current = wordRange{}
		}
		if current.text == "" {
			current.text = text
			current.start = start
			current.end = end
			continue
		}
		current.text += text
		if end > current.end {
			current.end = end
		}
	}
	if current.text != "" {
		words = append(words, current)
	}

	overlays := make([]Overlay, 0, len(words))
	for i, word := range words {
		word.text = cleanCaptionWord(word.text)
		if word.text == "" {
			continue
		}
		if i < len(words)-1 && word.end > words[i+1].start {
			word.end = words[i+1].start
		}
		if word.end <= word.start && i < len(words)-1 && words[i+1].start > word.start {
			word.end = words[i+1].start
		}
		if word.end <= word.start {
			word.end = word.start + 0.2
		}
		overlays = append(overlays, captionOverlay(word.text, word.start, word.end, cfg))
	}
	return overlays
}

// isSpecialWhisperToken reports whether token text is metadata rather than spoken text.
func isSpecialWhisperToken(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "[_") && strings.HasSuffix(text, "_]")
}

// cleanWhisperTokenText removes Whisper metadata markers that can appear next to spoken text.
func cleanWhisperTokenText(text string) string {
	text = whisperSpecialTokenPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// cleanCaptionWord removes punctuation around one word while preserving inner letters and apostrophes.
func cleanCaptionWord(word string) string {
	return strings.TrimFunc(word, unicode.IsPunct)
}

// estimatedWordOverlays splits one Whisper segment into one overlay per word using equal time slices.
func estimatedWordOverlays(text string, start, end float64, cfg TranscribeConfig) []Overlay {
	words := strings.Fields(text)
	if len(words) == 0 || end <= start {
		return nil
	}

	step := (end - start) / float64(len(words))
	overlays := make([]Overlay, 0, len(words))
	for i, word := range words {
		word = cleanCaptionWord(word)
		if word == "" {
			continue
		}
		wordStart := start + step*float64(i)
		wordEnd := wordStart + step
		if i == len(words)-1 {
			wordEnd = end
		}
		overlays = append(overlays, captionOverlay(word, wordStart, wordEnd, cfg))
	}
	return overlays
}

// parseWhisperTime converts Whisper timestamps such as "00:01:02.340" or "00:01:02,340" into seconds.
func parseWhisperTime(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty timestamp")
	}
	value = strings.ReplaceAll(value, ",", ".")
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		return seconds, nil
	}

	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timestamp %q", value)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}
	duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
	return duration.Seconds() + seconds, nil
}

// writeOverlays writes generated overlays as indented JSON.
func writeOverlays(path string, overlays []Overlay) error {
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
