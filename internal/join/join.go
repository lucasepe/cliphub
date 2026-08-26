package join

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

// defaultOutputPath returns the default joined video path beside the join plan.
func defaultOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	name = strings.TrimSuffix(name, "_join")
	return filepath.Join(dir, name+"_joined.mp4")
}

// validateConfig checks required paths before work begins.
func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in join JSON path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	return nil
}

// joinVideo loads a join plan and invokes ffmpeg to concatenate clips.
func joinVideo(cfg Config) error {
	items, err := readItems(cfg.Input)
	if err != nil {
		return err
	}
	if cfg.Audio && hasFade(items) {
		return errors.New("-audio cannot be used with JSON fades yet")
	}
	args, err := ffmpegArgs(cfg, items)
	if err != nil {
		return err
	}
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", args)
		return nil
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("join %q: %w", cfg.Output, err)
	}
	return nil
}

// readItems decodes the JSON join plan and resolves relative files beside the plan.
func readItems(path string) ([]Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read join plan %q: %w", path, err)
	}
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse join plan %q: %w", path, err)
	}
	if len(items) < 2 {
		return nil, errors.New("join plan must contain at least two clips")
	}
	baseDir := filepath.Dir(path)
	for i := range items {
		items[i].File = strings.TrimSpace(items[i].File)
		if items[i].File == "" {
			return nil, fmt.Errorf("clip %d file is required", i+1)
		}
		if items[i].Fade < 0 {
			return nil, fmt.Errorf("clip %d fade must be zero or greater", i+1)
		}
		if !filepath.IsAbs(items[i].File) {
			items[i].File = filepath.Join(baseDir, items[i].File)
		}
	}
	return items, nil
}

// ffmpegArgs builds a concat or xfade command for the provided clips.
func ffmpegArgs(cfg Config, items []Item) ([]string, error) {
	args := []string{"-y"}
	for _, item := range items {
		args = append(args, "-i", item.File)
	}
	filter, err := filterGraph(cfg, items)
	if err != nil {
		return nil, err
	}
	args = append(args, "-filter_complex", filter, "-map", "[vout]")
	if cfg.Audio {
		args = append(args, "-map", "[aout]", "-c:a", "aac")
	} else {
		args = append(args, "-an")
	}
	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", cfg.Output)
	return args, nil
}

// filterGraph returns the ffmpeg filtergraph for either plain concat or video fades.
func filterGraph(cfg Config, items []Item) (string, error) {
	if hasFade(items) {
		durations, err := probeDurations(items)
		if err != nil {
			return "", err
		}
		return fadeFilter(items, durations), nil
	}
	return concatFilter(len(items), cfg.Audio), nil
}

// concatFilter returns a simple concat filter for clips with matching media properties.
func concatFilter(count int, audio bool) string {
	var videoInputs strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&videoInputs, "[%d:v:0]", i)
	}
	video := fmt.Sprintf("%sconcat=n=%d:v=1:a=0[vout]", videoInputs.String(), count)
	if !audio {
		return video
	}
	var audioInputs strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&audioInputs, "[%d:a:0]", i)
	}
	audioFilter := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[aout]", audioInputs.String(), count)
	return video + ";" + audioFilter
}

// fadeFilter chains xfade filters using each item's fade duration toward the following clip.
func fadeFilter(items []Item, durations []float64) string {
	parts := make([]string, 0, len(items)*2)
	for i := range items {
		parts = append(parts, fmt.Sprintf("[%d:v:0]setpts=PTS-STARTPTS,format=yuv420p[v%d]", i, i))
	}
	input := "v0"
	elapsed := durations[0]
	for i := 1; i < len(items); i++ {
		fade := items[i-1].Fade
		if fade <= 0 {
			fade = 0.001
		}
		offset := elapsed - fade
		output := "vout"
		if i < len(items)-1 {
			output = fmt.Sprintf("x%d", i)
		}
		parts = append(parts, fmt.Sprintf("[%s][v%d]xfade=transition=fade:duration=%.3f:offset=%.3f[%s]", input, i, fade, offset, output))
		input = output
		elapsed += durations[i] - fade
	}
	return strings.Join(parts, ";")
}

func hasFade(items []Item) bool {
	for _, item := range items {
		if item.Fade > 0 {
			return true
		}
	}
	return false
}

// probeDurations reads clip durations with ffprobe; xfade needs them to compute offsets.
func probeDurations(items []Item) ([]float64, error) {
	durations := make([]float64, 0, len(items))
	for i, item := range items {
		duration, err := probeDuration(item.File)
		if err != nil {
			return nil, fmt.Errorf("clip %d duration: %w", i+1, err)
		}
		if i < len(items)-1 && item.Fade >= duration {
			return nil, fmt.Errorf("clip %d fade must be shorter than clip duration", i+1)
		}
		durations = append(durations, duration)
	}
	return durations, nil
}

// probeDuration returns the media duration in seconds using ffprobe.
func probeDuration(path string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nk=1:nw=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("run ffprobe: %w", err)
	}
	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &duration); err != nil {
		return 0, fmt.Errorf("parse ffprobe duration: %w", err)
	}
	if duration <= 0 {
		return 0, errors.New("duration must be positive")
	}
	return duration, nil
}
