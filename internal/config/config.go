package config

import "fmt"

type Config struct {
	Whisper     WhisperConfig     `yaml:"whisper"`
	FFmpeg      FFmpegConfig      `yaml:"ffmpeg"`
	Paths       PathsConfig       `yaml:"paths"`
	Logging     LoggingConfig     `yaml:"logging"`
	Performance PerformanceConfig `yaml:"performance"`
	DeepSeek    DeepSeekConfig    `yaml:"deepseek"`
	Gemini      GeminiConfig      `yaml:"gemini"`
}

type WhisperConfig struct {
	ModelPath  string `yaml:"model_path"`
	BinaryPath string `yaml:"binary_path"`
	Language   string `yaml:"language"`
	Prompt     string `yaml:"prompt"`
	Threads    int    `yaml:"threads"`
	UseGPU     bool   `yaml:"use_gpu"`
}

type FFmpegConfig struct {
	VideoBitrate string `yaml:"video_bitrate"`
	AudioCodec   string `yaml:"audio_codec"`
	Encoder      string `yaml:"encoder"`
	Preset       string `yaml:"preset"`
}

type PathsConfig struct {
	Input    string `yaml:"input"`
	Output   string `yaml:"output"`
	Archived string `yaml:"archived"`
	Temp     string `yaml:"temp"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type PerformanceConfig struct {
	MaxConcurrent int `yaml:"max_concurrent"`
}

type DeepSeekConfig struct {
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type GeminiConfig struct {
	Model  string `yaml:"model"`
	Prompt string `yaml:"prompt"`
}

func (c *Config) Validate() error {
	if c.Whisper.ModelPath == "" {
		return fmt.Errorf("whisper.model_path is required")
	}
	if c.Whisper.BinaryPath == "" {
		return fmt.Errorf("whisper.binary_path is required")
	}
	if c.Whisper.Language == "" {
		return fmt.Errorf("whisper.language is required")
	}
	if c.FFmpeg.Encoder == "" {
		return fmt.Errorf("ffmpeg.encoder is required")
	}
	if c.Paths.Input == "" {
		return fmt.Errorf("paths.input is required")
	}
	if c.Paths.Output == "" {
		return fmt.Errorf("paths.output is required")
	}

	if c.Paths.Archived == "" {
		c.Paths.Archived = "data/archived"
	}
	if c.Paths.Temp == "" {
		c.Paths.Temp = "data/temp"
	}
	if c.Performance.MaxConcurrent == 0 {
		c.Performance.MaxConcurrent = 2
	}
	if c.Whisper.Threads == 0 {
		c.Whisper.Threads = 8
	}
	if c.FFmpeg.Preset == "" {
		c.FFmpeg.Preset = "medium"
	}
	if c.DeepSeek.Model == "" {
		c.DeepSeek.Model = "deepseek-chat"
	}
	if c.DeepSeek.BaseURL == "" {
		c.DeepSeek.BaseURL = "https://api.deepseek.com/v1"
	}
	if c.Gemini.Model == "" {
		c.Gemini.Model = "gemini-2.5-flash"
	}
	if c.Gemini.Prompt == "" {
		c.Gemini.Prompt = `Bạn là một chuyên gia phân tích nội dung video đào tạo. Dựa trên phụ đề bên dưới, hãy viết một bản tóm tắt CHI TIẾT bằng TIẾNG VIỆT.

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
	}

	return nil
}
