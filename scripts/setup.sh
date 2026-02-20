#!/bin/bash

set -e

echo "=========================================="
echo "Video Processing Pipeline - Setup Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check Go FIRST - exit immediately if not found
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go first: brew install go"
    echo "Aborting setup. No changes made."
    exit 1
fi

echo -e "${GREEN}✓ Go installed:${NC} $(go version)"
echo ""

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo -e "${RED}Error: This script is designed for macOS${NC}"
    exit 1
fi

# Check if running on Apple Silicon
if [[ $(uname -m) != "arm64" ]]; then
    echo -e "${YELLOW}Warning: This project is optimized for Apple Silicon (M1/M2/M3/M4)${NC}"
    echo "You can continue, but hardware acceleration may not work optimally."
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Step 1: Checking prerequisites..."
echo ""

# Check FFmpeg - install if missing
if ! command -v ffmpeg &> /dev/null; then
    echo -e "${YELLOW}FFmpeg is not installed. Installing via Homebrew...${NC}"
    if command -v brew &> /dev/null; then
        brew install ffmpeg
        echo -e "${GREEN}✓ FFmpeg installed${NC}"
    else
        echo -e "${YELLOW}Homebrew not found. Please install FFmpeg manually: https://ffmpeg.org${NC}"
    fi
else
    echo -e "${GREEN}✓ FFmpeg installed:${NC} $(ffmpeg -version | head -n 1)"
fi

# Check VideoToolbox encoder (only if ffmpeg exists)
if command -v ffmpeg &> /dev/null; then
    if ffmpeg -encoders 2>/dev/null | grep -q "h264_videotoolbox"; then
        echo -e "${GREEN}✓ VideoToolbox encoder available${NC}"
    else
        echo -e "${YELLOW}✗ VideoToolbox encoder not found (hardware acceleration may be limited)${NC}"
    fi
fi

echo ""
echo "Step 2: Installing Go dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓ Dependencies installed${NC}"

echo ""
echo "Step 3: Setting up whisper.cpp..."
if [ ! -d "whisper.cpp" ]; then
    echo "Cloning whisper.cpp..."
    git clone https://github.com/ggerganov/whisper.cpp.git
    
    echo "Building whisper.cpp with Metal acceleration..."
    cd whisper.cpp
    make
    cd ..
    echo -e "${GREEN}✓ whisper.cpp built successfully${NC}"
else
    echo -e "${YELLOW}whisper.cpp already exists, skipping...${NC}"
fi

echo ""
echo "Step 4: Downloading Whisper model..."
if [ ! -f "models/ggml-large-v3-turbo.bin" ]; then
    echo "Downloading ggml-large-v3-turbo model (~1.5GB)..."
    echo "This may take a few minutes..."
    
    mkdir -p models
    cd models
    
    # Try to download the model
    if command -v wget &> /dev/null; then
        wget -q --show-progress https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
    elif command -v curl &> /dev/null; then
        curl -L -o ggml-large-v3-turbo.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
    else
        echo -e "${YELLOW}Neither wget nor curl found. Skipping model download.${NC}"
        echo "Download manually and save to models/ggml-large-v3-turbo.bin:"
        echo "  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin"
    fi

    cd ..
    echo -e "${GREEN}✓ Model downloaded${NC}"
else
    echo -e "${YELLOW}Model already exists, skipping...${NC}"
fi

echo ""
echo "Step 5: Creating directory structure..."
mkdir -p data/{input,processing,output}
echo -e "${GREEN}✓ Directories created${NC}"

echo ""
echo "Step 6: Building application..."
go build -ldflags="-s -w" -o vid-pipeline pipeline/main.go
echo -e "${GREEN}✓ Application built: ./vid-pipeline${NC}"

echo ""
echo "=========================================="
echo -e "${GREEN}Setup completed successfully!${NC}"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Review config.yaml and adjust settings if needed"
echo "2. Run the pipeline: ./vid-pipeline"
echo "3. Drop video files into: data/input/"
echo ""
echo "For more information, see README.md"
