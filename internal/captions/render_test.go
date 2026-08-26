package captions

import "testing"

func TestTextYTopKeepsBoxInsideCanvas(t *testing.T) {
	box := true
	fontSize := 160.0
	padding := 24.0
	boxPadding := 0.0
	overlay := Overlay{
		Gravity:    "top",
		FontSize:   &fontSize,
		Padding:    &padding,
		Box:        &box,
		BoxPadding: &boxPadding,
	}

	y := textY("top", 1920, 200, overlay)
	boxY := y - fontSize*0.85 - effectiveBoxPadding(overlay)
	if boxY < padding {
		t.Fatalf("boxY = %v, want at least padding %v", boxY, padding)
	}
}

func TestTextYTopKeepsTextInsideCanvasWithoutBox(t *testing.T) {
	box := false
	fontSize := 160.0
	padding := 24.0
	boxPadding := 0.0
	overlay := Overlay{
		Gravity:    "top",
		FontSize:   &fontSize,
		Padding:    &padding,
		Box:        &box,
		BoxPadding: &boxPadding,
	}

	y := textY("top", 1920, 200, overlay)
	textTop := y - fontSize*0.85
	if textTop < padding {
		t.Fatalf("textTop = %v, want at least padding %v", textTop, padding)
	}
}
