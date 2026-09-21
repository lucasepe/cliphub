# cliphub

`cliphub` is a small command-line toolbox for repeatable social-video workflows.

It does not try to replace a video editor. It covers the small operations that are easy to automate: cut clips, fit videos into social frames, add captions, transcribe speech, reverse or slow short moments, join clips, blur tracked details, and add a soundtrack.

## Prerequisites

- `ffmpeg` in your `PATH`
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
join         Join clips from a JSON plan.
preview      Create a compact video preview under a size limit.
reverse      Reverse a video for rewind-style effects.
slice        Cut a video into clips from a JSON cut list.
slow         Slow a video down for slow-motion effects.
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
```

## Quick Workflow

```sh
cliphub slice -in ride.mp4
cliphub fit -in ride_clip_2.mp4 -cover
cliphub preview -in ride.mp4 -max-size 30MB
cliphub slow -in ride_clip_2_fit.mp4
cliphub reverse -in ride_clip_3.mp4
cliphub trackblur -in ride_clip_2.mp4 -at 00:00:01 -x 640 -y 920 -radius 48
cliphub transcribe -in ride_clip_2_fit_slow.mp4 -model models/ggml-small.bin -lang it
cliphub script -in narration.txt -duration 30
cliphub captions -in ride_clip_2_fit_slow.mp4
cliphub join -in ride_join.json
cliphub soundtrack -in ride_joined.mp4 -audio music.mp3
```

## JSON Conventions

Some commands read sidecar JSON files beside the input media:

```text
captions     ride.mp4      -> ride_overlays.json
slice        ride.mp4      -> ride_slices.json
slow         ride.mp4      -> ride_slow.json (optional selective slow-motion ranges)
join         ride_join.json
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
