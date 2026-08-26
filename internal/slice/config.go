package slice

// SliceConfig contains command-line options for cutting one video into named clips.
type SliceConfig struct {
	Input  string
	Cuts   string
	DryRun bool
}

// SliceItem describes one clip to extract from the source video.
type SliceItem struct {
	From string `json:"from"`
	To   string `json:"to"`
	Name string `json:"name,omitempty"`
}

// SliceJob contains a validated clip range and the path ffmpeg should write.
type SliceJob struct {
	From   string
	To     string
	Output string
}
