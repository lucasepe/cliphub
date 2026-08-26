package slow

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultSlowPath(t *testing.T) {
	got := defaultSlowPath(filepath.Join("testdata", "D01_clip_4.MP4"))
	want := filepath.Join("testdata", "D01_clip_4_slow.MP4")
	if got != want {
		t.Fatalf("defaultSlowPath() = %q, want %q", got, want)
	}
}

func TestSlowFFmpegArgsDefaultsToVideoOnly(t *testing.T) {
	cfg := SlowConfig{
		Input:  "input.mp4",
		Output: "output.mp4",
		Factor: 2,
	}

	got := slowFFmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-filter_complex", "[0:v:0]setpts=2*PTS[vout]",
		"-map", "[vout]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-an",
		"output.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slowFFmpegArgs() = %#v, want %#v", got, want)
	}
}

func TestSlowFFmpegArgsWithAudio(t *testing.T) {
	cfg := SlowConfig{
		Input:  "input.mp4",
		Output: "output.mp4",
		Factor: 4,
		Audio:  true,
	}

	got := slowFFmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-filter_complex", "[0:v:0]setpts=4*PTS[vout];[0:a:0?]atempo=0.5,atempo=0.5[aout]",
		"-map", "[vout]",
		"-map", "[aout]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-c:a", "aac",
		"output.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slowFFmpegArgs() = %#v, want %#v", got, want)
	}
}

func TestValidateSlowConfigRejectsFastFactor(t *testing.T) {
	cfg := SlowConfig{
		Input:  "input.mp4",
		Output: "output.mp4",
		Factor: 1,
	}
	if err := validateSlowConfig(cfg); err == nil {
		t.Fatal("validateSlowConfig() error = nil, want error")
	}
}
