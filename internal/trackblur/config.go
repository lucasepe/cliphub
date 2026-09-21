package trackblur

const (
	defaultRadius       = 42
	defaultBlur         = 12
	defaultSearchRadius = 90
	defaultTemplateSize = 96
	defaultAdapt        = 0.12
)

// Config contains the settings for a tracked local blur.
type Config struct {
	Input        string
	Output       string
	Blurs        string
	At           string
	X            int
	Y            int
	Radius       int
	Blur         int
	SearchRadius int
	TemplateSize int
	Adapt        float64
	Shape        string
	Debug        bool
	KeepWorkdir  bool
	DryRun       bool
	items        []BlurItem
}

// BlurItem describes one moving region to blur on the source timeline.
type BlurItem struct {
	At           string  `json:"at"`
	To           string  `json:"to,omitempty"`
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Radius       int     `json:"radius,omitempty"`
	Blur         int     `json:"blur,omitempty"`
	SearchRadius int     `json:"search,omitempty"`
	TemplateSize int     `json:"template,omitempty"`
	Adapt        float64 `json:"adapt,omitempty"`
	Shape        string  `json:"shape,omitempty"`
}
