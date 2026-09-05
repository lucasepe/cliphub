package preview

import (
	"path/filepath"
	"testing"
)

func TestDefaultOutputPathUsesMP4(t *testing.T) {
	got := defaultOutputPath(filepath.Join("testdata", "ride.MOV"))
	want := filepath.Join("testdata", "ride_preview.mp4")
	if got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}

func TestParseSize(t *testing.T) {
	tests := map[string]int64{
		"30":     30_000_000,
		"30MB":   30_000_000,
		"1.5 MB": 1_500_000,
		"2MiB":   2 * 1024 * 1024,
		"750KB":  750_000,
	}
	for input, want := range tests {
		got, err := parseSize(input)
		if err != nil {
			t.Fatalf("parseSize(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("parseSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseSizeRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "0", "MB", "12XB", "-2MB"} {
		if _, err := parseSize(input); err == nil {
			t.Errorf("parseSize(%q) error = nil, want error", input)
		}
	}
}

func TestPreviewDimensionsPreservesLandscapeAspectRatio(t *testing.T) {
	width, height := previewDimensions(3840, 2160, 800_000)
	if width != 960 || height != 540 {
		t.Fatalf("previewDimensions() = %dx%d, want 960x540", width, height)
	}
}

func TestPreviewDimensionsPreservesPortraitAspectRatio(t *testing.T) {
	width, height := previewDimensions(1080, 1920, 400_000)
	if width != 404 || height != 720 {
		t.Fatalf("previewDimensions() = %dx%d, want 404x720", width, height)
	}
}

func TestBuildPlanStaysInsideLimit(t *testing.T) {
	info := mediaInfo{Duration: 300, Width: 1920, Height: 1080, HasAudio: true}
	plan, err := buildPlan(info, 30_000_000)
	if err != nil {
		t.Fatal(err)
	}
	projected := int64(float64(plan.VideoBitrate+plan.AudioBitrate) * info.Duration / 8)
	if projected > plan.TargetBytes {
		t.Fatalf("projected bytes = %d, limit = %d", projected, plan.TargetBytes)
	}
	if plan.Width != 720 || plan.Height != 404 {
		t.Fatalf("dimensions = %dx%d, want 720x404", plan.Width, plan.Height)
	}
}
