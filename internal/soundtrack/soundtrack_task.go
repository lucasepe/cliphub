package soundtrack

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the soundtrack command for the unified ClipHub CLI.
func Task(appName string) cl.Task {
	return &task{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*task)(nil)

type task struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *task) Name() string { return "soundtrack" }

func (task *task) Synopsis() string {
	return "Add or mix an external audio track into a video."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Copies the video stream and writes a new AAC audio mix.

DEFAULT OUTPUT:

  ride_joined.mp4 -> ride_joined_soundtrack.mp4

DEFAULT MIX:

  music volume: 1.0
  video volume: 0.25

Use -video-volume 0 to mute the original video audio. Use -loop when the
external audio is shorter than the video.

EXAMPLES:

  cliphub soundtrack -in ride_joined.mp4 -audio music.mp3
  cliphub soundtrack -in ride_joined.mp4 -audio music.mp3 -video-volume 0
  cliphub soundtrack -in ride_joined.mp4 -audio music.mp3 -music-volume 0.8 -video-volume 0.15
  cliphub soundtrack -in ride_joined.mp4 -audio music.mp3 -loop
  
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	task.cfg.MusicVolume = 1
	task.cfg.VideoVolume = 0.25

	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_soundtrack beside the input video")
	fs.StringVar(&task.cfg.Audio, "audio", "", "external audio file path")
	fs.Float64Var(&task.cfg.MusicVolume, "music-volume", task.cfg.MusicVolume, "external audio volume")
	fs.Float64Var(&task.cfg.VideoVolume, "video-volume", task.cfg.VideoVolume, "original video audio volume; use 0 to mute it")
	fs.BoolVar(&task.cfg.Loop, "loop", false, "loop external audio when it is shorter than the video")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without writing the output")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}
	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := addSoundtrack(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
