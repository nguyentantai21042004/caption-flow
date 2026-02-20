.PHONY: build run run-pipeline clean test install-deps setup-whisper help

# Build the application
build:
	@echo "Building caption-flow..."
	@go build -ldflags="-s -w" -o vid-pipeline cmd/pipeline/main.go
	@echo "Build complete: ./vid-pipeline"

# Run the application
run: build
	@echo "Starting caption-flow..."
	@./vid-pipeline

# Run pipeline: all files (default) or specific file(s)
run-pipeline: build
ifdef FILE
	@./vid-pipeline -target "$(FILE)"
else
	@./vid-pipeline -target-all
endif

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f vid-pipeline
	@rm -rf data/temp/*
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Install Go dependencies
install-deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

# Setup whisper.cpp (requires manual model download)
setup-whisper:
	@echo "Setting up whisper.cpp..."
	@if [ ! -d "whisper.cpp" ]; then \
		git clone https://github.com/ggerganov/whisper.cpp.git; \
		cd whisper.cpp && make; \
		echo "Whisper.cpp built successfully"; \
		echo ""; \
		echo "Next steps:"; \
		echo "1. Download model: cd whisper.cpp/models && ./download-ggml-model.sh large-v3-turbo"; \
		echo "2. Move model: mv whisper.cpp/models/ggml-large-v3-turbo.bin models/"; \
	else \
		echo "whisper.cpp already exists"; \
	fi

# Verify system requirements
verify:
	@echo "Verifying system requirements..."
	@echo "Go version:"
	@go version
	@echo ""
	@echo "FFmpeg version:"
	@ffmpeg -version | head -n 1
	@echo ""
	@echo "Checking VideoToolbox encoder:"
	@ffmpeg -encoders 2>/dev/null | grep videotoolbox || echo "VideoToolbox not found!"
	@echo ""
	@echo "Checking directories:"
	@ls -la data/

# Show help
help:
	@echo "Video Processing Pipeline - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build                        Build the application binary"
	@echo "  run                          Build and run (show usage)"
	@echo "  run-pipeline                 Process ALL video files in input"
	@echo "  run-pipeline FILE=\"name\"     Process specific file(s)"
	@echo "  clean                        Remove build artifacts and temp files"
	@echo "  test                         Run tests"
	@echo "  install-deps                 Install Go dependencies"
	@echo "  setup-whisper                Clone and build whisper.cpp"
	@echo "  verify                       Verify system requirements"
	@echo "  help                         Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make run-pipeline"
	@echo "  make run-pipeline FILE=\"1 How to login on the Global Goal Tool.mp4\""
	@echo "  make run-pipeline FILE=\"video1.mp4,video2.mp4\""
