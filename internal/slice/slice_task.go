package slice

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// SliceTask returns the video slicing command for the unified ClipHub CLI.
func SliceTask(appName string) cl.Task {
	return &sliceTask{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*sliceTask)(nil)

type sliceTask struct {
	appName string
	ctx     context.Context
	cfg     SliceConfig
}

func (task *sliceTask) Name() string { return "slice" }

func (task *sliceTask) Synopsis() string {
	return "Cut a video into clips from a JSON cut list."
}

func (task *sliceTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Reads clip ranges from JSON and writes every clip beside the input video.

DEFAULT FILE:

  ride.mp4 -> ride_slices.json

SLICE JSON:

  [
    {
      "from": "00:00:03",
      "to": "00:00:08",
      "name": "intro"
    },
    {
      "from": "00:01:12",
      "to": "00:01:20"
    }
  ]

FIELDS:

  from    required, hh:mm:ss or hh:mm:ss.xxx
  to      required, hh:mm:ss or hh:mm:ss.xxx
  name    optional output filename stem

If name is omitted, outputs are named INPUT_clip_N with the source extension.

EXAMPLES:

  cliphub slice -in ride.mp4
  cliphub slice -in ride.mp4 -cuts custom_slices.json
  cliphub slice -in ride.mp4 -dry-run
  
`,
	})
}

func (task *sliceTask) Ctx() context.Context { return task.ctx }

func (task *sliceTask) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Cuts, "cuts", "", "JSON file containing clip ranges; defaults to INPUT_BASENAME_slices.json beside the input video")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg commands without cutting")
}

func (task *sliceTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Cuts == "" && cfg.Input != "" {
		cfg.Cuts = defaultSlicesPath(cfg.Input)
	}
	if err := validateSliceConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := sliceVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
