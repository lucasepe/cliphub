package join

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the video joining command for the unified ClipHub CLI.
func Task(appName string) cl.Task {
	return &task{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*task)(nil)

type task struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *task) Name() string { return "join" }

func (task *task) Synopsis() string {
	return "Join clips from a JSON plan."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Reads a JSON plan and assembles the listed clips into one video.

DEFAULT OUTPUT:

  ride_join.json -> ride_joined.mp4

JOIN JSON:

  [
    {
      "file": "intro.mp4",
      "fade": 0.3
    },
    {
      "file": "curve_slow.mp4",
      "fade": 0.25
    },
    {
      "file": "outro.mp4"
    }
  ]

FIELDS:

  file    required clip path, relative to the JSON file unless absolute
  fade    optional seconds to fade from this clip into the next one

The last clip's fade is ignored. Original audio can be joined with -audio
only when the JSON plan does not use fades.

EXAMPLES:

  cliphub join -in ride_join.json
  cliphub join -in ride_join.json -out ride_final.mp4
  cliphub join -in ride_join.json -dry-run
  
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "join JSON path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_joined.mp4")
	fs.BoolVar(&task.cfg.Audio, "audio", false, "join original audio too; supported only without fades")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without joining")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}
	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := joinVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
