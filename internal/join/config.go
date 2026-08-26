package join

// Config contains command-line options for joining multiple clips into one video.
type Config struct {
	Input  string
	Output string
	Audio  bool
	DryRun bool
}

// Item describes one clip in a join plan.
type Item struct {
	File string  `json:"file"`
	Fade float64 `json:"fade,omitempty"`
}
