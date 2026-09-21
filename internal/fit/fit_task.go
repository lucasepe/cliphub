package fit

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the video fitting command for the unified ClipHub CLI.
func Task(appName string) cl.Task {
	return &task{
		ctx:     context.Background(),
		appName: appName,
	}
}

var _ cl.Task = (*task)(nil)

type task struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *task) Name() string { return "fit" }

func (task *task) Synopsis() string {
	return "Fit a video into a social-ready frame."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Scales a video into a target canvas without adding captions or other overlays.

DEFAULT OUTPUT:

  ride.mp4 -> ride_fit.mp4

DEFAULT FRAME:

  1080x1920

MODES:

  contain    scale the whole video into the frame and add padding when needed
  cover      fill the frame and center-crop the overflow
  width-first cover
             fill the frame with a blurred cover background, then place the
             video scaled to the output width on top to preserve more content

EXAMPLES:

  cliphub fit -in ride.mp4
  cliphub fit -in ride.mp4 -cover
  cliphub fit -in ride.mp4 -cover -width-first
  cliphub fit -in ride.mp4 -width 1080 -height 1080
  cliphub fit -in ride.mp4 -out reel.mp4 -dry-run
  
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "",
		"output video path; defaults to INPUT_BASENAME_fit beside the input video")
	fs.IntVar(&task.cfg.Width, "width", defaultWidth, "output video width")
	fs.IntVar(&task.cfg.Height, "height", defaultHeight, "output video height")
	fs.BoolVar(&task.cfg.Cover, "cover", false,
		"scale and center-crop the video to fill the output frame")
	fs.BoolVar(&task.cfg.WidthFirst, "width-first", false,
		"with -cover, preserve the full video width over a blurred cover background")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false,
		"print ffmpeg command without writing the output")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}

	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}

	if err := fitVideo(cfg); err != nil {
		return shared.FailTask(err)
	}

	return cl.ExitSuccess
}
