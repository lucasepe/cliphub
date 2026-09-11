package slice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

var invalidFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._ -]+`)

// defaultSlicesPath returns the default JSON path for clip ranges beside the input video.
func defaultSlicesPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+"_slices.json")
}

// validateSliceConfig checks that the required video and cut-list paths were provided.
func validateSliceConfig(cfg SliceConfig) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Cuts == "" {
		return errors.New("missing required -cuts JSON path")
	}
	return nil
}

// sliceVideo loads the clip list, prepares output paths, and writes every requested clip.
func sliceVideo(cfg SliceConfig) error {
	items, err := readSliceItems(cfg.Cuts)
	if err != nil {
		return err
	}
	jobs, err := buildSliceJobs(cfg.Input, items)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if cfg.DryRun {
			shared.PrintCommand("ffmpeg", sliceFFmpegArgs(cfg.Input, job))
			continue
		}
		if err := runSliceFFmpeg(cfg.Input, job); err != nil {
			return err
		}
	}
	return nil
}

// readSliceItems decodes a JSON array of clip ranges from path.
func readSliceItems(path string) ([]SliceItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cuts %q: %w", path, err)
	}
	var items []SliceItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse cuts %q: %w", path, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("cuts file %q contains no clips", path)
	}
	return items, nil
}

// buildSliceJobs validates each clip and resolves output files in the input video's directory.
func buildSliceJobs(inputPath string, items []SliceItem) ([]SliceJob, error) {
	dir := filepath.Dir(inputPath)
	videoName := videoBaseName(inputPath)
	ext := filepath.Ext(inputPath)
	if ext == "" {
		ext = ".mp4"
	}

	jobs := make([]SliceJob, 0, len(items))
	for i, item := range items {
		if err := validateSliceItem(i, item); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("%s_clip_%d", videoName, i+1)
		}
		name = cleanClipName(name)
		if name == "" {
			name = fmt.Sprintf("%s_clip_%d", videoName, i+1)
		}
		jobs = append(jobs, SliceJob{
			From:   item.From,
			To:     item.To,
			Output: filepath.Join(dir, name+ext),
		})
	}
	return jobs, nil
}

// videoBaseName returns the filename without extension for use in generated clip names.
func videoBaseName(inputPath string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// validateSliceItem checks timestamps and rejects empty or inverted clip ranges.
func validateSliceItem(index int, item SliceItem) error {
	_, _, err := shared.ValidateTimeRange(shared.TimeRange{From: item.From, To: item.To})
	if err != nil {
		return fmt.Errorf("clip %d: %w", index+1, err)
	}
	return nil
}

// cleanClipName turns a user-provided clip name into a single safe filename stem.
func cleanClipName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = invalidFilenameChars.ReplaceAllString(name, "")
	name = strings.Join(strings.Fields(name), "_")
	return strings.Trim(name, ". _-")
}

// runSliceFFmpeg asks ffmpeg to copy the selected time range into one output file.
func runSliceFFmpeg(inputPath string, job SliceJob) error {
	cmd := exec.Command("ffmpeg", sliceFFmpegArgs(inputPath, job)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("slice %q: %w", job.Output, err)
	}
	return nil
}

// sliceFFmpegArgs builds a stream-copy command that keeps the main video and optional audio only.
func sliceFFmpegArgs(inputPath string, job SliceJob) []string {
	return []string{
		"-y",
		"-ss", job.From,
		"-to", job.To,
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c", "copy",
		"-movflags", "+faststart",
		job.Output,
	}
}
