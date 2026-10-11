package soundfx

import (
	"context"
	"flag"
	"os"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// Task returns the soundfx command for the unified ClipHub CLI.
func Task(appName string) cl.Task {
	return &task{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*task)(nil)

type task struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *task) Name() string { return "soundfx" }

func (task *task) Synopsis() string {
	return "Mix timed sound effects into a video's original audio."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Copies the video stream and mixes one or more timed effects into its audio.

SINGLE EFFECT:

  cliphub soundfx -in ride.mp4 -audio reaction.mp3 -at 00:00:08

JSON PLAN:

  [
    {"at":"00:00:08", "audio":"reaction.mp3", "volume":1.0},
    {"at":"00:00:12.500", "audio":"horn.wav", "volume":0.7}
  ]

Relative audio paths are resolved beside the JSON file. The default JSON path
is INPUT_BASENAME_soundfx.json. The original audio remains at full volume by
default; use -video-volume to change it.

EXAMPLES:

  cliphub soundfx -in ride.mp4 -audio reaction.mp3 -at 00:00:08
  cliphub soundfx -in ride.mp4 -sounds ride_soundfx.json
  cliphub soundfx -in ride.mp4 -sounds ride_soundfx.json -video-volume 0.8
  cliphub soundfx -in ride.mp4 -sounds ride_soundfx.json -dry-run
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	task.cfg.Volume = 1
	task.cfg.VideoVolume = 1

	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output path; defaults to INPUT_BASENAME_soundfx beside the video")
	fs.StringVar(&task.cfg.Sounds, "sounds", "", "JSON sound-effects plan; defaults to INPUT_BASENAME_soundfx.json when present")
	fs.StringVar(&task.cfg.Audio, "audio", "", "single sound-effect file")
	fs.StringVar(&task.cfg.At, "at", "", "single effect start time as hh:mm:ss[.xxx]")
	fs.Float64Var(&task.cfg.Volume, "volume", task.cfg.Volume, "single sound-effect volume")
	fs.Float64Var(&task.cfg.VideoVolume, "video-volume", task.cfg.VideoVolume, "original video audio volume")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without writing the output")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input)
	}
	if cfg.Sounds == "" && cfg.Audio == "" && cfg.Input != "" {
		candidate := defaultSoundsPath(cfg.Input)
		if _, err := os.Stat(candidate); err == nil {
			cfg.Sounds = candidate
		}
	}
	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := addSoundEffects(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
