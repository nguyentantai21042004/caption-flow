package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var ptsTimeRe = regexp.MustCompile(`pts_time:([0-9.]+)`)

type ytMeta struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
}

// Enrich runs the full URL -> bundle pipeline. See interface doc.
func (e *implEnricher) Enrich(ctx context.Context, videoURL string) (*Bundle, error) {
	meta, err := e.metadata(ctx, videoURL)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}

	dir := filepath.Join(e.cfg.OutputDir, meta.ID)
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir bundle: %w", err)
	}
	e.log.Info(ctx, "Enriching %q (%s) -> %s", meta.Title, meta.ID, dir)

	videoPath, err := e.download(ctx, videoURL, dir)
	if err != nil {
		return nil, fmt.Errorf("download video: %w", err)
	}

	// Captions are best-effort: YouTube rate-limits the timedtext endpoint (429),
	// so a failure here must not lose the frames we can still extract.
	segments, err := e.captions(ctx, videoURL, dir)
	if err != nil {
		e.log.Warn(ctx, "Captions unavailable (%v) — continuing without transcript", err)
		segments = nil
	}
	e.log.Info(ctx, "Captions: %d segments", len(segments))

	times, err := e.keyframes(ctx, videoPath, dir, framesDir)
	if err != nil {
		return nil, fmt.Errorf("keyframes: %w", err)
	}
	e.log.Info(ctx, "Keyframes: %d", len(times))

	frames := e.mapFrames(framesDir, dir, times, segments)

	bundle := &Bundle{
		VideoID:  meta.ID,
		Title:    meta.Title,
		URL:      videoURL,
		Duration: meta.Duration,
		Dir:      dir,
		Segments: segments,
		Frames:   frames,
	}
	if err := e.writeArtifacts(bundle, dir); err != nil {
		return nil, fmt.Errorf("write artifacts: %w", err)
	}
	if !e.cfg.KeepVideo {
		_ = os.Remove(videoPath)
	}
	return bundle, nil
}

func (e *implEnricher) metadata(ctx context.Context, url string) (*ytMeta, error) {
	args := append(e.cookieArgs(), "-J", "--skip-download", "--no-warnings", url)
	out, err := e.exec.Execute(ctx, "yt-dlp", args...)
	if err != nil {
		return nil, err
	}
	var m ytMeta
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		return nil, fmt.Errorf("parse yt-dlp json: %w", err)
	}
	if m.ID == "" {
		return nil, fmt.Errorf("no video id resolved")
	}
	return &m, nil
}

// captions downloads subtitles (manual + auto) and parses them into segments.
func (e *implEnricher) captions(ctx context.Context, url, dir string) ([]Segment, error) {
	args := append(e.cookieArgs(),
		"--skip-download", "--write-subs", "--write-auto-subs",
		"--sub-langs", e.cfg.SubLangs, "--sub-format", "vtt", "--convert-subs", "vtt",
		// Default player_client serves subtitles best; a forced android/tv client
		// breaks the timedtext fetch. Sleep + retries to avoid tripping 429.
		"--retries", "5", "--retry-sleep", "3", "--sleep-requests", "1", "--sleep-subtitles", "1",
		"--no-warnings", "-o", filepath.Join(dir, "subs.%(ext)s"), url,
	)
	// yt-dlp can exit non-zero when ONE language 429s; that must not discard the
	// languages that DID download. Decide by what actually landed on disk.
	_, runErr := e.exec.Execute(ctx, "yt-dlp", args...)
	matches, _ := filepath.Glob(filepath.Join(dir, "subs*.vtt"))
	if len(matches) == 0 {
		if runErr != nil {
			return nil, runErr
		}
		return nil, fmt.Errorf("no subtitle file produced (video may have no captions)")
	}
	// Prefer the original Vietnamese track (most faithful to the lecturer's
	// speech), then any vi, then en, else first available.
	pick := matches[0]
	for _, pref := range []string{".vi-orig", ".vi", ".en"} {
		found := false
		for _, m := range matches {
			if strings.Contains(m, pref) {
				pick = m
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	content, err := os.ReadFile(pick)
	if err != nil {
		return nil, err
	}
	return parseSubtitles(string(content)), nil
}

func (e *implEnricher) download(ctx context.Context, url, dir string) (string, error) {
	args := append(e.cookieArgs(),
		"-f", e.cfg.VideoFormat, "--no-warnings",
		"-o", filepath.Join(dir, "video.%(ext)s"), url,
	)
	_, err := e.exec.Execute(ctx, "yt-dlp", args...)
	if err != nil {
		return "", err
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "video.*"))
	for _, m := range matches {
		if !strings.HasSuffix(m, ".vtt") {
			return m, nil
		}
	}
	return "", fmt.Errorf("downloaded video not found")
}

// keyframes extracts scene-change frames and returns their timestamps (seconds),
// falling back to a fixed interval if scene detection yields nothing.
func (e *implEnricher) keyframes(ctx context.Context, videoPath, dir, framesDir string) ([]float64, error) {
	timesFile := filepath.Join(dir, "frametimes.txt")
	vf := fmt.Sprintf("select='eq(n\\,0)+gt(scene\\,%g)',metadata=print:file=%s,scale=%d:-2",
		e.cfg.SceneThreshold, timesFile, e.cfg.FrameWidth)
	_, err := e.exec.Execute(ctx, "ffmpeg", "-y", "-i", videoPath,
		"-vf", vf, "-vsync", "vfr", "-q:v", "3", filepath.Join(framesDir, "%04d.jpg"))
	if err != nil {
		return nil, err
	}
	times := parsePtsTimes(timesFile)
	if len(times) < 5 {
		// Too few scene cuts (static slides / gradual fades) — sample every 30s.
		e.log.Warn(ctx, "Only %d scene frames; sampling every 30s instead", len(times))
		if err := os.RemoveAll(framesDir); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(framesDir, 0o755); err != nil {
			return nil, err
		}
		vf = fmt.Sprintf("fps=1/30,scale=%d:-2", e.cfg.FrameWidth)
		if _, err := e.exec.Execute(ctx, "ffmpeg", "-y", "-i", videoPath,
			"-vf", vf, "-q:v", "3", filepath.Join(framesDir, "%04d.jpg")); err != nil {
			return nil, err
		}
		jpgs, _ := filepath.Glob(filepath.Join(framesDir, "*.jpg"))
		for i := range jpgs {
			times = append(times, float64(i*30))
		}
	}
	return times, nil
}

func (e *implEnricher) mapFrames(framesDir, baseDir string, times []float64, segs []Segment) []Frame {
	jpgs, _ := filepath.Glob(filepath.Join(framesDir, "*.jpg"))
	sort.Strings(jpgs)
	n := len(jpgs)
	if len(times) < n {
		n = len(times)
	}
	frames := make([]Frame, 0, n)
	for i := 0; i < n; i++ {
		rel, _ := filepath.Rel(baseDir, jpgs[i])
		frames = append(frames, Frame{
			File:        rel,
			T:           times[i],
			NearestText: textAround(segs, times[i], 6.0),
		})
	}
	return frames
}

// writeArtifacts persists manifest.json and a plain-text transcript.
func (e *implEnricher) writeArtifacts(b *Bundle, dir string) error {
	manifest, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o644); err != nil {
		return err
	}
	var tb strings.Builder
	for _, s := range b.Segments {
		tb.WriteString(fmt.Sprintf("[%s] %s\n", clock(s.Start), s.Text))
	}
	return os.WriteFile(filepath.Join(dir, "transcript.txt"), []byte(tb.String()), 0o644)
}

func parsePtsTimes(file string) []float64 {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var out []float64
	for _, m := range ptsTimeRe.FindAllStringSubmatch(string(data), -1) {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func clock(sec float64) string {
	s := int(sec)
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// cookieArgs injects browser cookies into yt-dlp when configured, which is the
// reliable way around YouTube 429 / bot checks. Empty -> no cookies.
func (e *implEnricher) cookieArgs() []string {
	if e.cfg.CookiesBrowser == "" {
		return nil
	}
	return []string{"--cookies-from-browser", e.cfg.CookiesBrowser}
}
