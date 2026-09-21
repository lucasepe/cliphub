package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lucasepe/cliphub/internal/captions"
	"github.com/lucasepe/cliphub/internal/fit"
	"github.com/lucasepe/cliphub/internal/join"
	"github.com/lucasepe/cliphub/internal/preview"
	"github.com/lucasepe/cliphub/internal/reverse"
	"github.com/lucasepe/cliphub/internal/script"
	"github.com/lucasepe/cliphub/internal/slice"
	"github.com/lucasepe/cliphub/internal/slow"
	"github.com/lucasepe/cliphub/internal/soundtrack"
	"github.com/lucasepe/cliphub/internal/trackblur"
	"github.com/lucasepe/cliphub/internal/transcribe"
	"github.com/lucasepe/x/cl"
)

const (
	appName = "cliphub"
)

var (
	build = "dev"
)

func main() {
	top := flag.NewFlagSet(appName, flag.ContinueOnError)
	top.SetOutput(io.Discard)

	tool := cl.NewTool(top, appName)
	tool.Output = os.Stdout
	tool.Error = os.Stderr
	tool.Header = func(w io.Writer) {
		fmt.Fprintf(w, "┏┓┓•   ┓┏ ┳┳ ┳┓ (build: %s)\n", build)
		fmt.Fprintln(w, "┃ ┃┓┏┓ ┣┫ ┃┃ ┣┫ https://github.com/lucasepe")
		fmt.Fprintln(w, "┗┛┗┗┣┛ ┛┗ ┗┛ ┻┛")
		fmt.Fprintln(w, "    ┛ by Luca Sepe")
		fmt.Fprintln(w)
		fmt.Fprint(w, "Small video automation tools for social clips.\n")
		fmt.Fprintln(w)
	}

	tool.Register(captions.CaptionsTask(appName), "")
	tool.Register(transcribe.TranscribeTask(appName), "")
	tool.Register(script.Task(appName), "")
	tool.Register(fit.Task(appName), "")
	tool.Register(preview.Task(appName), "")
	tool.Register(slice.SliceTask(appName), "")
	tool.Register(reverse.ReverseTask(appName), "")
	tool.Register(slow.SlowTask(appName), "")
	tool.Register(join.Task(appName), "")
	tool.Register(soundtrack.Task(appName), "")
	tool.Register(trackblur.Task(appName), "")

	if err := top.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}

	os.Exit(int(tool.Execute(context.Background())))
}
