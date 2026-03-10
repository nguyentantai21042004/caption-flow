package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/genai"
)

// SummarizeAll discovers SRT files in outputDir (root), then for each:
//   - writes transcript docx to outputDir/transcripts/
//   - calls DeepSeek (fallback Gemini) and writes summary docx to outputDir/summaries/
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

		// 2) Summary DOCX — LLM-generated summary (DeepSeek primary, Gemini fallback)
		summary, err := s.callSummary(ctx, srtText)
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

		// Rate limiting delay between successfully processed files (except the last one)
		if i < len(srtFiles)-1 {
			s.logger.Info(ctx, "Sleeping 3s to respect LLM API rate limits...")
			time.Sleep(3 * time.Second)
		}
	}

	s.logger.Info(ctx, "Processing complete: %d success, %d failed", successCount, failCount)
	return nil
}

func (s *implSummarizer) callSummary(ctx context.Context, transcript string) (string, error) {
	if len(s.deepSeekKeys) == 0 {
		return "", fmt.Errorf("no DEEPSEEK_API_KEYS configured")
	}

	deepSeekSummary, deepSeekErr := s.callDeepSeek(ctx, transcript)
	if deepSeekErr == nil {
		return deepSeekSummary, nil
	}

	if len(s.geminiKeys) == 0 {
		return "", fmt.Errorf("deepseek failed and no GEMINI_API_KEYS configured: %w", deepSeekErr)
	}

	s.logger.Warn(ctx, "DeepSeek failed, falling back to Gemini: %v", deepSeekErr)
	geminiSummary, geminiErr := s.callGemini(ctx, transcript)
	if geminiErr != nil {
		return "", fmt.Errorf("deepseek failed: %v; gemini fallback failed: %w", deepSeekErr, geminiErr)
	}

	return geminiSummary, nil
}

// callDeepSeek sends transcript to DeepSeek API and returns summary text.
// Rotates API keys on quota/rate-limit related failures.
func (s *implSummarizer) callDeepSeek(ctx context.Context, transcript string) (string, error) {
	type deepSeekMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type deepSeekRequest struct {
		Model       string            `json:"model"`
		Messages    []deepSeekMessage `json:"messages"`
		Temperature float64           `json:"temperature,omitempty"`
	}
	type deepSeekChoice struct {
		Message deepSeekMessage `json:"message"`
	}
	type deepSeekResponse struct {
		Choices []deepSeekChoice `json:"choices"`
	}
	type deepSeekErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	prompt := fmt.Sprintf(s.prompt, transcript)
	requestBody := deepSeekRequest{
		Model: s.deepSeekModel,
		Messages: []deepSeekMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal deepseek request: %w", err)
	}

	attempts := len(s.deepSeekKeys) * 3
	var lastErr error
	backoff := 3 * time.Second

	for i := 0; i < attempts; i++ {
		key := s.deepSeekKeys[s.currentDeepSeekKey]
		url := strings.TrimRight(s.deepSeekBaseURL, "/") + "/chat/completions"

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestJSON))
		if err != nil {
			return "", fmt.Errorf("create deepseek request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+key)

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("send deepseek request: %w", err)
			s.rotateDeepSeekKey()
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read deepseek response: %w", readErr)
			s.rotateDeepSeekKey()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			var apiErr deepSeekErrorResponse
			if err := json.Unmarshal(body, &apiErr); err != nil {
				apiErr.Error.Message = string(body)
			}

			errMsg := strings.ToLower(apiErr.Error.Message)
			if resp.StatusCode == http.StatusTooManyRequests || strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "rate") {
				s.logger.Warn(ctx, "DeepSeek key %d rate limited, rotating... (attempt %d/%d). Sleeping for %v", s.currentDeepSeekKey+1, i+1, attempts, backoff)
				s.rotateDeepSeekKey()
				lastErr = fmt.Errorf("deepseek API error %d: %s", resp.StatusCode, apiErr.Error.Message)

				time.Sleep(backoff)
				backoff *= 2
				if backoff > 45*time.Second {
					backoff = 45 * time.Second
				}
				continue
			}

			return "", fmt.Errorf("deepseek API error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}

		var parsed deepSeekResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return "", fmt.Errorf("parse deepseek response: %w", err)
		}

		if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
			return "", fmt.Errorf("empty response from DeepSeek")
		}

		return parsed.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("all DeepSeek API keys exhausted: %w", lastErr)
}

// callGemini sends the transcript to Gemini and returns the summary text.
// Rotates API keys on 429 / quota errors.
func (s *implSummarizer) callGemini(ctx context.Context, transcript string) (string, error) {
	prompt := fmt.Sprintf(s.prompt, transcript)

	attempts := len(s.geminiKeys) * 3 // Try each key multiple times with backoff
	var lastErr error
	backoff := 5 * time.Second

	for i := 0; i < attempts; i++ {
		key := s.geminiKeys[s.currentGeminiKey]

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  key,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			lastErr = fmt.Errorf("create client: %w", err)
			s.rotateGeminiKey()
			continue
		}

		result, err := client.Models.GenerateContent(ctx, s.geminiModel, genai.Text(prompt), nil)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "RESOURCE_EXHAUSTED") || strings.Contains(errMsg, "retry in") {
				s.logger.Warn(ctx, "Gemini key %d rate limited, rotating... (attempt %d/%d). Sleeping for %v", s.currentGeminiKey+1, i+1, attempts, backoff)
				s.rotateGeminiKey()
				lastErr = err

				time.Sleep(backoff)
				backoff *= 2 // Exponential backoff
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
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

func (s *implSummarizer) rotateGeminiKey() {
	s.currentGeminiKey = (s.currentGeminiKey + 1) % len(s.geminiKeys)
}

func (s *implSummarizer) rotateDeepSeekKey() {
	s.currentDeepSeekKey = (s.currentDeepSeekKey + 1) % len(s.deepSeekKeys)
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
