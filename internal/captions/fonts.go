package captions

import (
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
)

// isKnownColorEmojiFont reports whether path points to a color emoji font that FreeType-based gg cannot render.
func isKnownColorEmojiFont(path string) bool {
	name := strings.ToLower(path)
	colorEmojiFonts := []string{
		"apple color emoji",
		"notocoloremoji",
		"noto color emoji",
		"seguiemj",
		"segoe ui emoji",
	}
	for _, font := range colorEmojiFonts {
		if strings.Contains(name, font) {
			return true
		}
	}
	return false
}

// loadFontFace parses TTF or OTF font data and returns a drawable face at the requested size.
// When path is empty, it uses the embedded Go Mono font from golang.org/x/image.
func loadFontFace(path string, size float64) (font.Face, error) {
	data := gomono.TTF
	if path != "" {
		fileData, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		data = fileData
	}
	parsedFont, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
