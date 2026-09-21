package trackblur

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrameRate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
		label string
	}{
		{name: "fraction", value: "30000/1001", want: 29.97002997002997, label: "30000/1001"},
		{name: "decimal", value: "25", want: 25, label: "25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, label, err := parseFrameRate(tt.value)
			if err != nil {
				t.Fatalf("parseFrameRate(%q) error = %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("parseFrameRate(%q) fps = %v, want %v", tt.value, got, tt.want)
			}
			if label != tt.label {
				t.Fatalf("parseFrameRate(%q) label = %q, want %q", tt.value, label, tt.label)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		value string
		want  float64
	}{
		{value: "1.25", want: 1.25},
		{value: "00:00:01.250", want: 1.25},
		{value: "00:01:02", want: 62},
	}

	for _, tt := range tests {
		got, err := parseTimestamp(tt.value)
		if err != nil {
			t.Fatalf("parseTimestamp(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("parseTimestamp(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestDefaultBlurPath(t *testing.T) {
	got := defaultBlurPath("/tmp/ride.MP4")
	want := filepath.Join("/tmp", "ride_blur.json")
	if got != want {
		t.Fatalf("defaultBlurPath() = %q, want %q", got, want)
	}
}

func TestReadBlurItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ride_blur.json")
	data := `[
		{"at":"00:00:01","x":888,"y":470,"radius":58},
		{"at":"00:00:04.500","to":"00:00:08","x":1320,"y":620}
	]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write test blur file: %v", err)
	}

	items, err := readBlurItems(path)
	if err != nil {
		t.Fatalf("readBlurItems() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("readBlurItems() len = %d, want 2", len(items))
	}
	if items[1].To != "00:00:08" {
		t.Fatalf("second item To = %q, want 00:00:08", items[1].To)
	}
}

func TestBuildTrackers(t *testing.T) {
	items := []BlurItem{
		{At: "00:00:01", To: "00:00:03", X: 10, Y: 20},
	}

	trackers, err := buildTrackers(items, mediaInfo{FPS: 30, FPSLabel: "30"})
	if err != nil {
		t.Fatalf("buildTrackers() error = %v", err)
	}
	if len(trackers) != 1 {
		t.Fatalf("buildTrackers() len = %d, want 1", len(trackers))
	}
	if trackers[0].startFrame != 30 {
		t.Fatalf("startFrame = %d, want 30", trackers[0].startFrame)
	}
	if trackers[0].endFrame != 90 {
		t.Fatalf("endFrame = %d, want 90", trackers[0].endFrame)
	}
	if trackers[0].item.Radius != defaultRadius {
		t.Fatalf("default radius = %d, want %d", trackers[0].item.Radius, defaultRadius)
	}
}
