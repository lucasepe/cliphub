package soundfx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

func defaultOutputPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_soundfx", "")
}

func defaultSoundsPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_soundfx", ".json")
}

func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	if cfg.Sounds == "" && cfg.Audio == "" {
		return errors.New("provide -sounds JSON or -audio with -at")
	}
	if cfg.Sounds != "" && cfg.Audio != "" {
		return errors.New("use either -sounds or -audio, not both")
	}
	if cfg.Audio != "" && strings.TrimSpace(cfg.At) == "" {
		return errors.New("-at is required with -audio")
	}
	if cfg.Volume < 0 {
		return errors.New("-volume must be zero or greater")
	}
	if cfg.VideoVolume < 0 {
		return errors.New("-video-volume must be zero or greater")
	}
	return nil
}

func addSoundEffects(cfg Config) error {
	effects, err := loadEffects(cfg)
	if err != nil {
		return err
	}
	cfg.effects = effects
	args := ffmpegArgs(cfg)
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", args)
		return nil
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("soundfx %q: %w", cfg.Output, err)
	}
	return nil
}

func loadEffects(cfg Config) ([]effect, error) {
	if cfg.Sounds != "" {
		return readItems(cfg.Sounds)
	}
	at, err := shared.ParseClockTime(cfg.At)
	if err != nil {
		return nil, fmt.Errorf("-at: %w", err)
	}
	return []effect{{Audio: cfg.Audio, At: at, Volume: cfg.Volume}}, nil
}

func readItems(path string) ([]effect, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sound effects %q: %w", path, err)
	}
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse sound effects %q: %w", path, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("sound effects file %q contains no effects", path)
	}

	baseDir := filepath.Dir(path)
	effects := make([]effect, 0, len(items))
	for i, item := range items {
		item.Audio = strings.TrimSpace(item.Audio)
		if item.Audio == "" {
			return nil, fmt.Errorf("sound effect %d audio is required", i+1)
		}
		if !filepath.IsAbs(item.Audio) {
			item.Audio = filepath.Join(baseDir, item.Audio)
		}
		at, err := shared.ParseClockTime(item.At)
		if err != nil {
			return nil, fmt.Errorf("sound effect %d at: %w", i+1, err)
		}
		if item.Volume < 0 {
			return nil, fmt.Errorf("sound effect %d volume must be zero or greater", i+1)
		}
		if item.Volume == 0 {
			item.Volume = 1
		}
		effects = append(effects, effect{Audio: item.Audio, At: at, Volume: item.Volume})
	}
	return effects, nil
}

func ffmpegArgs(cfg Config) []string {
	args := []string{"-y", "-i", cfg.Input}
	for _, item := range cfg.effects {
		args = append(args, "-i", item.Audio)
	}
	args = append(args,
		"-filter_complex", audioFilter(cfg),
		"-map", "0:v:0",
		"-map", "[aout]",
		"-c:v", "copy",
		"-c:a", "aac",
		"-movflags", "+faststart",
		cfg.Output,
	)
	return args
}

func audioFilter(cfg Config) string {
	parts := []string{fmt.Sprintf("[0:a:0]volume=%s[base]", formatFloat(cfg.VideoVolume))}
	inputs := strings.Builder{}
	inputs.WriteString("[base]")
	for i, item := range cfg.effects {
		label := fmt.Sprintf("sfx%d", i)
		delay := strconv.FormatInt(int64(math.Round(item.At*1000)), 10)
		parts = append(parts, fmt.Sprintf("[%d:a:0]volume=%s,adelay=%s:all=1[%s]", i+1, formatFloat(item.Volume), delay, label))
		inputs.WriteString("[" + label + "]")
	}
	parts = append(parts, fmt.Sprintf("%samix=inputs=%d:duration=first:dropout_transition=0:normalize=0[aout]", inputs.String(), len(cfg.effects)+1))
	return strings.Join(parts, ";")
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
