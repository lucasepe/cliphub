package slow

// SlowConfig contains command-line options for slowing down one video.
type SlowConfig struct {
	Input  string
	Output string
	Factor float64
	Audio  bool
	DryRun bool
}
