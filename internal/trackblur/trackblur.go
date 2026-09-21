package trackblur

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lucasepe/cliphub/internal/shared"
)

type mediaInfo struct {
	Width    int
	Height   int
	FPS      float64
	FPSLabel string
}

type point struct {
	x int
	y int
}

func defaultOutputPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_trackblur", ".mp4")
}

func defaultBlurPath(inputPath string) string {
	return shared.PathWithSuffix(inputPath, "_blur", ".json")
}

func blurTrackedPoint(cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	info, err := probeVideo(cfg.Input)
	if err != nil {
		return err
	}

	items, err := blurItems(cfg)
	if err != nil {
		return err
	}
	trackers, err := buildTrackers(items, info)
	if err != nil {
		return err
	}

	workdir, err := os.MkdirTemp("", "cliphub-trackblur-")
	if err != nil {
		return fmt.Errorf("create frame workspace: %w", err)
	}
	if cfg.KeepWorkdir {
		fmt.Printf("workdir: %s\n", workdir)
	} else {
		defer os.RemoveAll(workdir)
	}

	framesDir := filepath.Join(workdir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return fmt.Errorf("create frames directory: %w", err)
	}

	extract := extractFramesArgs(cfg.Input, filepath.Join(framesDir, "%08d.png"))
	assemble := assembleFramesArgs(framesDir, info.FPSLabel, cfg.Input, cfg.Output)
	if cfg.DryRun {
		shared.PrintCommand("ffmpeg", extract)
		fmt.Printf("# process %d tracked blur region(s); first start frame %d\n", len(trackers), firstStartFrame(trackers))
		shared.PrintCommand("ffmpeg", assemble)
		return nil
	}

	if err := runFFmpeg(extract); err != nil {
		return fmt.Errorf("extract frames: %w", err)
	}

	framePaths, err := filepath.Glob(filepath.Join(framesDir, "*.png"))
	if err != nil {
		return fmt.Errorf("list extracted frames: %w", err)
	}
	if len(framePaths) == 0 {
		return errors.New("ffmpeg did not extract any frames")
	}
	for i, tracker := range trackers {
		if tracker.startFrame >= len(framePaths) {
			return fmt.Errorf("blur %d -at points past the extracted video frames (%d >= %d)", i+1, tracker.startFrame, len(framePaths))
		}
	}

	if err := processFrames(framePaths, trackers, cfg); err != nil {
		return err
	}

	if err := runFFmpeg(assemble); err != nil {
		return fmt.Errorf("assemble %q: %w", cfg.Output, err)
	}

	return nil
}

func validateConfig(cfg Config) error {
	if cfg.Input == "" {
		return errors.New("missing required -in video path")
	}
	if cfg.Output == "" {
		return errors.New("missing required -out video path")
	}
	if cfg.Blurs == "" && (cfg.X < 0 || cfg.Y < 0) {
		return errors.New("missing required -x and -y source-video coordinates")
	}
	if cfg.Radius <= 0 {
		return errors.New("-radius must be positive")
	}
	if cfg.Blur <= 0 {
		return errors.New("-blur must be positive")
	}
	if cfg.SearchRadius <= 0 {
		return errors.New("-search must be positive")
	}
	if cfg.TemplateSize <= 4 {
		return errors.New("-template must be greater than 4")
	}
	if cfg.Adapt < 0 || cfg.Adapt > 1 {
		return errors.New("-adapt must be between 0 and 1")
	}
	if cfg.Shape != "circle" {
		return errors.New("-shape currently supports only circle")
	}

	inAbs, _ := filepath.Abs(cfg.Input)
	outAbs, _ := filepath.Abs(cfg.Output)
	if inAbs == outAbs {
		return errors.New("-out must differ from -in")
	}

	return nil
}

func blurItems(cfg Config) ([]BlurItem, error) {
	if cfg.Blurs != "" {
		return readBlurItems(cfg.Blurs)
	}
	return []BlurItem{{
		At:           cfg.At,
		X:            cfg.X,
		Y:            cfg.Y,
		Radius:       cfg.Radius,
		Blur:         cfg.Blur,
		SearchRadius: cfg.SearchRadius,
		TemplateSize: cfg.TemplateSize,
		Adapt:        cfg.Adapt,
		Shape:        cfg.Shape,
	}}, nil
}

func readBlurItems(path string) ([]BlurItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read blurs %q: %w", path, err)
	}
	var items []BlurItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse blurs %q: %w", path, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("blurs file %q contains no regions", path)
	}
	return items, nil
}

type tracker struct {
	item       BlurItem
	startFrame int
	endFrame   int
	center     point
	template   template
	started    bool
}

func buildTrackers(items []BlurItem, info mediaInfo) ([]tracker, error) {
	trackers := make([]tracker, 0, len(items))
	for i, item := range items {
		item = withDefaults(item)
		if err := validateBlurItem(i, item); err != nil {
			return nil, err
		}
		at, err := parseTimestamp(item.At)
		if err != nil {
			return nil, fmt.Errorf("blur %d at: %w", i+1, err)
		}
		endFrame := math.MaxInt
		if strings.TrimSpace(item.To) != "" {
			to, err := parseTimestamp(item.To)
			if err != nil {
				return nil, fmt.Errorf("blur %d to: %w", i+1, err)
			}
			if to <= at {
				return nil, fmt.Errorf("blur %d to must be greater than at", i+1)
			}
			endFrame = int(math.Round(to * info.FPS))
		}
		trackers = append(trackers, tracker{
			item:       item,
			startFrame: int(math.Round(at * info.FPS)),
			endFrame:   endFrame,
			center:     point{item.X, item.Y},
		})
	}
	return trackers, nil
}

func firstStartFrame(trackers []tracker) int {
	if len(trackers) == 0 {
		return 0
	}
	first := trackers[0].startFrame
	for _, tracker := range trackers[1:] {
		if tracker.startFrame < first {
			first = tracker.startFrame
		}
	}
	return first
}

func withDefaults(item BlurItem) BlurItem {
	if strings.TrimSpace(item.At) == "" {
		item.At = "00:00:00"
	}
	if item.Radius == 0 {
		item.Radius = defaultRadius
	}
	if item.Blur == 0 {
		item.Blur = defaultBlur
	}
	if item.SearchRadius == 0 {
		item.SearchRadius = defaultSearchRadius
	}
	if item.TemplateSize == 0 {
		item.TemplateSize = defaultTemplateSize
	}
	if item.Adapt == 0 {
		item.Adapt = defaultAdapt
	}
	if item.Shape == "" {
		item.Shape = "circle"
	}
	return item
}

func validateBlurItem(index int, item BlurItem) error {
	prefix := fmt.Sprintf("blur %d", index+1)
	if item.X < 0 || item.Y < 0 {
		return fmt.Errorf("%s missing required x and y source-video coordinates", prefix)
	}
	if item.Radius <= 0 {
		return fmt.Errorf("%s radius must be positive", prefix)
	}
	if item.Blur <= 0 {
		return fmt.Errorf("%s blur must be positive", prefix)
	}
	if item.SearchRadius <= 0 {
		return fmt.Errorf("%s search must be positive", prefix)
	}
	if item.TemplateSize <= 4 {
		return fmt.Errorf("%s template must be greater than 4", prefix)
	}
	if item.Adapt < 0 || item.Adapt > 1 {
		return fmt.Errorf("%s adapt must be between 0 and 1", prefix)
	}
	if item.Shape != "circle" {
		return fmt.Errorf("%s shape currently supports only circle", prefix)
	}
	return nil
}

func probeVideo(path string) (mediaInfo, error) {
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,avg_frame_rate",
		"-of", "default=nokey=1:noprint_wrappers=1",
		path,
	}
	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return mediaInfo{}, fmt.Errorf("probe %q with ffprobe: %w", path, err)
	}

	lines := strings.Fields(string(out))
	if len(lines) < 3 {
		return mediaInfo{}, errors.New("ffprobe did not report width, height, and frame rate")
	}
	width, err := strconv.Atoi(lines[0])
	if err != nil {
		return mediaInfo{}, fmt.Errorf("parse video width %q: %w", lines[0], err)
	}
	height, err := strconv.Atoi(lines[1])
	if err != nil {
		return mediaInfo{}, fmt.Errorf("parse video height %q: %w", lines[1], err)
	}
	fps, label, err := parseFrameRate(lines[2])
	if err != nil {
		return mediaInfo{}, err
	}
	if width <= 0 || height <= 0 {
		return mediaInfo{}, errors.New("ffprobe reported invalid video dimensions")
	}

	return mediaInfo{Width: width, Height: height, FPS: fps, FPSLabel: label}, nil
}

func parseFrameRate(value string) (float64, string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0/0" {
		return 0, "", errors.New("ffprobe reported an invalid frame rate")
	}
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		if len(parts) != 2 {
			return 0, "", fmt.Errorf("invalid frame rate %q", value)
		}
		num, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, "", fmt.Errorf("parse frame rate numerator %q: %w", parts[0], err)
		}
		den, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, "", fmt.Errorf("parse frame rate denominator %q: %w", parts[1], err)
		}
		if num <= 0 || den <= 0 {
			return 0, "", fmt.Errorf("invalid frame rate %q", value)
		}
		return num / den, value, nil
	}

	fps, err := strconv.ParseFloat(value, 64)
	if err != nil || fps <= 0 {
		return 0, "", fmt.Errorf("invalid frame rate %q", value)
	}
	return fps, value, nil
}

func parseTimestamp(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if strings.Contains(value, ":") {
		return shared.ParseClockTime(value)
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 {
		return 0, fmt.Errorf("expected seconds or hh:mm:ss.xxx, got %q", value)
	}
	return seconds, nil
}

func extractFramesArgs(input, pattern string) []string {
	return []string{
		"-y",
		"-i", input,
		"-map", "0:v:0",
		"-fps_mode", "passthrough",
		pattern,
	}
}

func assembleFramesArgs(framesDir, fpsLabel, input, output string) []string {
	return []string{
		"-y",
		"-framerate", fpsLabel,
		"-i", filepath.Join(framesDir, "%08d.png"),
		"-i", input,
		"-map", "0:v:0",
		"-map", "1:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "copy",
		"-shortest",
		"-movflags", "+faststart",
		output,
	}
}

func runFFmpeg(args []string) error {
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func processFrames(paths []string, trackers []tracker, cfg Config) error {
	for i := 0; i < len(paths); i++ {
		frame, err := loadNRGBA(paths[i])
		if err != nil {
			return fmt.Errorf("read frame %d: %w", i+1, err)
		}
		output := image.NewNRGBA(frame.Bounds())
		draw.Draw(output, output.Bounds(), frame, frame.Bounds().Min, draw.Src)

		changed := false
		for n := range trackers {
			tracker := &trackers[n]
			if i < tracker.startFrame || i > tracker.endFrame {
				continue
			}
			if !tracker.started {
				tracker.center = clampPoint(tracker.center, frame.Bounds(), tracker.item.TemplateSize/2)
				tracker.template = captureTemplate(frame, tracker.center, tracker.item.TemplateSize)
				tracker.started = true
				if cfg.Debug {
					fmt.Printf("blur=%d frame=%d x=%d y=%d correlation=1.000\n", n+1, i+1, tracker.center.x, tracker.center.y)
				}
			} else {
				correlation := 1.0
				tracker.center, correlation = findBestMatch(frame, tracker.template, tracker.center, tracker.item.SearchRadius)
				tracker.template = updateTemplate(frame, tracker.template, tracker.center, tracker.item.Adapt)
				if cfg.Debug && i%30 == 0 {
					fmt.Printf("blur=%d frame=%d x=%d y=%d correlation=%.3f\n", n+1, i+1, tracker.center.x, tracker.center.y, correlation)
				}
			}
			applyCircularBlur(output, tracker.center, tracker.item.Radius, tracker.item.Blur)
			changed = true
		}

		if changed {
			if err := savePNG(paths[i], output); err != nil {
				return fmt.Errorf("write frame %d: %w", i+1, err)
			}
		}
	}

	return nil
}

func loadNRGBA(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)
	return dst, nil
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func captureTemplate(img *image.NRGBA, center point, size int) template {
	if size%2 == 1 {
		size--
	}
	half := size / 2
	bounds := img.Bounds()
	left := clamp(center.x-half, bounds.Min.X, bounds.Max.X-size)
	top := clamp(center.y-half, bounds.Min.Y, bounds.Max.Y-size)

	values := make([]float64, size*size)
	idx := 0
	for y := top; y < top+size; y++ {
		for x := left; x < left+size; x++ {
			values[idx] = float64(luminance(img.NRGBAAt(x, y)))
			idx++
		}
	}
	tmpl := template{size: size, half: half, values: values}
	tmpl.normalize()
	return tmpl
}

type template struct {
	size   int
	half   int
	values []float64
	denom  float64
}

func (tmpl *template) normalize() {
	if len(tmpl.values) == 0 {
		return
	}
	var sum float64
	for _, value := range tmpl.values {
		sum += value
	}
	mean := sum / float64(len(tmpl.values))
	var sq float64
	for i, value := range tmpl.values {
		centered := value - mean
		tmpl.values[i] = centered
		sq += centered * centered
	}
	tmpl.denom = math.Sqrt(sq)
}

func updateTemplate(img *image.NRGBA, current template, center point, rate float64) template {
	if rate == 0 {
		return current
	}
	next := captureTemplate(img, center, current.size)
	values := make([]float64, len(current.values))
	for i := range current.values {
		values[i] = current.values[i]*(1-rate) + next.values[i]*rate
	}
	updated := template{size: current.size, half: current.half, values: values}
	updated.normalize()
	return updated
}

func findBestMatch(img *image.NRGBA, tmpl template, prev point, searchRadius int) (point, float64) {
	bounds := img.Bounds()
	minX := clamp(prev.x-searchRadius, bounds.Min.X+tmpl.half, bounds.Max.X-tmpl.half)
	maxX := clamp(prev.x+searchRadius, bounds.Min.X+tmpl.half, bounds.Max.X-tmpl.half)
	minY := clamp(prev.y-searchRadius, bounds.Min.Y+tmpl.half, bounds.Max.Y-tmpl.half)
	maxY := clamp(prev.y+searchRadius, bounds.Min.Y+tmpl.half, bounds.Max.Y-tmpl.half)

	best := prev
	bestScore := math.Inf(-1)
	step := 4
	for y := minY; y <= maxY; y += step {
		for x := minX; x <= maxX; x += step {
			score := matchScore(img, tmpl, point{x, y}, step)
			if score > bestScore {
				bestScore = score
				best = point{x, y}
			}
		}
	}

	refined := best
	refinedScore := bestScore
	for y := best.y - step; y <= best.y+step; y++ {
		for x := best.x - step; x <= best.x+step; x++ {
			if x < bounds.Min.X+tmpl.half || x >= bounds.Max.X-tmpl.half || y < bounds.Min.Y+tmpl.half || y >= bounds.Max.Y-tmpl.half {
				continue
			}
			score := matchScore(img, tmpl, point{x, y}, 2)
			if score > refinedScore {
				refinedScore = score
				refined = point{x, y}
			}
		}
	}

	return refined, refinedScore
}

func matchScore(img *image.NRGBA, tmpl template, center point, sampleStep int) float64 {
	if tmpl.denom == 0 {
		return math.Inf(-1)
	}
	left := center.x - tmpl.half
	top := center.y - tmpl.half
	var sum float64
	count := 0
	for y := 0; y < tmpl.size; y += sampleStep {
		for x := 0; x < tmpl.size; x += sampleStep {
			sum += float64(luminance(img.NRGBAAt(left+x, top+y)))
			count++
		}
	}
	mean := sum / float64(count)

	var numerator float64
	var currentSq float64
	for y := 0; y < tmpl.size; y += sampleStep {
		for x := 0; x < tmpl.size; x += sampleStep {
			current := float64(luminance(img.NRGBAAt(left+x, top+y))) - mean
			target := tmpl.values[y*tmpl.size+x]
			numerator += current * target
			currentSq += current * current
		}
	}
	if currentSq == 0 {
		return math.Inf(-1)
	}
	return numerator / (math.Sqrt(currentSq) * tmpl.denom)
}

func applyCircularBlur(img *image.NRGBA, center point, radius int, blurRadius int) {
	bounds := img.Bounds()
	minX := clamp(center.x-radius, bounds.Min.X, bounds.Max.X-1)
	maxX := clamp(center.x+radius, bounds.Min.X, bounds.Max.X-1)
	minY := clamp(center.y-radius, bounds.Min.Y, bounds.Max.Y-1)
	maxY := clamp(center.y+radius, bounds.Min.Y, bounds.Max.Y-1)

	original := image.NewNRGBA(bounds)
	draw.Draw(original, bounds, img, bounds.Min, draw.Src)
	sums := newIntegralImage(original)

	r2 := radius * radius
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := x - center.x
			dy := y - center.y
			if dx*dx+dy*dy > r2 {
				continue
			}
			img.SetNRGBA(x, y, sums.average(x, y, blurRadius))
		}
	}
}

type integralImage struct {
	bounds image.Rectangle
	width  int
	r      []uint64
	g      []uint64
	b      []uint64
	a      []uint64
}

func newIntegralImage(img *image.NRGBA) integralImage {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	stride := width + 1
	size := stride * (height + 1)
	sums := integralImage{
		bounds: bounds,
		width:  stride,
		r:      make([]uint64, size),
		g:      make([]uint64, size),
		b:      make([]uint64, size),
		a:      make([]uint64, size),
	}

	for y := 1; y <= height; y++ {
		var rowR, rowG, rowB, rowA uint64
		for x := 1; x <= width; x++ {
			c := img.NRGBAAt(bounds.Min.X+x-1, bounds.Min.Y+y-1)
			rowR += uint64(c.R)
			rowG += uint64(c.G)
			rowB += uint64(c.B)
			rowA += uint64(c.A)
			idx := y*stride + x
			above := (y-1)*stride + x
			sums.r[idx] = sums.r[above] + rowR
			sums.g[idx] = sums.g[above] + rowG
			sums.b[idx] = sums.b[above] + rowB
			sums.a[idx] = sums.a[above] + rowA
		}
	}

	return sums
}

func (img integralImage) average(x, y, radius int) color.NRGBA {
	minX := clamp(x-radius, img.bounds.Min.X, img.bounds.Max.X-1)
	maxX := clamp(x+radius, img.bounds.Min.X, img.bounds.Max.X-1)
	minY := clamp(y-radius, img.bounds.Min.Y, img.bounds.Max.Y-1)
	maxY := clamp(y+radius, img.bounds.Min.Y, img.bounds.Max.Y-1)

	x1 := minX - img.bounds.Min.X
	x2 := maxX - img.bounds.Min.X + 1
	y1 := minY - img.bounds.Min.Y
	y2 := maxY - img.bounds.Min.Y + 1
	count := uint64((x2 - x1) * (y2 - y1))

	return color.NRGBA{
		R: uint8(img.sum(img.r, x1, y1, x2, y2) / count),
		G: uint8(img.sum(img.g, x1, y1, x2, y2) / count),
		B: uint8(img.sum(img.b, x1, y1, x2, y2) / count),
		A: uint8(img.sum(img.a, x1, y1, x2, y2) / count),
	}
}

func (img integralImage) sum(values []uint64, x1, y1, x2, y2 int) uint64 {
	bottomRight := y2*img.width + x2
	bottomLeft := y2*img.width + x1
	topRight := y1*img.width + x2
	topLeft := y1*img.width + x1
	return values[bottomRight] - values[bottomLeft] - values[topRight] + values[topLeft]
}

func luminance(c color.NRGBA) uint8 {
	return uint8((int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000)
}

func clampPoint(p point, bounds image.Rectangle, margin int) point {
	return point{
		x: clamp(p.x, bounds.Min.X+margin, bounds.Max.X-margin-1),
		y: clamp(p.y, bounds.Min.Y+margin, bounds.Max.Y-margin-1),
	}
}

func clamp(value, low, high int) int {
	if low > high {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
