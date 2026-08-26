package soundtrack

// Config contains command-line options for adding a soundtrack to a video.
type Config struct {
	Input       string
	Output      string
	Audio       string
	MusicVolume float64
	VideoVolume float64
	Loop        bool
	DryRun      bool
}
