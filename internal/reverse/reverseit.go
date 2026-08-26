package reverse

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lucasepe/cliphub/internal/shared"
)

// defaultReversePath returns the default reversed video path beside the input video.
func defaultReversePath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_reverse", "")
}

// validateReverseConfig checks the required input and output paths.
func validateReverseConfig(cfg ReverseConfig) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	return nil
}

// reverseVideo invokes ffmpeg with reverse filters for video and, when requested, audio.
func reverseVideo(cfg ReverseConfig) error {
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", reverseFFmpegArgs(cfg))
		return nil
	}
	cmd := exec.Command("ffmpeg", reverseFFmpegArgs(cfg)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reverse %q: %w", cfg.Output, err)
	}
	return nil
}

// reverseFFmpegArgs builds the ffmpeg command arguments for reversing one video.
func reverseFFmpegArgs(cfg ReverseConfig) []string {
	args := []string{
		"-y",
		"-i", cfg.Input,
		"-filter_complex", reverseFilter(cfg),
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

// reverseFilter returns the ffmpeg filtergraph for video-only or video-plus-audio reversal.
func reverseFilter(cfg ReverseConfig) string {
	if cfg.Audio {
		return "[0:v:0]reverse[vout];[0:a:0?]areverse[aout]"
	}
	return "[0:v:0]reverse[vout]"
}
