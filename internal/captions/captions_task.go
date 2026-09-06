package captions

import (
	"context"
	"flag"

	"github.com/lucasepe/cliphub/internal/shared"
	"github.com/lucasepe/x/cl"
)

// CaptionsTask returns the caption rendering command for the unified ClipHub CLI.
func CaptionsTask(appName string) cl.Task {
	return &captionsTask{
		ctx:     context.Background(),
		appName: appName,
	}
}

var _ cl.Task = (*captionsTask)(nil)

type captionsTask struct {
	appName string
	ctx     context.Context
	cfg     Config
}

func (task *captionsTask) Name() string { return "captions" }

func (task *captionsTask) Synopsis() string {
	return "Render social videos with timed text overlays."
}

func (task *captionsTask) Usage() string {
	return shared.TaskUsage(shared.UsageData{
		Synopsis: task.Synopsis(),
		AppName:  task.appName,
		Command:  task.Name(),
		Body: `
Reads timed text overlays from JSON and renders a captioned video.

OVERLAY FILE LOOKUP:

  If -overlays is not set, captions looks for:

    {CLIP_FILENAME}_overlays.json

  beside the input video.

  Example:

    ride.mp4 -> ride_overlays.json

  If the overlay file does not exist, the video is still rendered without
  text overlays.

DEFAULT OUTPUT:

  If -out is not set, captions writes:

    {CLIP_FILENAME}_captioned.{EXT}

  Example:

    ride.mp4 -> ride_captioned.mp4

OVERLAY JSON:

  [
    {
      "text": "Weekend ride 🏍️🔥",
      "gravity": "bottom",
      "start": 1.5,
      "end": 5.0,
      "font_size": 72,
      "box": true
    }
  ]

REQUIRED JSON FIELDS:

  text     UTF-8 text to render
  start    start time in seconds
  end      end time in seconds

OPTIONAL JSON FIELDS:

  gravity      top, center, or bottom
  font_size    text size in pixels
  safe_area    optional preset: instagram-reel
  padding_top, padding_bottom, padding_left, padding_right
               directional padding in pixels; overrides safe_area
  max_chars    wrapping limit per line
  box          draw a readable background
  box_alpha    background opacity from 0 to 1
  box_padding  inner background padding
  box_radius   background corner radius

EMOJI CACHE:

  Missing Twemoji PNGs are downloaded on first use.
  Override the cache directory with CLIPHUB_EMOJI_CACHE.

EXAMPLES:

  cliphub captions -in ride.mp4
  cliphub captions -in ride.mp4 -cover
  cliphub captions -in ride.mp4 -overlays custom.json -out final.mp4
  cliphub captions -in ride.mp4 -dry-run
  
`,
	})
}

func (task *captionsTask) Ctx() context.Context { return task.ctx }

func (task *captionsTask) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&task.cfg.Input, "in", "", "input video path")
	fs.StringVar(&task.cfg.Output, "out", "", "output video path; defaults to INPUT_BASENAME_captioned beside the input video")
	fs.StringVar(&task.cfg.FontPath, "font", "", "path to a TTF/OTF text font")
	fs.IntVar(&task.cfg.Width, "width", defaultWidth, "output video width")
	fs.IntVar(&task.cfg.Height, "height", defaultHeight, "output video height")
	fs.StringVar(&task.cfg.Overlays, "overlays", "", "JSON file containing timed text overlays; defaults to INPUT_BASENAME_overlays.json beside the input video")
	fs.BoolVar(&task.cfg.Cover, "cover", false, "scale and center-crop the video to fill the output frame")
	fs.BoolVar(&task.cfg.DryRun, "dry-run", false, "print ffmpeg command without rendering")
}

func (task *captionsTask) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) cl.ExitStatus {
	cfg := task.cfg
	if cfg.Output == "" && cfg.Input != "" {
		cfg.Output = defaultCaptionedPath(cfg.Input)
	}
	if cfg.Overlays == "" && cfg.Input != "" {
		cfg.Overlays = DefaultOverlaysPath(cfg.Input)
	}
	if cfg.EmojiCacheDir == "" {
		cacheDir, err := defaultEmojiCacheDir()
		if err != nil {
			return shared.FailTask(err)
		}
		cfg.EmojiCacheDir = cacheDir
	}
	if err := validateConfig(cfg); err != nil {
		return shared.FailTask(err)
	}
	if err := processVideo(cfg); err != nil {
		return shared.FailTask(err)
	}
	return cl.ExitSuccess
}
