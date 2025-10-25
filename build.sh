#!/bin/bash
# Simple build script for hyprwhspr-go

set -e

echo "🚀 hyprwhspr-go build script"
echo ""

# Detect CUDA availability
source ./detect_cuda.sh
if [ "$CUDA_AVAILABLE" = "1" ]; then
    echo "🚀 CUDA detected - building with GPU acceleration"
    CUDA_FLAGS="-DGGML_CUDA=ON"
    export USE_CUDA=1
else
    echo "💻 CUDA not found - building with CPU only"
    CUDA_FLAGS=""
    export USE_CUDA=0
fi
echo ""

# Check for whisper.cpp
if [ ! -d "whisper.cpp" ]; then
    echo "📥 Cloning whisper.cpp..."
    git clone https://github.com/ggerganov/whisper.cpp
fi

# Build whisper.cpp with CMake
if [ ! -d "whisper.cpp/build" ]; then
    echo "🔨 Building whisper.cpp with CMake..."
    cmake -S whisper.cpp -B whisper.cpp/build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF $CUDA_FLAGS
    cmake --build whisper.cpp/build --target whisper
    echo "✅ whisper.cpp ready!"
else
    echo "✅ whisper.cpp already built (use 'rm -rf whisper.cpp/build' to rebuild)"
fi

echo ""
echo "🔨 Building hyprwhspr..."

# Create bin directory
mkdir -p bin

# Build single binary with CGo
if [ "$USE_CUDA" = "1" ]; then
    CGO_ENABLED=1 go build -tags cuda -o bin/hyprwhspr .
else
    CGO_ENABLED=1 go build -o bin/hyprwhspr .
fi

echo ""
echo "✅ Build complete!"
if [ "$USE_CUDA" = "1" ]; then
    echo "   🚀 Built with CUDA support"
else
    echo "   💻 Built with CPU only"
fi
echo ""
echo "Binary: bin/hyprwhspr"
echo ""
echo "Usage:"
echo "  ./bin/hyprwhspr          # Start daemon"
echo "  ./bin/hyprwhspr toggle   # Toggle recording"
echo "  ./bin/hyprwhspr start    # Start recording"
echo "  ./bin/hyprwhspr stop     # Stop recording"
echo "  ./bin/hyprwhspr status   # Check status"
echo "  ./bin/hyprwhspr help     # Show help"
echo ""
echo "Install (optional):"
echo "  sudo cp bin/hyprwhspr /usr/local/bin/"
echo ""
echo "Then use as:"
echo "  hyprwhspr toggle"
