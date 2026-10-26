# cliphub

`cliphub` is a small command-line toolbox for repeatable social-video workflows.

It does not try to replace a video editor. It covers the small operations that are easy to automate: cut clips, fit videos into social frames, add captions, transcribe speech, reverse or slow short moments, join clips, blur tracked details, and add a soundtrack.

## Prerequisites

- `ffmpeg` in your `PATH`
- `ffmpeg` with `libx264` and common video filters when using `fx`
- `ffprobe` in your `PATH` when using `preview` or `join` with fades
- Optional: `whisper.cpp` CLI when using `transcribe`

On macOS:

```sh
brew install ffmpeg
brew install whisper-cpp
```


## Commands

```text
captions     Render social videos with timed text overlays.
fit          Fit a video into a social-ready frame.
fx           Apply stylized color, ink, or grayscale effects to a video.
join         Join clips from a JSON plan.
preview      Create a compact video preview under a size limit.
reverse      Reverse a video for rewind-style effects.
slice        Cut a video into clips from a JSON cut list.
slow         Slow a video down for slow-motion effects.
soundfx      Mix timed sound effects into a video's original audio.
soundtrack   Add or mix an external audio track into a video.
trackblur    Blur a moving point by tracking it through the video.
script       Generate timed overlay JSON from a text script.
transcribe   Generate timed overlay JSON from speech.
```

Every command has its own detailed help:

```sh
cliphub captions -h
cliphub slice -h
cliphub slow -h
cliphub join -h
cliphub fx -h
```

## Video Effects (`fx`)

`fx` creates a new H.264 MP4 beside the input video and copies its audio when present. The original file is untouched. Choose an effect with `-effect`; `gta` is the default.

| Effect | Appearance | Default output for `ride.mp4` |
| --- | --- | --- |
| `gta` | Saturated, posterized colors and dark outlines | `ride_gta.mp4` |
| `cartoon` | Black ink contours on white | `ride_cartoon.mp4` |
| `cartoon-color` | Color mix with dark contours; often a darker result | `ride_cartoon-color.mp4` |
| `gray-steps` | Five grayscale tones: 0, 50, 100, 150, 200 | `ride_gray-steps.mp4` |
| `cel-shade` | Smooth, flat colors with ink outlines | `ride_cel-shade.mp4` |
| `pixel-art` | 320-pixel-wide picture with a 32-color palette, enlarged to the original frame size | `ride_pixel-art.mp4` |

Run these commands with `cliphub` installed and `ride.mp4` in the current directory:

```sh
cliphub fx -in ride.mp4                    # gta
cliphub fx -in ride.mp4 -effect cartoon
cliphub fx -in ride.mp4 -effect cartoon-color
cliphub fx -in ride.mp4 -effect gray-steps
cliphub fx -in ride.mp4 -effect cel-shade
cliphub fx -in ride.mp4 -effect pixel-art
```

For `pixel-art`, `-pixel-width` controls the working image width before enlargement: a larger value makes smaller pixels and reveals more detail. `-palette-colors` sets the maximum number of colors per frame. The defaults are `320` and `32`; the original, chunkier look used `160` and `16`.

```sh
cliphub fx -in ride.mp4 -effect pixel-art \
  -pixel-width 160 -palette-colors 16 -out ride_pixel-chunky.mp4

cliphub fx -in ride.mp4 -effect pixel-art \
  -pixel-width 240 -palette-colors 24 -out ride_pixel-medium.mp4
```

For `cel-shade`, the default edge thresholds are `-edge-low 0.08 -edge-high 0.15`. Raising them detects fewer contours; it does not make the lines thicker. `-color-mask` controls the number of tones **per RGB channel**:

| Mask | Levels per channel | Look |
| --- | ---: | --- |
| `240` | 16 | More gradual colors |
| `224` (default) | 8 | Balanced flat colors |
| `192` | 4 | Stronger retro posterization |

Save each experiment to a different file with `-out`:

```sh
cliphub fx -in ride.mp4 -effect cel-shade \
  -edge-low 0.11 -edge-high 0.20 -out ride_cel-clean.mp4

cliphub fx -in ride.mp4 -effect cel-shade \
  -color-mask 240 -out ride_cel-smooth.mp4

cliphub fx -in ride.mp4 -effect cel-shade \
  -color-mask 192 -out ride_cel-retro.mp4
```

Use `-dry-run` to print the FFmpeg command without rendering. An existing output file is overwritten when rendering. If you are working from a clone of this repository and have not installed `cliphub`, replace `cliphub` in these examples with `go run .` while in the repository root.

## Quick Workflow

```sh
cliphub slice -in ride.mp4
cliphub fit -in ride_clip_2.mp4 -cover
cliphub fx -in ride_clip_2.mp4
cliphub preview -in ride.mp4 -max-size 30MB
cliphub slow -in ride_clip_2_fit.mp4
cliphub reverse -in ride_clip_3.mp4
cliphub trackblur -in ride_clip_2.mp4 -at 00:00:01 -x 640 -y 920 -radius 48
cliphub trackblur -in ride_clip_2.mp4 -at 00:00:01 -x 640 -y 920 -shape rectangle -width 140 -height 48
cliphub transcribe -in ride_clip_2_fit_slow.mp4 -model models/ggml-small.bin -lang it
cliphub script -in narration.txt -duration 30
cliphub captions -in ride_clip_2_fit_slow.mp4
cliphub join -in ride_join.json
cliphub soundfx -in ride.mp4 -audio reaction.mp3 -at 00:00:08
cliphub soundtrack -in ride_joined.mp4 -audio music.mp3
```

## JSON Conventions

Some commands read sidecar JSON files beside the input media:

```text
captions     ride.mp4      -> ride_overlays.json
slice        ride.mp4      -> ride_slices.json
slow         ride.mp4      -> ride_slow.json (optional selective slow-motion ranges)
join         ride_join.json
soundfx      ride.mp4      -> ride_soundfx.json (optional timed effects)
```

Use the command help to see the expected JSON format:

```sh
cliphub captions -h
cliphub slice -h
cliphub slow -h
cliphub join -h
```

## Emoji Cache

`captions` renders emoji using Twemoji PNG assets. Missing emoji are downloaded on first use and stored in the operating system cache directory.

Override the cache location with:

```sh
CLIPHUB_EMOJI_CACHE="$HOME/.cache/cliphub/emoji" cliphub captions -in input.mp4
```

## How To Install

### Using the _install.sh_ script (macOS & Linux)

Simply run the following command in your terminal:

```sh
curl -sL https://raw.githubusercontent.com/lucasepe/cliphub/main/install.sh | bash
```

To install a specific version:

```sh
curl -sL https://raw.githubusercontent.com/lucasepe/cliphub/main/install.sh | bash -s -- v0.1.0
```

This script will:

- Detect your operating system and architecture
- Download the latest release binary
- Install it into _/usr/local/bin_ when writable
- Fall back to _$HOME/.local/bin_ otherwise
- Remind you to add _$HOME/.local/bin_ to your _PATH_ when needed


### Manually download the latest binaries from the [releases page](https://github.com/lucasepe/cliphub/releases/latest):

- [macOS](https://github.com/lucasepe/cliphub/releases/latest)
- [Windows](https://github.com/lucasepe/cliphub/releases/latest)
- [Linux (arm64)](https://github.com/lucasepe/cliphub/releases/latest)
- [Linux (amd64)](https://github.com/lucasepe/cliphub/releases/latest)

Unpack the binary into any directory that is part of your _PATH_.

## If you have [Go](https://go.dev/dl/) installed

You can also install it using:

```bash
go install github.com/lucasepe/cliphub@latest
```

Make sure your `$GOPATH/bin` is in your PATH to run `cliphub` from anywhere.
