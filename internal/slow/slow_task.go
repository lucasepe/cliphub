package slow

import (
	"context"
	"flag"
	"os"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// SlowTask returns the slow-motion command for the unified ClipHub CLI.
func SlowTask(appName string) cl.Task {
	return &slowTask{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*slowTask)(nil)

type slowTask struct {
	appName string
	ctx     context.Context
	cfg     SlowConfig
}

func (task *slowTask) Name() string { return "slow" }

func (task *slowTask) Synopsis() string {
	return "Slow a video down for slow-motion effects."
}

func (task *slowTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Changes video presentation timestamps to create a simple slow-motion effect.

DEFAULT OUTPUT:

  turn.mp4 -> turn_slow.mp4

The default factor is 2, meaning half speed and double duration.
By default audio is dropped. Use -audio to slow audio too.

SELECTIVE SLOW MOTION:

If INPUT_BASENAME_slow.json exists, only its source-timeline ranges are slowed:

  [
    {"from": "00:00:03.500", "to": "00:00:07"},
    {"from": "00:01:12", "to": "00:01:15.250"}
  ]

Use -cuts to select another JSON file. If the default file does not exist, the
whole video is slowed as before. Explicitly requested files must exist.

EXAMPLES:

  cliphub slow -in turn.mp4
  cliphub slow -in turn.mp4 -factor 2.5
  cliphub slow -in turn.mp4 -factor 2.5 -audio
  cliphub slow -in turn.mp4 -cuts highlights.json
  cliphub slow -in turn.mp4 -dry-run
  
`,
	})
}

func (task *slowTask) Ctx() context.Context { return task.ctx }

func (task *slowTask) SetFlags(fs *flag.FlagSet) {
	task.cfg.Factor = 2

	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_slow beside the input video")
	fs.StringVar(&task.cfg.Cuts, "cuts", "", "JSON file containing ranges to slow; auto-detects INPUT_BASENAME_slow.json")
	fs.Float64Var(&task.cfg.Factor, "factor", task.cfg.Factor, "slowdown factor; 2 means half speed")
	fs.BoolVar(&task.cfg.Audio, "audio", false, "slow audio too when an audio stream is present")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without slowing")
}

func (task *slowTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultSlowPath(cfg.Input)
	}
	if cfg.Cuts == "" && cfg.Input != "" {
		cuts := defaultSlowCutsPath(cfg.Input)
		if _, err := os.Stat(cuts); err == nil {
			cfg.Cuts = cuts
		} else if !os.IsNotExist(err) {
			return shared.FailTask(err)
		}
	}
	if err := validateSlowConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := slowVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
