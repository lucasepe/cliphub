package join

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath(filepath.Join("testdata", "ride_join.json"))
	want := filepath.Join("testdata", "ride_joined.mp4")
	if got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}

func TestConcatFilterVideoOnly(t *testing.T) {
	got := concatFilter(3, false)
	want := "[0:v:0][1:v:0][2:v:0]concat=n=3:v=1:a=0[vout]"
	if got != want {
		t.Fatalf("concatFilter() = %q, want %q", got, want)
	}
}

func TestConcatFilterWithAudio(t *testing.T) {
	got := concatFilter(2, true)
	want := "[0:v:0][1:v:0]concat=n=2:v=1:a=0[vout];[0:a:0][1:a:0]concat=n=2:v=0:a=1[aout]"
	if got != want {
		t.Fatalf("concatFilter() = %q, want %q", got, want)
	}
}

func TestFadeFilter(t *testing.T) {
	items := []Item{
		{File: "a.mp4", Fade: 0.5},
		{File: "b.mp4", Fade: 0.25},
		{File: "c.mp4"},
	}
	got := fadeFilter(items, []float64{3, 4, 5})
	want := "[0:v:0]setpts=PTS-STARTPTS,format=yuv420p[v0];[1:v:0]setpts=PTS-STARTPTS,format=yuv420p[v1];[2:v:0]setpts=PTS-STARTPTS,format=yuv420p[v2];[v0][v1]xfade=transition=fade:duration=0.500:offset=2.500[x1];[x1][v2]xfade=transition=fade:duration=0.250:offset=6.250[vout]"
	if got != want {
		t.Fatalf("fadeFilter() = %q, want %q", got, want)
	}
}

func TestFFmpegArgsWithoutFade(t *testing.T) {
	cfg := Config{Input: "plan.json", Output: "out.mp4"}
	items := []Item{{File: "a.mp4"}, {File: "b.mp4"}}
	got, err := ffmpegArgs(cfg, items)
	if err != nil {
		t.Fatalf("ffmpegArgs() error = %v", err)
	}
	want := []string{
		"-y",
		"-i", "a.mp4",
		"-i", "b.mp4",
		"-filter_complex", "[0:v:0][1:v:0]concat=n=2:v=1:a=0[vout]",
		"-map", "[vout]",
		"-an",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"out.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ffmpegArgs() = %#v, want %#v", got, want)
	}
}
