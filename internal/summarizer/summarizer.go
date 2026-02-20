package summarizer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/genai"
)

const summaryPrompt = `Bạn là một chuyên gia phân tích nội dung video đào tạo. Dựa trên phụ đề bên dưới, hãy viết một bản tóm tắt CHI TIẾT bằng TIẾNG VIỆT.

Yêu cầu:
- Bắt đầu bằng tiêu đề tổng quan (1 câu) mô tả chủ đề video
- Liệt kê TẤT CẢ các bước / nội dung chính theo thứ tự xuất hiện
- Giải thích chi tiết từng bước, bao gồm các lưu ý, mẹo, cảnh báo quan trọng
- Nếu có thuật ngữ chuyên ngành, giữ nguyên thuật ngữ tiếng Anh trong ngoặc
- Sử dụng format markdown: heading, bullet points, bold cho từ khóa quan trọng
- Cuối cùng thêm phần "Lưu ý quan trọng" nếu có thông tin cần nhấn mạnh

Phụ đề video:
---
%s
---`

// SummarizeAll discovers SRT files in outputDir (root), then for each:
//   - writes transcript docx to outputDir/transcripts/
//   - calls Gemini and writes summary docx to outputDir/summaries/
//   - moves the processed SRT to outputDir/archived/
func (s *implSummarizer) SummarizeAll(ctx context.Context, outputDir string) error {
	srtFiles, err := s.discoverSRTFiles(outputDir)
	if err != nil {
		return fmt.Errorf("discover SRT files: %w", err)
	}

	if len(srtFiles) == 0 {
		s.logger.Info(ctx, "No SRT files found in %s", outputDir)
		return nil
	}

	transcriptsDir := filepath.Join(outputDir, "transcripts")
	summariesDir := filepath.Join(outputDir, "summaries")
	archivedDir := filepath.Join(outputDir, "archived")

	for _, dir := range []string{transcriptsDir, summariesDir, archivedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	s.logger.Info(ctx, "Found %d SRT files to process", len(srtFiles))
	s.logger.Info(ctx, "  Transcripts -> %s", transcriptsDir)
	s.logger.Info(ctx, "  Summaries   -> %s", summariesDir)
	s.logger.Info(ctx, "  Archived    -> %s", archivedDir)

	successCount := 0
	failCount := 0

	for i, srtPath := range srtFiles {
		videoName := strings.TrimSuffix(filepath.Base(srtPath), ".srt")
		s.logger.Info(ctx, "[%d/%d] Processing: %s", i+1, len(srtFiles), videoName)

		content, err := os.ReadFile(srtPath)
		if err != nil {
			s.logger.Error(ctx, "Failed to read %s: %v", srtPath, err)
			failCount++
			continue
		}
		srtText := string(content)

		// 1) Transcript DOCX — raw SRT content formatted as docx
		txDocx := filepath.Join(transcriptsDir, videoName+".docx")
		if err := srtToDocx(videoName, srtText, txDocx); err != nil {
			s.logger.Error(ctx, "Failed to write transcript %s: %v", txDocx, err)
			failCount++
			continue
		}
		s.logger.Info(ctx, "  ✓ Transcript: %s", txDocx)

		// 2) Summary DOCX — LLM-generated summary
		summary, err := s.callGemini(ctx, srtText)
		if err != nil {
			s.logger.Error(ctx, "Failed to summarize %s: %v", videoName, err)
			failCount++
			continue
		}

		sumDocx := filepath.Join(summariesDir, videoName+".docx")
		if err := markdownToDocx(videoName, strings.TrimSpace(summary), sumDocx); err != nil {
			s.logger.Error(ctx, "Failed to write summary %s: %v", sumDocx, err)
			failCount++
			continue
		}
		s.logger.Info(ctx, "  ✓ Summary:    %s", sumDocx)

		// 3) Archive — move processed SRT so it won't be re-processed
		srtDest := filepath.Join(archivedDir, filepath.Base(srtPath))
		if err := os.Rename(srtPath, srtDest); err != nil {
			s.logger.Warn(ctx, "Failed to archive SRT %s: %v", srtPath, err)
		}

		s.logger.Info(ctx, "[DONE] %s", videoName)
		successCount++
	}

	s.logger.Info(ctx, "Processing complete: %d success, %d failed", successCount, failCount)
	return nil
}

// callGemini sends the transcript to Gemini and returns the summary text.
// Rotates API keys on 429 / quota errors.
func (s *implSummarizer) callGemini(ctx context.Context, transcript string) (string, error) {
	prompt := fmt.Sprintf(summaryPrompt, transcript)

	attempts := len(s.apiKeys)
	var lastErr error

	for range attempts {
		key := s.apiKeys[s.currentKey]

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  key,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			lastErr = fmt.Errorf("create client: %w", err)
			s.rotateKey()
			continue
		}

		result, err := client.Models.GenerateContent(ctx, s.model, genai.Text(prompt), nil)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "RESOURCE_EXHAUSTED") {
				s.logger.Warn(ctx, "Key %d rate limited, rotating...", s.currentKey+1)
				s.rotateKey()
				lastErr = err
				continue
			}
			return "", fmt.Errorf("generate content: %w", err)
		}

		if result != nil && len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
			var text string
			for _, part := range result.Candidates[0].Content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}
			return text, nil
		}

		return "", fmt.Errorf("empty response from Gemini")
	}

	return "", fmt.Errorf("all API keys exhausted: %w", lastErr)
}

func (s *implSummarizer) rotateKey() {
	s.currentKey = (s.currentKey + 1) % len(s.apiKeys)
}

func (s *implSummarizer) discoverSRTFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) == ".srt" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
