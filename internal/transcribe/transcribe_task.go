package transcribe

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/captions"
	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// TranscribeTask returns the speech-to-overlays command for the unified ClipHub CLI.
func TranscribeTask(appName string) cl.Task {
	return &transcribeTask{ctx: context.Background(), appName: appName}
}

var _ cl.Task = (*transcribeTask)(nil)

type transcribeTask struct {
	appName string
	ctx     context.Context
	cfg     TranscribeConfig
}

func (task *transcribeTask) Name() string { return "transcribe" }

func (task *transcribeTask) Synopsis() string {
	return "Generate timed overlay JSON from speech."
}

func (task *transcribeTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Extracts audio with ffmpeg, runs whisper.cpp, and writes overlay JSON for captions.

DEFAULT OUTPUT:

  ride.mp4 -> ride_overlays.json

OUTPUT JSON:

  [
    {
      "text": "Today we stop for a coffee",
      "gravity": "bottom",
      "start": 4.56,
      "end": 7.92,
      "font_size": 64,
      "box": true
    }
  ]

Use -words to write one overlay item per spoken word instead of one item per
Whisper segment.

EXAMPLES:

  cliphub transcribe -in ride.mp4 -model models/ggml-small.bin -lang it
  cliphub transcribe -in ride.mp4 -model models/ggml-small.bin -words
  cliphub transcribe -in ride.mp4 -model models/ggml-small.bin -out draft.json
  
`,
	})
}

func (task *transcribeTask) Ctx() context.Context { return task.ctx }

func (task *transcribeTask) SetFlags(fs *flag.FlagSet) {
	task.cfg.Whisper = "whisper-cli"
	task.cfg.Language = "it"
	task.cfg.Gravity = "bottom"
	task.cfg.FontSize = defaultFontSize
	task.cfg.PaddingTop = defaultPadding
	task.cfg.PaddingBottom = defaultPadding
	task.cfg.PaddingLeft = defaultPadding
	task.cfg.PaddingRight = defaultPadding
	task.cfg.MaxChars = defaultMaxChars
	task.cfg.Box = true
	task.cfg.BoxAlpha = defaultBoxAlpha

	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output overlays JSON path; defaults to INPUT_BASENAME_overlays.json beside the input video")
	fs.StringVar(&task.cfg.Whisper, "whisper", task.cfg.Whisper, "whisper.cpp CLI path")
	fs.BoolVar(&task.cfg.NoGPU, "no-gpu", false, "disable whisper.cpp GPU acceleration")
	fs.StringVar(&task.cfg.Model, "model", "", "whisper.cpp model path")
	fs.StringVar(&task.cfg.Language, "lang", task.cfg.Language, "spoken language code")
	fs.StringVar(&task.cfg.Gravity, "gravity", task.cfg.Gravity, "overlay gravity: top, center, or bottom")
	fs.Float64Var(&task.cfg.FontSize, "font-size", task.cfg.FontSize, "overlay font size")
	fs.Float64Var(&task.cfg.PaddingTop, "padding-top", task.cfg.PaddingTop, "overlay top padding")
	fs.Float64Var(&task.cfg.PaddingBottom, "padding-bottom", task.cfg.PaddingBottom, "overlay bottom padding")
	fs.Float64Var(&task.cfg.PaddingLeft, "padding-left", task.cfg.PaddingLeft, "overlay left padding")
	fs.Float64Var(&task.cfg.PaddingRight, "padding-right", task.cfg.PaddingRight, "overlay right padding")
	fs.IntVar(&task.cfg.MaxChars, "max-chars", task.cfg.MaxChars, "maximum characters per overlay line")
	fs.BoolVar(&task.cfg.Box, "box", task.cfg.Box, "draw a semi-transparent box behind generated captions")
	fs.Float64Var(&task.cfg.BoxAlpha, "box-alpha", task.cfg.BoxAlpha, "caption box opacity from 0 to 1")
	fs.Float64Var(&task.cfg.BoxPadding, "box-padding", 0, "caption box inner padding")
	fs.Float64Var(&task.cfg.BoxRadius, "box-radius", 0, "caption box corner radius")
	fs.BoolVar(&task.cfg.Words, "words", false, "write one timed overlay per spoken word")
}

func (task *transcribeTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = captions.DefaultOverlaysPath(cfg.Input)
	}
	if err := validateTranscribeConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := transcribeVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
