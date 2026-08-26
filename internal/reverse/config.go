package reverse

// ReverseConfig contains command-line options for reversing one video.
type ReverseConfig struct {
	Input  string
	Output string
	Audio  bool
	DryRun bool
}
