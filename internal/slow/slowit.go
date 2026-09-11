package slow

import (
	"encoding/json"
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

// defaultSlowCutsPath returns the optional selective-slow JSON path beside the input video.
func defaultSlowCutsPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_slow", ".json")
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
	if cfg.Cuts != "" {
		ranges, err := readSlowRanges(cfg.Cuts)
		if err != nil {
			return err
		}
		cfg.ranges = ranges
	}
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

// readSlowRanges decodes and validates chronologically ordered, non-overlapping ranges.
func readSlowRanges(path string) ([]slowRange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read slow cuts %q: %w", path, err)
	}
	var items []SlowItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse slow cuts %q: %w", path, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("slow cuts file %q contains no ranges", path)
	}

	ranges := make([]slowRange, 0, len(items))
	for i, item := range items {
		from, to, err := shared.ValidateTimeRange(shared.TimeRange{From: item.From, To: item.To})
		if err != nil {
			return nil, fmt.Errorf("slow range %d: %w", i+1, err)
		}
		if i > 0 && from < ranges[i-1].To {
			return nil, fmt.Errorf("slow range %d overlaps or precedes range %d", i+1, i)
		}
		ranges = append(ranges, slowRange{From: from, To: to})
	}
	return ranges, nil
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
	if len(cfg.ranges) > 0 {
		return selectiveSlowFilter(cfg)
	}
	videoFilter := fmt.Sprintf("[0:v:0]setpts=%s*PTS[vout]", formatFloat(cfg.Factor))
	if !cfg.Audio {
		return videoFilter
	}
	return videoFilter + ";[0:a:0?]" + atempoFilter(1/cfg.Factor) + "[aout]"
}

type slowSegment struct {
	from float64
	to   float64
	slow bool
	tail bool
}

// selectiveSlowFilter splits the original timeline, slows selected pieces, and rejoins it.
func selectiveSlowFilter(cfg SlowConfig) string {
	segments := make([]slowSegment, 0, len(cfg.ranges)*2+1)
	cursor := 0.0
	for _, r := range cfg.ranges {
		if r.From > cursor {
			segments = append(segments, slowSegment{from: cursor, to: r.From})
		}
		segments = append(segments, slowSegment{from: r.From, to: r.To, slow: true})
		cursor = r.To
	}
	segments = append(segments, slowSegment{from: cursor, tail: true})

	filters := make([]string, 0, len(segments)*2+1)
	inputs := strings.Builder{}
	for i, segment := range segments {
		trim := "start=" + formatFloat(segment.from)
		if !segment.tail {
			trim += ":end=" + formatFloat(segment.to)
		}
		videoPTS := "PTS-STARTPTS"
		if segment.slow {
			videoPTS = "(" + videoPTS + ")*" + formatFloat(cfg.Factor)
		}
		filters = append(filters, fmt.Sprintf("[0:v:0]trim=%s,setpts=%s[v%d]", trim, videoPTS, i))
		inputs.WriteString(fmt.Sprintf("[v%d]", i))

		if cfg.Audio {
			audioFilter := fmt.Sprintf("[0:a:0]atrim=%s,asetpts=PTS-STARTPTS", trim)
			if segment.slow {
				audioFilter += "," + atempoFilter(1/cfg.Factor)
			}
			filters = append(filters, fmt.Sprintf("%s[a%d]", audioFilter, i))
			inputs.WriteString(fmt.Sprintf("[a%d]", i))
		}
	}
	concat := fmt.Sprintf("%sconcat=n=%d:v=1:a=0[vout]", inputs.String(), len(segments))
	if cfg.Audio {
		concat = fmt.Sprintf("%sconcat=n=%d:v=1:a=1[vout][aout]", inputs.String(), len(segments))
	}
	filters = append(filters, concat)
	return strings.Join(filters, ";")
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
