package fit

const (
	defaultWidth  = 1080
	defaultHeight = 1920
)

// Config contains command-line options for fitting a video into a target frame.
type Config struct {
	Input  string
	Output string
	Width  int
	Height int
	Cover  bool
	DryRun bool
}
