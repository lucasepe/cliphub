package slow

// SlowConfig contains command-line options for slowing down one video.
type SlowConfig struct {
	Input  string
	Output string
	Cuts   string
	Factor float64
	Audio  bool
	DryRun bool
	ranges []slowRange
}

// SlowItem describes one interval to slow on the source timeline.
type SlowItem struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type slowRange struct {
	From float64
	To   float64
}
