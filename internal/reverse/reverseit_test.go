package reverse

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultReversePath(t *testing.T) {
	got := defaultReversePath(filepath.Join("testdata", "D01_clip_4.MP4"))
	want := filepath.Join("testdata", "D01_clip_4_reverse.MP4")
	if got != want {
		t.Fatalf("defaultReversePath() = %q, want %q", got, want)
	}
}

func TestReverseFFmpegArgsDefaultsToVideoOnly(t *testing.T) {
	cfg := ReverseConfig{
		Input:  "input.mp4",
		Output: "output.mp4",
	}

	got := reverseFFmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-filter_complex", "[0:v:0]reverse[vout]",
		"-map", "[vout]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-an",
		"output.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reverseFFmpegArgs() = %#v, want %#v", got, want)
	}
}

func TestReverseFFmpegArgsWithAudio(t *testing.T) {
	cfg := ReverseConfig{
		Input:  "input.mp4",
		Output: "output.mp4",
		Audio:  true,
	}

	got := reverseFFmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-filter_complex", "[0:v:0]reverse[vout];[0:a:0?]areverse[aout]",
		"-map", "[vout]",
		"-map", "[aout]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-c:a", "aac",
		"output.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reverseFFmpegArgs() = %#v, want %#v", got, want)
	}
}
