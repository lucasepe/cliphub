package slow

import (
	"os"
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

func TestDefaultSlowCutsPath(t *testing.T) {
	got := defaultSlowCutsPath(filepath.Join("testdata", "D01_clip_4.MP4"))
	want := filepath.Join("testdata", "D01_clip_4_slow.json")
	if got != want {
		t.Fatalf("defaultSlowCutsPath() = %q, want %q", got, want)
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

func TestReadSlowRanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slow.json")
	data := []byte(`[
  {"from":"00:00:03.500","to":"00:00:07"},
  {"from":"00:00:10","to":"00:00:12.250"}
]`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	ranges, err := readSlowRanges(path)
	if err != nil {
		t.Fatalf("readSlowRanges() error = %v", err)
	}
	want := []slowRange{{From: 3.5, To: 7}, {From: 10, To: 12.25}}
	if !reflect.DeepEqual(ranges, want) {
		t.Fatalf("readSlowRanges() = %#v, want %#v", ranges, want)
	}
}

func TestReadSlowRangesRejectsOverlap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slow.json")
	data := []byte(`[
  {"from":"00:00:03","to":"00:00:07"},
  {"from":"00:00:06","to":"00:00:09"}
]`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSlowRanges(path); err == nil {
		t.Fatal("readSlowRanges() error = nil, want overlap error")
	}
}

func TestSelectiveSlowFilterVideoOnly(t *testing.T) {
	cfg := SlowConfig{
		Factor: 2,
		ranges: []slowRange{{From: 3, To: 7}, {From: 10, To: 12}},
	}
	got := slowFilter(cfg)
	want := "[0:v:0]trim=start=0:end=3,setpts=PTS-STARTPTS[v0];" +
		"[0:v:0]trim=start=3:end=7,setpts=(PTS-STARTPTS)*2[v1];" +
		"[0:v:0]trim=start=7:end=10,setpts=PTS-STARTPTS[v2];" +
		"[0:v:0]trim=start=10:end=12,setpts=(PTS-STARTPTS)*2[v3];" +
		"[0:v:0]trim=start=12,setpts=PTS-STARTPTS[v4];" +
		"[v0][v1][v2][v3][v4]concat=n=5:v=1:a=0[vout]"
	if got != want {
		t.Fatalf("slowFilter() = %q, want %q", got, want)
	}
}

func TestSelectiveSlowFilterWithAudio(t *testing.T) {
	cfg := SlowConfig{
		Factor: 4,
		Audio:  true,
		ranges: []slowRange{{From: 0, To: 2}},
	}
	got := slowFilter(cfg)
	want := "[0:v:0]trim=start=0:end=2,setpts=(PTS-STARTPTS)*4[v0];" +
		"[0:a:0]atrim=start=0:end=2,asetpts=PTS-STARTPTS,atempo=0.5,atempo=0.5[a0];" +
		"[0:v:0]trim=start=2,setpts=PTS-STARTPTS[v1];" +
		"[0:a:0]atrim=start=2,asetpts=PTS-STARTPTS[a1];" +
		"[v0][a0][v1][a1]concat=n=2:v=1:a=1[vout][aout]"
	if got != want {
		t.Fatalf("slowFilter() = %q, want %q", got, want)
	}
}
