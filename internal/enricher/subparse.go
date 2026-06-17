package enricher

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	cueLineRe  = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}[.,]\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}[.,]\d{3})`)
	tagRe      = regexp.MustCompile(`<[^>]*>`)        // inline <00:00:..> / <c> tags
	htmlEntity = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&nbsp;", " ", "&#39;", "'", "&quot;", `"`)
)

// parseSubtitles parses VTT or SRT content into timestamped segments, stripping
// inline tags/cue settings and collapsing the rolling duplicates that YouTube
// auto-captions produce.
func parseSubtitles(content string) []Segment {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var segs []Segment
	i := 0
	for i < len(lines) {
		m := cueLineRe.FindStringSubmatch(lines[i])
		if m == nil {
			i++
			continue
		}
		start, end := parseTS(m[1]), parseTS(m[2])
		i++
		var text []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			clean := strings.TrimSpace(htmlEntity.Replace(tagRe.ReplaceAllString(lines[i], "")))
			if clean != "" {
				text = append(text, clean)
			}
			i++
		}
		joined := strings.Join(text, " ")
		if joined == "" {
			continue
		}
		// Drop rolling duplicates: auto-captions repeat the previous line.
		if n := len(segs); n > 0 && segs[n-1].Text == joined {
			segs[n-1].End = end
			continue
		}
		segs = append(segs, Segment{Index: len(segs), Start: start, End: end, Text: joined})
	}
	return segs
}

// parseTS converts "HH:MM:SS.mmm" or "HH:MM:SS,mmm" to seconds.
func parseTS(ts string) float64 {
	ts = strings.Replace(ts, ",", ".", 1)
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.ParseFloat(parts[0], 64)
	m, _ := strconv.ParseFloat(parts[1], 64)
	s, _ := strconv.ParseFloat(parts[2], 64)
	return h*3600 + m*60 + s
}

// textAround returns the transcript spoken within [t-window, t+window] seconds.
func textAround(segs []Segment, t, window float64) string {
	var parts []string
	for _, s := range segs {
		if s.End >= t-window && s.Start <= t+window {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, " ")
}
