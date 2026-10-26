package fx

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

func Task(appName string) cl.Task {
	return &task{appName: appName, ctx: context.Background()}
}

var _ cl.Task = (*task)(nil)

type task struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *task) Name() string { return "fx" }

func (task *task) Synopsis() string {
	return "Apply stylized color, ink, pixel art, or grayscale effects to a video."
}

func (task *task) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Preserves the original audio in every mode.
Requires ffmpeg with libx264 and the gblur, bilateral, eq, lutrgb,
lut, edgedetect, dilation, blend, palettegen and paletteuse filters.

EFFECTS:

  gta            saturated painted look with textured dark outlines (default)
  cartoon        black ink contours on a white background
  cartoon-color  colormix colors with dark detected outlines
  gray-steps     five flat grayscale tones
  cel-shade      smoothed, quantized colors with ink outlines
  pixel-art      320-pixel-wide image with a 32-color palette and sharp pixels

CEL-SHADE TUNING:

  -edge-low/-edge-high  edge detection thresholds; higher values detect fewer edges
  -color-mask           RGB bit mask; 240 = 16, 224 = 8, 192 = 4 levels per channel

PIXEL-ART TUNING:

  -pixel-width          width before pixel enlargement; larger = smaller pixels
  -palette-colors       maximum colors in the per-frame palette

DEFAULT OUTPUT:

  ride.mp4 -> ride_EFFECT.mp4

EXAMPLES:

  cliphub fx -in ride.mp4
  cliphub fx -in ride.mp4 -effect cartoon
  cliphub fx -in ride.mp4 -effect cartoon-color
  cliphub fx -in ride.mp4 -effect gray-steps
  cliphub fx -in ride.mp4 -effect cel-shade
  cliphub fx -in ride.mp4 -effect pixel-art
  cliphub fx -in ride.mp4 -effect pixel-art -pixel-width 160 -palette-colors 16
  cliphub fx -in ride.mp4 -effect cel-shade -edge-low 0.11 -edge-high 0.20
  cliphub fx -in ride.mp4 -effect cel-shade -color-mask 240
  cliphub fx -in ride.mp4 -effect cartoon -out ride_comic.mp4
  cliphub fx -in ride.mp4 -dry-run
`,
	})
}

func (task *task) Ctx() context.Context { return task.ctx }

func (task *task) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output MP4 path; defaults to INPUT_BASENAME_EFFECT.mp4 beside the input")
	fs.StringVar(&task.cfg.Effect, "effect", "gta", "visual effect: gta, cartoon, cartoon-color, gray-steps, cel-shade or pixel-art")
	fs.Float64Var(&task.cfg.EdgeLow, "edge-low", defaultEdgeLow, "cel-shade lower edge threshold")
	fs.Float64Var(&task.cfg.EdgeHigh, "edge-high", defaultEdgeHigh, "cel-shade upper edge threshold")
	fs.IntVar(&task.cfg.ColorMask, "color-mask", defaultColorMask, "cel-shade RGB bit mask (e.g. 192, 224 or 240)")
	fs.IntVar(&task.cfg.PixelWidth, "pixel-width", defaultPixelWidth, "pixel-art width before enlarging to the original frame")
	fs.IntVar(&task.cfg.PaletteColors, "palette-colors", defaultPaletteColors, "pixel-art maximum colors per frame")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without rendering")
}

func (task *task) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultOutputPath(cfg.Input, cfg.Effect)
	}
	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := render(ctx, cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
