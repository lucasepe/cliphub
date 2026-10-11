package soundfx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	input := filepath.Join("testdata", "clip.mp4")
	if got, want := defaultOutputPath(input), filepath.Join("testdata", "clip_soundfx.mp4"); got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
	if got, want := defaultSoundsPath(input), filepath.Join("testdata", "clip_soundfx.json"); got != want {
		t.Fatalf("defaultSoundsPath() = %q, want %q", got, want)
	}
}

func TestReadItemsResolvesPathsAndDefaultsVolume(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "effects.json")
	if err := os.WriteFile(path, []byte(`[{"at":"00:00:08","audio":"reaction.mp3"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readItems(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []effect{{Audio: filepath.Join(dir, "reaction.mp3"), At: 8, Volume: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readItems() = %#v, want %#v", got, want)
	}
}

func TestAudioFilterDelaysAndMixesEffects(t *testing.T) {
	cfg := Config{
		VideoVolume: 0.8,
		effects: []effect{
			{At: 8, Volume: 1},
			{At: 12.5, Volume: 0.7},
		},
	}
	got := audioFilter(cfg)
	want := "[0:a:0]volume=0.8[base];[1:a:0]volume=1,adelay=8000:all=1[sfx0];[2:a:0]volume=0.7,adelay=12500:all=1[sfx1];[base][sfx0][sfx1]amix=inputs=3:duration=first:dropout_transition=0:normalize=0[aout]"
	if got != want {
		t.Fatalf("audioFilter() = %q, want %q", got, want)
	}
}

func TestFFmpegArgsCopiesVideo(t *testing.T) {
	cfg := Config{
		Input:       "input.mp4",
		Output:      "out.mp4",
		VideoVolume: 1,
		effects:     []effect{{Audio: "reaction.mp3", At: 8, Volume: 1}},
	}
	got := ffmpegArgs(cfg)
	want := []string{
		"-y", "-i", "input.mp4", "-i", "reaction.mp3",
		"-filter_complex", "[0:a:0]volume=1[base];[1:a:0]volume=1,adelay=8000:all=1[sfx0];[base][sfx0]amix=inputs=2:duration=first:dropout_transition=0:normalize=0[aout]",
		"-map", "0:v:0", "-map", "[aout]", "-c:v", "copy", "-c:a", "aac", "-movflags", "+faststart", "out.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ffmpegArgs() = %#v, want %#v", got, want)
	}
}
