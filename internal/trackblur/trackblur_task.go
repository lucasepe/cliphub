package trackblur

import (
	"context"
	"flag"
	"os"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the tracked blur command for the unified ClipHub CLI.
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

func (task *task) Name() string { return "trackblur" }

func (task *task) Synopsis() string {
	return "Blur a moving point by tracking it through the video."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Tracks a small image patch around a point and applies a circular blur that follows it.

The coordinates are in source-video pixels. Pick a distinctive point at -at, usually
near the center of the object you want to hide.

DEFAULT OUTPUT:

  ride.mp4 -> ride_trackblur.mp4

MULTIPLE BLURS:

If INPUT_BASENAME_blur.json exists, trackblur reads all blur regions from it:

  [
    {"at": "00:00:01", "x": 888, "y": 470, "radius": 58},
    {"at": "00:00:04.500", "to": "00:00:08", "x": 1320, "y": 620, "radius": 42}
  ]

Use -blurs to select another JSON file. If no JSON is found, -x and -y define
a single blur that starts at -at and continues to the end of the video.

EXAMPLES:

  cliphub trackblur -in ride.mp4 -at 00:00:01 -x 640 -y 920 -radius 48
  cliphub trackblur -in ride.mp4 -blurs ride_blur.json
  cliphub trackblur -in ride.mp4 -at 1.0 -x 640 -y 920 -radius 48 -blur 18 -search 120 -adapt 0.18
  cliphub trackblur -in ride.mp4 -at 1.0 -x 640 -y 920 -out ride_hidden.mp4
  cliphub trackblur -in ride.mp4 -at 1.0 -x 640 -y 920 -dry-run
  
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_trackblur beside the input video")
	fs.StringVar(&task.cfg.Blurs, "blurs", "", "JSON file containing blur regions; auto-detects INPUT_BASENAME_blur.json")
	fs.StringVar(&task.cfg.At, "at", "00:00:00", "timestamp where -x and -y are selected, as seconds or hh:mm:ss.xxx")
	fs.IntVar(&task.cfg.X, "x", -1, "initial x coordinate in source-video pixels")
	fs.IntVar(&task.cfg.Y, "y", -1, "initial y coordinate in source-video pixels")
	fs.IntVar(&task.cfg.Radius, "radius", defaultRadius, "blur circle radius in pixels")
	fs.IntVar(&task.cfg.Blur, "blur", defaultBlur, "box blur radius in pixels")
	fs.IntVar(&task.cfg.SearchRadius, "search", defaultSearchRadius, "maximum per-frame tracking movement in pixels")
	fs.IntVar(&task.cfg.TemplateSize, "template", defaultTemplateSize, "square tracking patch size in pixels")
	fs.Float64Var(&task.cfg.Adapt, "adapt", defaultAdapt, "template adaptation rate from 0 to 1")
	fs.StringVar(&task.cfg.Shape, "shape", "circle", "blur mask shape; currently only circle")
	fs.BoolVar(&task.cfg.Debug, "debug", false, "print tracked coordinates while processing")
	fs.BoolVar(&task.cfg.KeepWorkdir, "keep-workdir", false, "keep the temporary frame workspace for inspection")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg commands without processing frames")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}
	if cfg.Blurs == "" && cfg.Input != "" {
		blurs := defaultBlurPath(cfg.Input)
		if _, err := os.Stat(blurs); err == nil {
			cfg.Blurs = blurs
		} else if !os.IsNotExist(err) {
			return shared.FailTask(err)
		}
	}

	if err := blurTrackedPoint(cfg); err != nil {
		return shared.FailTask(err)
	}

	return cl.ExitSuccess
}
