package summarizer

import (
	"regexp"
	"strings"

	"github.com/gomutex/godocx"
	"github.com/gomutex/godocx/docx"
)

const (
	fontName = "Times New Roman"
	fontSize = 13
)

var (
	reHeading  = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	reBold     = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBullet   = regexp.MustCompile(`^[\-\*]\s+(.+)$`)
	reNumberd  = regexp.MustCompile(`^\d+\.\s+(.+)$`)
	reSrtTime  = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`)
	reSrtIndex = regexp.MustCompile(`^\d+$`)
)

// markdownToDocx converts markdown text to a styled docx file.
func markdownToDocx(title, markdown, outputPath string) error {
	doc, err := godocx.NewDocument()
	if err != nil {
		return err
	}

	addStyledRun(doc.AddParagraph(""), title, true, 16)

	lines := strings.Split(markdown, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || trimmed == "---" {
			continue
		}

		if m := reHeading.FindStringSubmatch(trimmed); m != nil {
			level := len(m[1])
			size := headingSize(level)
			p := doc.AddParagraph("")
			addStyledRun(p, m[2], true, size)
			continue
		}

		if m := reBullet.FindStringSubmatch(trimmed); m != nil {
			p := doc.AddParagraph("")
			addRichText(p, "• "+m[1])
			continue
		}

		if m := reNumberd.FindStringSubmatch(trimmed); m != nil {
			p := doc.AddParagraph("")
			addRichText(p, trimmed)
			continue
		}

		p := doc.AddParagraph("")
		addRichText(p, trimmed)
	}

	return doc.SaveTo(outputPath)
}

// srtToDocx converts raw SRT subtitle content to a clean transcript docx.
// Strips sequence numbers and timestamps, keeps only dialogue text.
func srtToDocx(title, srtContent, outputPath string) error {
	doc, err := godocx.NewDocument()
	if err != nil {
		return err
	}

	addStyledRun(doc.AddParagraph(""), title, true, 16)
	doc.AddParagraph("")

	lines := strings.Split(srtContent, "\n")
	var textLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || reSrtIndex.MatchString(trimmed) || reSrtTime.MatchString(trimmed) {
			continue
		}
		textLines = append(textLines, trimmed)
	}

	// Group into paragraphs: merge consecutive text lines, split on duplicates
	seen := make(map[string]bool)
	for _, t := range textLines {
		if seen[t] {
			continue
		}
		seen[t] = true
		p := doc.AddParagraph("")
		p.AddText(t).Font(fontName).Size(fontSize).Color("000000")
	}

	return doc.SaveTo(outputPath)
}

func headingSize(level int) uint64 {
	switch level {
	case 1:
		return 16
	case 2:
		return 15
	case 3:
		return 14
	default:
		return fontSize
	}
}

func addStyledRun(p *docx.Paragraph, text string, bold bool, size uint64) {
	text = cleanMarkdownInline(text)
	run := p.AddText(text).Font(fontName).Size(size).Color("000000")
	if bold {
		run.Bold(true)
	}
}

func addRichText(p *docx.Paragraph, text string) {
	parts := reBold.Split(text, -1)
	matches := reBold.FindAllStringSubmatch(text, -1)

	for i, part := range parts {
		if part != "" {
			clean := cleanMarkdownInline(part)
			p.AddText(clean).Font(fontName).Size(fontSize).Color("000000")
		}
		if i < len(matches) {
			clean := cleanMarkdownInline(matches[i][1])
			p.AddText(clean).Font(fontName).Size(fontSize).Color("000000").Bold(true)
		}
	}
}

func cleanMarkdownInline(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	return s
}
