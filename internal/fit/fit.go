package fit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lucasepe/cliphub/internal/shared"
)

// defaultOutputPath returns the default fitted video path beside the input video.
func defaultOutputPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_fit", "")
}

// validateConfig checks required paths and target frame dimensions.
func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}

	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return errors.New("-width and -height must be positive")
	}

	return nil
}

// fitVideo invokes ffmpeg to scale the input video into the requested frame.
func fitVideo(cfg Config) error {
	args := ffmpegArgs(cfg)
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", args)
		return nil
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fit %q: %w", cfg.Output, err)
	}

	return nil
}

// ffmpegArgs builds the ffmpeg command arguments for fitting one video.
func ffmpegArgs(cfg Config) []string {
	return []string{
		"-y",
		"-i", cfg.Input,
		"-vf", videoFilter(cfg),
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "copy",
		"-movflags", "+faststart",
		cfg.Output,
	}
}

// videoFilter returns a contain or cover scaling filter for the requested frame.
func videoFilter(cfg Config) string {
	if cfg.Cover {
		return fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:(iw-ow)/2:(ih-oh)/2",
			cfg.Width,
			cfg.Height,
			cfg.Width,
			cfg.Height,
		)
	}
	return fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
		cfg.Width,
		cfg.Height,
		cfg.Width,
		cfg.Height,
	)
}
