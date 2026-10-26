package fx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lucasepe/cliphub/internal/shared"
)

type Config struct {
	Input         string
	Output        string
	Effect        string
	EdgeLow       float64
	EdgeHigh      float64
	ColorMask     int
	PixelWidth    int
	PaletteColors int
	DryRun        bool
}

const (
	defaultEdgeLow       = 0.08
	defaultEdgeHigh      = 0.15
	defaultColorMask     = 224
	defaultPixelWidth    = 320
	defaultPaletteColors = 32
)

func defaultOutputPath(input, effect string) string {
	return shared.PathWithSuffix(input, "_"+effect, ".mp4")
}

func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	if cfg.Effect != "gta" && cfg.Effect != "cartoon" && cfg.Effect != "cartoon-color" && cfg.Effect != "gray-steps" && cfg.Effect != "cel-shade" && cfg.Effect != "pixel-art" {
		return fmt.Errorf("unknown effect %q (choose gta, cartoon, cartoon-color, gray-steps, cel-shade or pixel-art)", cfg.Effect)
	}
	if cfg.Effect == "cel-shade" {
		if cfg.EdgeLow < 0 || cfg.EdgeHigh > 1 || cfg.EdgeLow >= cfg.EdgeHigh {
			return errors.New("-edge-low and -edge-high must satisfy 0 <= low < high <= 1")
		}
		if cfg.ColorMask < 0 || cfg.ColorMask > 255 {
			return errors.New("-color-mask must be between 0 and 255")
		}
	} else if cfg.EdgeLow != defaultEdgeLow || cfg.EdgeHigh != defaultEdgeHigh || cfg.ColorMask != defaultColorMask {
		return errors.New("-edge-low, -edge-high and -color-mask are only available with -effect cel-shade")
	}
	if cfg.Effect == "pixel-art" {
		if cfg.PixelWidth < 16 || cfg.PixelWidth > 3840 {
			return errors.New("-pixel-width must be between 16 and 3840")
		}
		if cfg.PaletteColors < 2 || cfg.PaletteColors > 256 {
			return errors.New("-palette-colors must be between 2 and 256")
		}
	} else if cfg.PixelWidth != defaultPixelWidth || cfg.PaletteColors != defaultPaletteColors {
		return errors.New("-pixel-width and -palette-colors are only available with -effect pixel-art")
	}
	in, err := filepath.Abs(cfg.Input)
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	out, err := filepath.Abs(cfg.Output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if in == out {
		return errors.New("input and output must be different files")
	}
	return nil
}

// gtaFilter gives video a saturated, painted look with simplified tones and dark outlines.
const gtaFilter = "[0:v:0]format=rgb24,split[base][edge];" +
	"[base]gblur=sigma=1.2,eq=contrast=1.12:saturation=1.5:brightness=0.025," +
	"lutrgb=r=trunc(val/24)*24+12:g=trunc(val/24)*24+12:b=trunc(val/24)*24+12," +
	"format=rgb24[paint];" +
	"[edge]gblur=sigma=1.2,edgedetect=low=0.08:high=0.2,negate,format=rgb24[lines];" +
	"[paint][lines]blend=all_mode=multiply:all_opacity=0.55,format=rgb24[vout]"

// cartoonFilter draws thick black ink contours on a white background.
const cartoonFilter = "[0:v:0]format=gray," +
	"bilateral=sigmaS=4:sigmaR=0.15:planes=1," +
	"edgedetect=low=0.06:high=0.16:mode=wires," +
	"dilation,negate,lut=y=if(gte(val\\,180)\\,255\\,0),format=rgb24[vout]"

// cartoonColorFilter combines colormix colors with dark detected contours.
// RGB formats keep the monochrome edge branch from making the output grayscale.
const cartoonColorFilter = "[0:v:0]format=rgb24,split[edge_src][color_src];" +
	"[edge_src]edgedetect=low=0.1:high=0.2,format=gray,negate,format=rgb24[edges];" +
	"[color_src]edgedetect=mode=colormix:high=0.05,format=rgb24[colors];" +
	"[colors][edges]blend=all_mode=multiply,format=rgb24[vout]"

// grayStepsFilter maps luma to the same five values as the nested geq recipe.
// A lookup table evaluates the expression once per level instead of per pixel.
const grayStepsFilter = "[0:v:0]format=gray," +
	"lut=y=min(200\\,max(0\\,floor((val-1)/50)*50)),format=rgb24[vout]"

// pixelArtFilter builds a small palette per frame, then scales the
// result back to the input dimensions with nearest-neighbor interpolation.
func pixelArtFilter(cfg Config) string {
	return fmt.Sprintf(
		"[0:v:0]split[work][ref];"+
			"[work]scale=%d:-2:flags=area,split[small][palette_src];"+
			"[palette_src]palettegen=max_colors=%d:stats_mode=single:reserve_transparent=0[pal];"+
			"[small][pal]paletteuse=dither=none:new=1[limited];"+
			"[limited][ref]scale=w=ref_w:h=ref_h:flags=neighbor,format=rgb24[vout]",
		cfg.PixelWidth, cfg.PaletteColors,
	)
}

// celShadeFilter quantizes smoothed RGB channels, then multiplies in ink edges.
func celShadeFilter(cfg Config) string {
	return fmt.Sprintf(
		"[0:v:0]format=rgb24,split[base][edge];"+
			"[base]bilateral=sigmaS=5:sigmaR=0.1:planes=7,"+
			"lutrgb=r=bitand(val\\,%d):g=bitand(val\\,%d):b=bitand(val\\,%d),"+
			"format=rgb24[flat];"+
			"[edge]edgedetect=low=%g:high=%g,format=gray,negate,format=rgb24[ink];"+
			"[flat][ink]blend=all_mode=multiply,format=rgb24[vout]",
		cfg.ColorMask, cfg.ColorMask, cfg.ColorMask, cfg.EdgeLow, cfg.EdgeHigh,
	)
}

func ffmpegArgs(cfg Config) []string {
	filter := gtaFilter
	switch cfg.Effect {
	case "cartoon":
		filter = cartoonFilter
	case "cartoon-color":
		filter = cartoonColorFilter
	case "gray-steps":
		filter = grayStepsFilter
	case "cel-shade":
		filter = celShadeFilter(cfg)
	case "pixel-art":
		filter = pixelArtFilter(cfg)
	}
	args := []string{"-y"}
	if cfg.Effect == "pixel-art" {
		// palettegen emits a log line per frame; keep progress and errors visible.
		args = append(args, "-hide_banner", "-loglevel", "error", "-stats")
	}
	args = append(args,
		"-i", cfg.Input,
		"-filter_complex", filter,
		"-map", "[vout]",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "medium",
		"-crf", "20",
	)
	if cfg.Effect == "cel-shade" {
		args = append(args, "-tune", "animation")
	}
	args = append(args,
		"-c:a", "copy",
		"-movflags", "+faststart",
		cfg.Output,
	)
	return args
}

func render(ctx context.Context, cfg Config) error {
	args := ffmpegArgs(cfg)
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", args)
		return nil
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apply %s effect to %q: %w", cfg.Effect, cfg.Output, err)
	}
	return nil
}
