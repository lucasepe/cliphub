package preview

const defaultMaxSize = "30MB"

// Config contains command-line options for making a compact video preview.
type Config struct {
	Input   string
	Output  string
	MaxSize string
	DryRun  bool
}

type mediaInfo struct {
	Duration float64
	Width    int
	Height   int
	HasAudio bool
	Size     int64
}

type encodePlan struct {
	Width        int
	Height       int
	VideoBitrate int64
	AudioBitrate int64
	TargetBytes  int64
	BudgetBytes  int64
}
