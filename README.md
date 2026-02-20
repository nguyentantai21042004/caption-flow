# Video Processing Pipeline

Automated video processing pipeline that transcribes audio and burns subtitles into videos using Whisper and FFmpeg, optimized for Apple Silicon.

## Features

- 🎯 Automatic video detection and processing
- 🎤 High-accuracy speech-to-text transcription (Whisper)
- 📝 Hardcoded subtitles with no font issues on macOS
- ⚡ Hardware-accelerated video encoding (Apple Silicon)
- 🔄 Automatic cleanup of temporary files
- 📊 Structured logging with multiple levels

## Prerequisites

### Hardware

- Apple Silicon Mac (M1, M2, M3, M4, or later)
- 16GB RAM minimum (32GB recommended for large models)

### Software

- Go 1.21 or later
- FFmpeg with VideoToolbox support
- whisper.cpp compiled with Metal acceleration

## Installation

### 1. Install Dependencies

```bash
# Install Go
brew install go

# Install FFmpeg
brew install ffmpeg

# Verify FFmpeg has VideoToolbox support
ffmpeg -encoders | grep videotoolbox
```

### 2. Setup Whisper

```bash
# Clone whisper.cpp
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp

# Build with Metal acceleration
make

# Download model
cd models
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
cd ../..
```

### 3. Build the Application

```bash
# Clone this repository
cd caption-flow

# Install Go dependencies
go mod download

# Build the binary
go build -o vid-pipeline cmd/pipeline/main.go
```

## Configuration

Edit `config.yaml` to customize settings:

```yaml
whisper:
  model_path: "models/ggml-large-v3-turbo.bin"
  binary_path: "./whisper.cpp/main"
  language: "en"
  prompt: "technical terms, code, architecture, API"

ffmpeg:
  video_bitrate: "5M"
  audio_codec: "copy"
  encoder: "h264_videotoolbox"

paths:
  input: "data/input"
  processing: "data/processing"
  output: "data/output"

logging:
  level: "info" # debug, info, warn, error
  format: "text"
```

## Usage

### Run the Pipeline

```bash
./vid-pipeline
```

The application will:

1. Monitor the `data/input` folder
2. Automatically process any video files dropped into it
3. Output processed videos to `data/output`

### Processing Steps

For each video, the pipeline:

1. Extracts audio (16kHz mono WAV)
2. Transcribes using Whisper → generates SRT subtitle
3. Converts SRT to ASS format (fixes macOS font issues)
4. Burns subtitle into video using hardware acceleration
5. Saves final video and subtitle to output folder
6. Cleans up temporary files

### Supported Video Formats

- MP4 (.mp4)
- MOV (.mov)
- AVI (.avi)
- MKV (.mkv)
- WebM (.webm)

## Project Structure

```
caption-flow/
├── cmd/
│   └── pipeline/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── logger/                  # Structured logging
│   ├── processor/               # Video processing logic
│   └── watcher/                 # File system monitoring
├── pkg/
│   └── executor/                # Command execution wrapper
├── scripts/
│   └── setup.sh                 # Setup script
├── data/
│   ├── input/                   # Drop videos here
│   ├── output/                  # Final results
│   ├── archived/                # Processed source videos
│   └── temp/                    # Temporary processing files
├── models/                      # Whisper models
├── config.yaml                  # Configuration file
└── README.md
```

## Deployment

### Option 1: Run as macOS Daemon (launchd)

Create `~/Library/LaunchAgents/com.yourname.vidpipeline.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.yourname.vidpipeline</string>
    <key>ProgramArguments</key>
    <array>
        <string>/full/path/to/vid-pipeline</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/full/path/to/caption-flow</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/vidpipeline.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/vidpipeline.error.log</string>
</dict>
</plist>
```

Load the daemon:

```bash
launchctl load ~/Library/LaunchAgents/com.yourname.vidpipeline.plist
```

### Option 2: Run Manually

```bash
./vid-pipeline
```

Press `Ctrl+C` to stop gracefully.

## Troubleshooting

### Transcription Issues

**Problem**: Incorrect transcription or hallucination

- **Solution**: Verify `-l en` flag is set in config
- **Solution**: Customize `prompt` field with domain-specific keywords
- **Solution**: Try a different model (medium vs large)

**Problem**: Slow transcription

- **Solution**: Verify Metal acceleration is enabled (check whisper.cpp build)
- **Solution**: Ensure audio is 16kHz mono

### Video Issues

**Problem**: Subtitles not visible

- **Solution**: Check ASS file exists in processing folder
- **Solution**: Verify FFmpeg conversion step completed

**Problem**: Video quality degraded

- **Solution**: Increase `video_bitrate` in config (e.g., "8M")
- **Solution**: Verify hardware encoder is being used

### Application Issues

**Problem**: Watcher not detecting files

- **Solution**: Check folder permissions
- **Solution**: Verify path in config.yaml is correct

**Problem**: Memory usage high

- **Solution**: Ensure cleanup functions are running
- **Solution**: Check for stuck processes

## Performance

Expected performance on M4 Pro:

- Processing speed: < 0.3x realtime (10 min video → < 3 min)
- CPU usage: < 80% average
- Memory usage: < 4GB per video
- Transcription accuracy: > 95% (English)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
