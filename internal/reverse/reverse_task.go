package reverse

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// ReverseTask returns the video reversing command for the unified ClipHub CLI.
func ReverseTask(appName string) cl.Task {
	return &reverseTask{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*reverseTask)(nil)

type reverseTask struct {
	appName string
	ctx     context.Context
	cfg     ReverseConfig
}

func (task *reverseTask) Name() string { return "reverse" }

func (task *reverseTask) Synopsis() string {
	return "Reverse a video for rewind-style effects."
}

func (task *reverseTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Reverses the video stream and writes a new H.264 MP4 file.

DEFAULT OUTPUT:

  throw.mp4 -> throw_reverse.mp4

By default audio is dropped. Use -audio to reverse audio too.
This is best for short clips because ffmpeg buffers frames internally.

EXAMPLES:

  cliphub reverse -in throw.mp4
  cliphub reverse -in throw.mp4 -audio
  cliphub reverse -in throw.mp4 -out force_pull.mp4
  cliphub reverse -in throw.mp4 -dry-run
  
`,
	})
}

func (task *reverseTask) Ctx() context.Context { return task.ctx }

func (task *reverseTask) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_reverse beside the input video")
	fs.BoolVar(&task.cfg.Audio, "audio", false, "reverse audio too when an audio stream is present")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without reversing")
}

func (task *reverseTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultReversePath(cfg.Input)
	}
	if err := validateReverseConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := reverseVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
