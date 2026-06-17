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
	return collapseRolling(segs)
}

// collapseRolling merges YouTube auto-caption's rolling-window lines (each line
// repeats the tail of the previous plus a few new words) back into clean,
// reasonably-sized segments.
func collapseRolling(segs []Segment) []Segment {
	out := make([]Segment, 0, len(segs))
	for _, s := range segs {
		if n := len(out); n > 0 {
			if merged, ok := mergeOverlap(out[n-1].Text, s.Text); ok && len(merged) <= 220 {
				out[n-1].Text = merged
				out[n-1].End = s.End
				continue
			}
		}
		out = append(out, s)
	}
	for i := range out {
		out[i].Index = i
	}
	return out
}

// mergeOverlap returns a+b deduplicated when b is a continuation of a (b starts
// with a, or a's word-suffix equals b's word-prefix). ok=false if unrelated.
func mergeOverlap(a, b string) (string, bool) {
	if a == b || strings.HasPrefix(b, a) {
		return b, true
	}
	aw, bw := strings.Fields(a), strings.Fields(b)
	max := len(aw)
	if len(bw) < max {
		max = len(bw)
	}
	for k := max; k >= 2; k-- {
		if strings.Join(aw[len(aw)-k:], " ") == strings.Join(bw[:k], " ") {
			return a + " " + strings.Join(bw[k:], " "), true
		}
	}
	return "", false
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
