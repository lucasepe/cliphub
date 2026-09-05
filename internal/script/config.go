package script

const (
	defaultFontSize = 80
	defaultGravity  = "bottom"
	readableWPM     = 200
)

type Config struct {
	Input    string
	Output   string
	Duration float64
	Words    bool
}
