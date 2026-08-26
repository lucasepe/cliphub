package soundtrack

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lucasepe/cliphub/internal/shared"
)

// defaultOutputPath returns the default soundtracked video path beside the input video.
func defaultOutputPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_soundtrack", "")
}

// validateConfig checks paths and volume ranges before invoking ffmpeg.
func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	if cfg.Audio == "" {
		return errors.New("missing required -audio path")
	}
	if cfg.MusicVolume < 0 {
		return errors.New("-music-volume must be zero or greater")
	}
	if cfg.VideoVolume < 0 {
		return errors.New("-video-volume must be zero or greater")
	}
	return nil
}

// addSoundtrack invokes ffmpeg to mix or replace the original video audio.
func addSoundtrack(cfg Config) error {
	args := ffmpegArgs(cfg)
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", args)
		return nil
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("soundtrack %q: %w", cfg.Output, err)
	}
	return nil
}

// ffmpegArgs builds a command that copies video and writes one mixed AAC audio stream.
func ffmpegArgs(cfg Config) []string {
	args := []string{"-y", "-i", cfg.Input}
	if cfg.Loop {
		args = append(args, "-stream_loop", "-1")
	}
	args = append(args, "-i", cfg.Audio)
	args = append(args, "-filter_complex", audioFilter(cfg), "-map", "0:v:0", "-map", "[aout]", "-c:v", "copy", "-c:a", "aac", "-shortest", "-movflags", "+faststart", cfg.Output)
	return args
}

// audioFilter mixes the original audio with the external track, or uses only the external track.
func audioFilter(cfg Config) string {
	music := fmt.Sprintf("[1:a:0]volume=%g[music]", cfg.MusicVolume)
	if cfg.VideoVolume == 0 {
		return music + ";[music]anull[aout]"
	}
	video := fmt.Sprintf("[0:a:0]volume=%g[video]", cfg.VideoVolume)
	return video + ";" + music + ";[video][music]amix=inputs=2:duration=shortest:dropout_transition=0[aout]"
}
