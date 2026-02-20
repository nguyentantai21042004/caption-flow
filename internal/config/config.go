package config

import "fmt"

// Config represents the application configuration
type Config struct {
	Whisper     WhisperConfig     `yaml:"whisper"`
	FFmpeg      FFmpegConfig      `yaml:"ffmpeg"`
	Paths       PathsConfig       `yaml:"paths"`
	Logging     LoggingConfig     `yaml:"logging"`
	Performance PerformanceConfig `yaml:"performance"`
}

// WhisperConfig contains Whisper-related settings
type WhisperConfig struct {
	ModelPath  string `yaml:"model_path"`
	BinaryPath string `yaml:"binary_path"`
	Language   string `yaml:"language"`
	Prompt     string `yaml:"prompt"`
	Threads    int    `yaml:"threads"`
	UseGPU     bool   `yaml:"use_gpu"`
}

// FFmpegConfig contains FFmpeg-related settings
type FFmpegConfig struct {
	VideoBitrate string `yaml:"video_bitrate"`
	AudioCodec   string `yaml:"audio_codec"`
	Encoder      string `yaml:"encoder"`
	Preset       string `yaml:"preset"`
	MaxBitrate   string `yaml:"max_bitrate"`
	BufSize      string `yaml:"bufsize"`
}

// PathsConfig contains directory paths
type PathsConfig struct {
	Input      string `yaml:"input"`
	Processing string `yaml:"processing"`
	Output     string `yaml:"output"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// PerformanceConfig contains performance tuning settings
type PerformanceConfig struct {
	MaxConcurrent   int `yaml:"max_concurrent"`
	BufferSize      int `yaml:"buffer_size"`
	CleanupInterval int `yaml:"cleanup_interval"`
}

// Validate checks if the configuration is valid
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
	if c.Paths.Processing == "" {
		return fmt.Errorf("paths.processing is required")
	}
	if c.Paths.Output == "" {
		return fmt.Errorf("paths.output is required")
	}

	// Set defaults for performance config
	if c.Performance.MaxConcurrent == 0 {
		c.Performance.MaxConcurrent = 2
	}
	if c.Performance.BufferSize == 0 {
		c.Performance.BufferSize = 8388608 // 8MB
	}
	if c.Performance.CleanupInterval == 0 {
		c.Performance.CleanupInterval = 300
	}

	// Set defaults for Whisper
	if c.Whisper.Threads == 0 {
		c.Whisper.Threads = 8
	}

	// Set defaults for FFmpeg
	if c.FFmpeg.Preset == "" {
		c.FFmpeg.Preset = "medium"
	}

	return nil
}
