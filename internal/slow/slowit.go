package slow

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

// defaultSlowPath returns the default slowed video path beside the input video.
func defaultSlowPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_slow", "")
}

// validateSlowConfig checks required paths and the slowdown factor.
func validateSlowConfig(cfg SlowConfig) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	if cfg.Factor <= 1 {
		return errors.New("-factor must be greater than 1")
	}
	return nil
}

// slowVideo invokes ffmpeg with setpts and optional atempo filters.
func slowVideo(cfg SlowConfig) error {
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", slowFFmpegArgs(cfg))
		return nil
	}
	cmd := exec.Command("ffmpeg", slowFFmpegArgs(cfg)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("slow %q: %w", cfg.Output, err)
	}
	return nil
}

// slowFFmpegArgs builds the ffmpeg command arguments for slowing one video.
func slowFFmpegArgs(cfg SlowConfig) []string {
	args := []string{
		"-y",
		"-i", cfg.Input,
		"-filter_complex", slowFilter(cfg),
		"-map", "[vout]",
	}
	if cfg.Audio {
		args = append(args, "-map", "[aout]")
	}
	args = append(args,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
	)
	if cfg.Audio {
		args = append(args, "-c:a", "aac")
	} else {
		args = append(args, "-an")
	}
	args = append(args, cfg.Output)
	return args
}

// slowFilter returns the ffmpeg filtergraph for video-only or video-plus-audio slow motion.
func slowFilter(cfg SlowConfig) string {
	videoFilter := fmt.Sprintf("[0:v:0]setpts=%s*PTS[vout]", formatFloat(cfg.Factor))
	if !cfg.Audio {
		return videoFilter
	}
	return videoFilter + ";[0:a:0?]" + atempoFilter(1/cfg.Factor) + "[aout]"
}

// atempoFilter builds an ffmpeg atempo chain whose factors are each inside ffmpeg's supported range.
func atempoFilter(tempo float64) string {
	var parts []string
	for tempo < 0.5 {
		parts = append(parts, "atempo=0.5")
		tempo /= 0.5
	}
	for tempo > 2 {
		parts = append(parts, "atempo=2")
		tempo /= 2
	}
	parts = append(parts, "atempo="+formatFloat(tempo))
	return strings.Join(parts, ",")
}

// formatFloat writes a compact decimal number suitable for ffmpeg filter expressions.
func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
