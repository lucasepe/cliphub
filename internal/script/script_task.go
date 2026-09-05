package script

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

func Task(appName string) cl.Task {
	return &scriptTask{
		ctx:     context.Background(),
		appName: appName,
	}
}

var _ cl.Task = (*scriptTask)(nil)

type scriptTask struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *scriptTask) Name() string { return "script" }

func (task *scriptTask) Synopsis() string {
	return "Generate timed overlay JSON from a text script."
}

func (task *scriptTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Reads a text file and spreads its contents across the requested duration.
Each non-empty line becomes an overlay. Use -words to create one overlay per
word. Longer text and punctuation receive proportionally more screen time.

All generated overlays use gravity "bottom" and font size 80. If the requested
duration requires more than 200 words per minute, the file is still written and
a readability warning is printed.

DEFAULT OUTPUT:

  narration.txt -> narration_overlays.json

EXAMPLES:

  cliphub script -in narration.txt -duration 30
  cliphub script -in narration.txt -duration 12.5 -words
  cliphub script -in narration.txt -duration 30 -out captions.json
`,
	})
}

func (task *scriptTask) Ctx() context.Context { return task.ctx }

func (task *scriptTask) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input text path")
	fs.StringVar(&task.cfg.Output, "out", "",
		"output overlays JSON path; defaults to INPUT_BASENAME_overlays.json")
	fs.Float64Var(&task.cfg.Duration, "duration", 0,
		"total overlay duration in seconds")
	fs.BoolVar(&task.cfg.Words, "words", false,
		"write one timed overlay per word")
}

func (task *scriptTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		ext := filepath.Ext(cfg.Input)
		cfg.Output = strings.TrimSuffix(cfg.Input, ext) + "_overlays.json"
	}

	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}

	overlays, wordCount, err := generate(cfg)
	if err != nil {
		return shared.FailTask(err)
	}

	if err := writeOverlays(cfg.Output, overlays); err != nil {
		return shared.FailTask(err)
	}

	if wpm := wordsPerMinute(wordCount, cfg.Duration); wpm > readableWPM {
		recommended := float64(wordCount) * 60 / readableWPM
		fmt.Fprintf(os.Stderr, "warning: %.0f words/minute may be difficult to read; use at least %.1f seconds for %d words\n", wpm, recommended, wordCount)
	}

	return cl.ExitSuccess
}
