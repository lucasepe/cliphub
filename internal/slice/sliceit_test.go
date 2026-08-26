package slice

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultSlicesPath(t *testing.T) {
	got := defaultSlicesPath(filepath.Join("testdata", "ride.mp4"))
	want := filepath.Join("testdata", "ride_slices.json")
	if got != want {
		t.Fatalf("defaultSlicesPath() = %q, want %q", got, want)
	}
}

func TestBuildSliceJobs(t *testing.T) {
	items := []SliceItem{
		{From: "00:00:01", To: "00:00:03", Name: "Golden hour"},
		{From: "00:00:05", To: "00:00:07"},
	}

	jobs, err := buildSliceJobs(filepath.Join("testdata", "input.mp4"), items)
	if err != nil {
		t.Fatalf("buildSliceJobs() error = %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}

	wantFirst := filepath.Join("testdata", "Golden_hour.mp4")
	if jobs[0].Output != wantFirst {
		t.Fatalf("jobs[0].Output = %q, want %q", jobs[0].Output, wantFirst)
	}

	wantSecond := filepath.Join("testdata", "input_clip_2.mp4")
	if jobs[1].Output != wantSecond {
		t.Fatalf("jobs[1].Output = %q, want %q", jobs[1].Output, wantSecond)
	}
}

func TestValidateSliceItemRejectsInvertedRange(t *testing.T) {
	item := SliceItem{From: "00:00:10", To: "00:00:05"}
	if err := validateSliceItem(0, item); err == nil {
		t.Fatal("validateSliceItem() error = nil, want error")
	}
}

func TestSliceFFmpegArgsKeepOnlyMainVideoAndOptionalAudio(t *testing.T) {
	job := SliceJob{
		From:   "00:00:01",
		To:     "00:00:03",
		Output: "clip.mp4",
	}

	got := sliceFFmpegArgs("input.mp4", job)
	want := []string{
		"-y",
		"-ss", "00:00:01",
		"-to", "00:00:03",
		"-i", "input.mp4",
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c", "copy",
		"-movflags", "+faststart",
		"clip.mp4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sliceFFmpegArgs() = %#v, want %#v", got, want)
	}
}
