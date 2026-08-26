package soundtrack

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath(filepath.Join("testdata", "clip.mp4"))
	want := filepath.Join("testdata", "clip_soundtrack.mp4")
	if got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}

func TestAudioFilterMixesOriginalAndMusic(t *testing.T) {
	cfg := Config{MusicVolume: 0.8, VideoVolume: 0.25}
	got := audioFilter(cfg)
	want := "[0:a:0]volume=0.25[video];[1:a:0]volume=0.8[music];[video][music]amix=inputs=2:duration=shortest:dropout_transition=0[aout]"
	if got != want {
		t.Fatalf("audioFilter() = %q, want %q", got, want)
	}
}

func TestAudioFilterCanMuteOriginalAudio(t *testing.T) {
	cfg := Config{MusicVolume: 1, VideoVolume: 0}
	got := audioFilter(cfg)
	want := "[1:a:0]volume=1[music];[music]anull[aout]"
	if got != want {
		t.Fatalf("audioFilter() = %q, want %q", got, want)
	}
}

func TestFFmpegArgsWithLoop(t *testing.T) {
	cfg := Config{
		Input:       "input.mp4",
		Output:      "out.mp4",
		Audio:       "music.mp3",
		MusicVolume: 0.8,
		VideoVolume: 0,
		Loop:        true,
	}
	got := ffmpegArgs(cfg)
	want := []string{
		"-y",
		"-i", "input.mp4",
		"-stream_loop", "-1",
		"-i", "music.mp3",
		"-filter_complex", "[1:a:0]volume=0.8[music];[music]anull[aout]",
		"-map", "0:v:0",
		"-map", "[aout]",
		"-c:v", "copy",
		"-c:a", "aac",
		"-shortest",
		"-movflags", "+faststart",
		"out.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ffmpegArgs() = %#v, want %#v", got, want)
	}
}
