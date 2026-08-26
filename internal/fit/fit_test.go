package fit

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath(filepath.Join("testdata", "ride.MP4"))
	want := filepath.Join("testdata", "ride_fit.MP4")
	if got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}

func TestVideoFilterContain(t *testing.T) {
	cfg := Config{Width: 1080, Height: 1920}
	got := videoFilter(cfg)
	want := "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2"
	if got != want {
		t.Fatalf("videoFilter() = %q, want %q", got, want)
	}
}

func TestVideoFilterCover(t *testing.T) {
	cfg := Config{Width: 1080, Height: 1920, Cover: true}
	got := videoFilter(cfg)
	want := "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920:(iw-ow)/2:(ih-oh)/2"
	if got != want {
		t.Fatalf("videoFilter() = %q, want %q", got, want)
	}
}

func TestFFmpegArgs(t *testing.T) {
	cfg := Config{
		Input:  "input.mp4",
		Output: "output.mp4",
		Width:  1080,
		Height: 1920,
	}
	got := ffmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-vf", "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2",
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "copy",
		"-movflags", "+faststart",
		"output.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ffmpegArgs() = %#v, want %#v", got, want)
	}
}
