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

func TestVideoFilterCoverWidthFirst(t *testing.T) {
	cfg := Config{Width: 1080, Height: 1920, Cover: true, WidthFirst: true}
	got := videoFilter(cfg)
	want := "[0:v:0]split=2[bg][fg];[bg]scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920:(iw-ow)/2:(ih-oh)/2,boxblur=20:1[bg];[fg]scale=1080:-2[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2,setsar=1"
	if got != want {
		t.Fatalf("videoFilter() = %q, want %q", got, want)
	}
}

func TestValidateConfigRejectsWidthFirstWithoutCover(t *testing.T) {
	cfg := Config{
		Input:      "input.mp4",
		Output:     "output.mp4",
		Width:      1080,
		Height:     1920,
		WidthFirst: true,
	}
	if err := validateConfig(cfg); err == nil {
		t.Fatal("validateConfig() error = nil, want error")
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
