package preview

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

const (
	sizeSafetyFactor = 0.95
	defaultAudioRate = int64(64_000)
	minimumVideoRate = int64(50_000)
)

func defaultOutputPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_preview", ".mp4")
}

func makePreview(cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	targetBytes, err := parseSize(cfg.MaxSize)
	if err != nil {
		return fmt.Errorf("invalid -max-size: %w", err)
	}

	info, err := probeMedia(cfg.Input)
	if err != nil {
		return err
	}

	plan, err := buildPlan(info, targetBytes)
	if err != nil {
		return err
	}
	printPlan(cfg, info, plan)

	if cfg.DryRun {
		first, second := ffmpegArgs(cfg, plan, "<passlog>")
		shared.PrintCommand("ffmpeg", first)
		shared.PrintCommand("ffmpeg", second)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "cliphub-preview-")
	if err != nil {
		return fmt.Errorf("create two-pass workspace: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	first, second := ffmpegArgs(cfg, plan, filepath.Join(tmpDir, "pass"))
	if err := runFFmpeg(first); err != nil {
		return fmt.Errorf("preview first pass: %w", err)
	}

	if err := runFFmpeg(second); err != nil {
		return fmt.Errorf("preview %q: %w", cfg.Output, err)
	}

	return nil
}

func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}

	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}

	inAbs, _ := filepath.Abs(cfg.Input)
	outAbs, _ := filepath.Abs(cfg.Output)
	if inAbs == outAbs {
		return errors.New("-out must differ from -in")
	}

	return nil
}

func probeMedia(path string) (mediaInfo, error) {
	args := []string{
		"-v",
		"error",
		"-show_entries",
		"format=duration,size:stream=codec_type,width,height:stream_tags=rotate:stream_side_data=rotation",
		"-of",
		"json",
		path,
	}

	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return mediaInfo{}, fmt.Errorf("probe %q with ffprobe: %w", path, err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			Tags      struct {
				Rotate string `json:"rotate"`
			} `json:"tags"`
			SideData []struct {
				Rotation int `json:"rotation"`
			} `json:"side_data_list"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return mediaInfo{}, fmt.Errorf("decode ffprobe output: %w", err)
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil || duration <= 0 || math.IsInf(duration, 0) || math.IsNaN(duration) {
		return mediaInfo{}, errors.New("ffprobe did not report a valid video duration")
	}

	size, _ := strconv.ParseInt(result.Format.Size, 10, 64)
	info := mediaInfo{Duration: duration, Size: size}

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "audio":
			info.HasAudio = true
		case "video":
			if info.Width != 0 {
				continue
			}

			info.Width, info.Height = stream.Width, stream.Height
			rotation, _ := strconv.Atoi(stream.Tags.Rotate)
			for _, side := range stream.SideData {
				if side.Rotation != 0 {
					rotation = side.Rotation
					break
				}
			}

			if abs(rotation)%180 == 90 {
				info.Width, info.Height = info.Height, info.Width
			}
		}
	}

	if info.Width <= 0 || info.Height <= 0 {
		return mediaInfo{}, errors.New("ffprobe did not find a video stream")
	}

	return info, nil
}

func buildPlan(info mediaInfo, targetBytes int64) (encodePlan, error) {
	budgetBytes := int64(float64(targetBytes) * sizeSafetyFactor)
	totalRate := int64(float64(budgetBytes*8) / info.Duration)
	audioRate := int64(0)

	if info.HasAudio {
		audioRate = defaultAudioRate
	}

	videoRate := totalRate - audioRate
	if videoRate < minimumVideoRate {
		minimum := int64(float64(minimumVideoRate+audioRate) * info.Duration / 8 / sizeSafetyFactor)
		return encodePlan{}, fmt.Errorf("-max-size is too small for a %.1f second video (need room for at least %s)", info.Duration, formatSize(minimum))
	}

	width, height := previewDimensions(info.Width, info.Height, videoRate)

	return encodePlan{
		Width:        width,
		Height:       height,
		VideoBitrate: videoRate,
		AudioBitrate: audioRate,
		TargetBytes:  targetBytes,
		BudgetBytes:  budgetBytes,
	}, nil
}

func previewDimensions(width, height int, videoRate int64) (int, int) {
	maxLong := 480

	switch {
	case videoRate >= 3_000_000:
		maxLong = 1920
	case videoRate >= 1_500_000:
		maxLong = 1280
	case videoRate >= 700_000:
		maxLong = 960
	case videoRate >= 350_000:
		maxLong = 720
	}

	long := max(width, height)
	if long <= maxLong {
		return even(width), even(height)
	}

	scale := float64(maxLong) / float64(long)

	return even(
			int(math.Round(float64(width) * scale))),
		even(int(math.Round(float64(height) * scale)))
}

func ffmpegArgs(cfg Config, plan encodePlan, passlog string) ([]string, []string) {
	videoRate := strconv.FormatInt(plan.VideoBitrate, 10)

	filter := fmt.Sprintf("scale=%d:%d", plan.Width, plan.Height)

	common := []string{
		"-y", "-i",
		cfg.Input,
		"-map", "0:v:0",
		"-vf", filter,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-b:v", videoRate,
		"-passlogfile", passlog,
	}

	first := append(
		append([]string{}, common...),
		"-pass", "1",
		"-an", "-f",
		"null", os.DevNull)

	second := append(
		append([]string{}, common...), "-pass", "2")

	if plan.AudioBitrate > 0 {
		second = append(second, "-map", "0:a?", "-c:a", "aac", "-b:a", strconv.FormatInt(plan.AudioBitrate, 10))
	} else {
		second = append(second, "-an")
	}
	second = append(second, "-movflags", "+faststart", cfg.Output)

	return first, second
}

func runFFmpeg(args []string) error {
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseSize(value string) (int64, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return 0, errors.New("size is empty")
	}

	i := 0
	for i < len(s) && (s[i] == '.' || s[i] >= '0' && s[i] <= '9') {
		i++
	}

	if i == 0 {
		return 0, fmt.Errorf("%q is not a positive size", value)
	}

	number, err := strconv.ParseFloat(s[:i], 64)
	if err != nil || number <= 0 || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, fmt.Errorf("%q is not a positive size", value)
	}

	unit := strings.ToUpper(strings.TrimSpace(s[i:]))
	multiplier := float64(1_000_000)

	switch unit {
	case "", "MB":
	case "KB":
		multiplier = 1_000
	case "GB":
		multiplier = 1_000_000_000
	case "KIB":
		multiplier = 1 << 10
	case "MIB":
		multiplier = 1 << 20
	case "GIB":
		multiplier = 1 << 30
	default:
		return 0, fmt.Errorf("unknown unit %q", s[i:])
	}

	bytes := number * multiplier
	if bytes > math.MaxInt64 {
		return 0, errors.New("size is too large")
	}

	return int64(bytes), nil
}

func printPlan(cfg Config, info mediaInfo, plan encodePlan) {
	fmt.Printf("Input:      %s (%dx%d, %.1fs, %s)\n",
		cfg.Input, info.Width, info.Height, info.Duration, formatSize(info.Size))
	fmt.Printf("Output:     %s (%dx%d)\n",
		cfg.Output, plan.Width, plan.Height)
	fmt.Printf("Size limit: %s (encoding budget %s)\n",
		formatSize(plan.TargetBytes), formatSize(plan.BudgetBytes))
	fmt.Printf("Bitrate:    video %.0f kb/s",
		float64(plan.VideoBitrate)/1000)

	if plan.AudioBitrate > 0 {
		fmt.Printf(", audio %.0f kb/s", float64(plan.AudioBitrate)/1000)
	}

	fmt.Println()
}

func formatSize(bytes int64) string {
	if bytes <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/1_000_000)
}

func even(value int) int {
	if value < 2 {
		return 2
	}
	return value - value%2
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
