package soundfx

// Config contains command-line options for adding timed sound effects.
type Config struct {
	Input       string
	Output      string
	Sounds      string
	Audio       string
	At          string
	Volume      float64
	VideoVolume float64
	DryRun      bool
	effects     []effect
}

// Item describes one sound effect on the video timeline.
type Item struct {
	At     string  `json:"at"`
	Audio  string  `json:"audio"`
	Volume float64 `json:"volume,omitempty"`
}

type effect struct {
	Audio  string
	At     float64
	Volume float64
}
