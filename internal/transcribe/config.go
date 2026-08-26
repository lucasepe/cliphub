package transcribe

import "github.com/lucasepe/cliphub/internal/captions"

const (
	defaultFontSize = 64
	defaultPadding  = 96
	defaultMaxChars = 24
	defaultBoxAlpha = 0.45
)

type Overlay = captions.Overlay

type TranscribeConfig struct {
	Input      string
	Output     string
	Whisper    string
	NoGPU      bool
	Model      string
	Language   string
	Gravity    string
	FontSize   float64
	Padding    float64
	MaxChars   int
	Box        bool
	BoxAlpha   float64
	BoxPadding float64
	BoxRadius  float64
	Words      bool
}

type WhisperOutput struct {
	Transcription []WhisperSegment `json:"transcription"`
}

type WhisperSegment struct {
	Text       string            `json:"text"`
	Timestamps WhisperTimestamps `json:"timestamps"`
	Tokens     []WhisperToken    `json:"tokens,omitempty"`
}

type WhisperTimestamps struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type WhisperToken struct {
	Text    string         `json:"text"`
	Offsets WhisperOffsets `json:"offsets"`
}

type WhisperOffsets struct {
	From int `json:"from"`
	To   int `json:"to"`
}
