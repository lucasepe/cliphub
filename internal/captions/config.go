package captions

const (
	defaultWidth     = 1080
	defaultHeight    = 1920
	defaultFontSize  = 64
	defaultPadding   = 96
	defaultMaxChars  = 24
	emojiGapRatio    = 0.18
	defaultBoxAlpha  = 0.45
	emojiCacheEnv    = "CLIPHUB_EMOJI_CACHE"
	twemojiBaseURL   = "https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/72x72"
	maxOverlayInputs = 200
)

// Config contains all command-line options needed to render text and process a video.
type Config struct {
	Input         string
	Output        string
	FontPath      string
	Width         int
	Height        int
	EmojiCacheDir string
	Overlays      string
	Cover         bool
	DryRun        bool
}

// Overlay describes one text item to draw over a specific time range.
type Overlay struct {
	Text       string   `json:"text"`
	Gravity    string   `json:"gravity"`
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	Box        *bool    `json:"box,omitempty"`
	BoxAlpha   *float64 `json:"box_alpha,omitempty"`
	BoxPadding *float64 `json:"box_padding,omitempty"`
	BoxRadius  *float64 `json:"box_radius,omitempty"`
	FontSize   *float64 `json:"font_size,omitempty"`
	Padding    *float64 `json:"padding,omitempty"`
	MaxChars   *int     `json:"max_chars,omitempty"`
}

// RenderJob joins an overlay configuration with the generated PNG path used by ffmpeg.
type RenderJob struct {
	Overlay Overlay
	Path    string
}
