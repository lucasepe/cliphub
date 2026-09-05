package captions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

// runFFmpeg scales the source into the target canvas and applies every timed PNG overlay.
func runFFmpeg(cfg Config, jobs []RenderJob) error {
	if len(jobs) <= maxOverlayInputs {
		return runFFmpegPass(cfg, jobs)
	}

	input := cfg.Input
	var tempOutputs []string
	defer func() {
		for _, path := range tempOutputs {
			_ = os.Remove(path)
		}
	}()

	for start := 0; start < len(jobs); start += maxOverlayInputs {
		end := start + maxOverlayInputs
		if end > len(jobs) {
			end = len(jobs)
		}

		output := cfg.Output
		if end < len(jobs) {
			file, err := os.CreateTemp("", "cliphub-pass-*.mp4")
			if err != nil {
				return fmt.Errorf("create intermediate video: %w", err)
			}
			output = file.Name()
			if err := file.Close(); err != nil {
				return fmt.Errorf("close intermediate video %q: %w", output, err)
			}
			tempOutputs = append(tempOutputs, output)
		}

		passCfg := cfg
		passCfg.Input = input
		passCfg.Output = output
		if err := runFFmpegPass(passCfg, jobs[start:end]); err != nil {
			return err
		}
		input = output
	}
	return nil
}

// runFFmpegPass applies a bounded number of overlay inputs in one ffmpeg invocation.
func runFFmpegPass(cfg Config, jobs []RenderJob) error {
	args := renderFFmpegArgs(cfg, jobs)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run ffmpeg: %w", err)
	}
	return nil
}

// renderFFmpegArgs builds one ffmpeg command that applies a bounded number of overlays.
func renderFFmpegArgs(cfg Config, jobs []RenderJob) []string {
	args := []string{"-y", "-i", cfg.Input}
	for _, job := range jobs {
		args = append(args, "-i", job.Path)
	}
	args = append(args, "-filter_complex", ffmpegFilter(cfg, jobs))
	args = append(args,
		"-map", "[vout]",
		"-map", "0:a?",
		"-c:a", "copy",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		cfg.Output,
	)
	return args
}

// printRenderCommands prints the render command or commands that would be executed.
func printRenderCommands(cfg Config, jobs []RenderJob) {
	if len(jobs) <= maxOverlayInputs {
		shared.PrintCommand("ffmpeg", renderFFmpegArgs(cfg, jobs))
		return
	}

	input := cfg.Input
	for start := 0; start < len(jobs); start += maxOverlayInputs {
		end := start + maxOverlayInputs
		if end > len(jobs) {
			end = len(jobs)
		}
		output := cfg.Output
		if end < len(jobs) {
			output = fmt.Sprintf(
				"<cliphub-pass-%03d.mp4>",
				start/maxOverlayInputs+1)
		}
		passCfg := cfg
		passCfg.Input = input
		passCfg.Output = output
		shared.PrintCommand("ffmpeg",
			renderFFmpegArgs(passCfg, jobs[start:end]))
		input = output
	}
}

// ffmpegFilter builds a simple overlay chain, enabling each PNG only between its start and end timestamps.
func ffmpegFilter(cfg Config, jobs []RenderJob) string {
	base := baseVideoFilter(cfg)
	if len(jobs) == 0 {
		return base + ";[base]copy[vout]"
	}

	parts := []string{base}
	input := "base"
	for i, job := range jobs {
		output := "vout"
		if i < len(jobs)-1 {
			output = fmt.Sprintf("v%d", i)
		}
		parts = append(parts, fmt.Sprintf(
			"[%s][%d:v]overlay=0:0:enable='between(t,%.3f,%.3f)'[%s]",
			input,
			i+1,
			job.Overlay.Start,
			job.Overlay.End,
			output,
		))
		input = output
	}
	return strings.Join(parts, ";")
}

// baseVideoFilter returns the first ffmpeg video filter, either letterboxed contain or center-cropped cover.
func baseVideoFilter(cfg Config) string {
	if cfg.Cover {
		return fmt.Sprintf(
			"[0:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:(iw-ow)/2:(ih-oh)/2[base]",
			cfg.Width,
			cfg.Height,
			cfg.Width,
			cfg.Height,
		)
	}
	return fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[base]",
		cfg.Width,
		cfg.Height,
		cfg.Width,
		cfg.Height,
	)
}
