package preview

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the compact preview command for the unified ClipHub CLI.
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

func (task *task) Name() string { return "preview" }
func (task *task) Synopsis() string {
	return "Create a compact video preview under a size limit."
}
func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Creates a smaller H.264 copy suitable for messaging and quick sharing.
The original aspect ratio is preserved. A two-pass encode budgets the video
and AAC audio bitrates from the duration and requested maximum file size.

DEFAULT OUTPUT:

  ride.mp4 -> ride_preview.mp4

SIZE VALUES:

  Plain numbers and MB use decimal megabytes. KB, GB, KiB, MiB and GiB are
  also accepted. The default is 30MB.

EXAMPLES:

  cliphub preview -in ride.mp4
  cliphub preview -in ride.mp4 -max-size 30MB
  cliphub preview -in ride.mp4 -max-size 25MiB -out ride_small.mp4
  cliphub preview -in ride.mp4 -max-size 30MB -dry-run
  
`,
	})
}

func (task *task) SetFlags(fs *flag.FlagSet) {
	task.cfg.MaxSize = defaultMaxSize
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output path; defaults to INPUT_BASENAME_preview beside the input video")
	fs.StringVar(&task.cfg.MaxSize, "max-size", task.cfg.MaxSize, "maximum output size, for example 30MB or 25MiB")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "show the planned size, dimensions and ffmpeg commands without encoding")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}

	if err := makePreview(cfg); err != nil {
		return shared.FailTask(err)
	}

	return cl.ExitSuccess
}
