package config

import (
	"os"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Whisper: WhisperConfig{
					ModelPath:  "models/test.bin",
					BinaryPath: "./whisper",
					Language:   "en",
				},
				FFmpeg: FFmpegConfig{
					Encoder: "h264_videotoolbox",
				},
				Paths: PathsConfig{
					Input:  "data/input",
					Output: "data/output",
				},
			},
			wantErr: false,
		},
		{
			name: "missing model path",
			config: Config{
				Whisper: WhisperConfig{
					BinaryPath: "./whisper",
					Language:   "en",
				},
				FFmpeg: FFmpegConfig{
					Encoder: "h264_videotoolbox",
				},
				Paths: PathsConfig{
					Input:  "data/input",
					Output: "data/output",
				},
			},
			wantErr: true,
		},
		{
			name: "missing paths",
			config: Config{
				Whisper: WhisperConfig{
					ModelPath:  "models/test.bin",
					BinaryPath: "./whisper",
					Language:   "en",
				},
				FFmpeg: FFmpegConfig{
					Encoder: "h264_videotoolbox",
				},
				Paths: PathsConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
whisper:
  model_path: "models/test.bin"
  binary_path: "./whisper"
  language: "en"
  prompt: "test"

ffmpeg:
  video_bitrate: "5M"
  audio_codec: "copy"
  encoder: "h264_videotoolbox"

paths:
  input: "data/input"
  output: "data/output"

logging:
  level: "info"
  format: "text"
`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}

	if cfg.Whisper.ModelPath != "models/test.bin" {
		t.Errorf("ModelPath = %v, want %v", cfg.Whisper.ModelPath, "models/test.bin")
	}

	if cfg.Paths.Input != "data/input" {
		t.Errorf("Input = %v, want %v", cfg.Paths.Input, "data/input")
	}
}

func TestLoadInvalidFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}
