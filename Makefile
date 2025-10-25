.PHONY: all build clean install test whisper

all: build

# Detect CUDA availability
CUDA_CHECK := $(shell ./detect_cuda.sh)
ifeq ($(CUDA_CHECK),CUDA_AVAILABLE=1)
    CUDA_FLAGS := -DGGML_CUDA=ON
    USE_CUDA := 1
    $(info 🚀 CUDA detected - building with GPU acceleration)
else
    CUDA_FLAGS :=
    USE_CUDA := 0
    $(info 💻 CUDA not found - building with CPU only)
endif

whisper:
	@echo "📥 Setting up whisper.cpp..."
	@if [ ! -d "whisper.cpp" ]; then \
		git clone https://github.com/ggerganov/whisper.cpp; \
	fi
	@echo "🔨 Building whisper.cpp with CMake..."
	@cmake -S whisper.cpp -B whisper.cpp/build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF $(CUDA_FLAGS)
	@cmake --build whisper.cpp/build --target whisper
	@echo "✅ whisper.cpp ready!"

build: whisper
	@echo "🔨 Building hyprwhspr..."
	@mkdir -p bin
	@if [ "$(USE_CUDA)" = "1" ]; then \
		CGO_ENABLED=1 go build -tags cuda -o bin/hyprwhspr .; \
	else \
		CGO_ENABLED=1 go build -o bin/hyprwhspr .; \
	fi
	@echo "✅ Build complete!"
	@if [ "$(USE_CUDA)" = "1" ]; then \
		echo "   🚀 Built with CUDA support"; \
	else \
		echo "   💻 Built with CPU only"; \
	fi
	@echo "   Binary: bin/hyprwhspr"
	@echo ""
	@echo "Usage:"
	@echo "   ./bin/hyprwhspr          # Start daemon"
	@echo "   ./bin/hyprwhspr toggle   # Toggle recording"

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	@echo "✅ Clean complete!"

clean-all: clean
	@echo "🧹 Cleaning whisper.cpp..."
	rm -rf whisper.cpp
	@echo "✅ Full clean complete!"

install: build
	@echo "📦 Installing..."
	sudo cp bin/hyprwhspr /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/"
	@echo ""
	@echo "Now you can use:"
	@echo "   hyprwhspr          # Start daemon"
	@echo "   hyprwhspr toggle   # Toggle recording"

test:
	@echo "🧪 Running tests..."
	go test ./...

fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed!"

run: build
	@echo "🚀 Starting hyprwhspr daemon..."
	./bin/hyprwhspr

help:
	@echo "hyprwhspr - Makefile commands"
	@echo ""
	@echo "  make build      - Build binary (auto-downloads whisper.cpp)"
	@echo "  make whisper    - Download and build whisper.cpp"
	@echo "  make clean      - Remove built binary"
	@echo "  make clean-all  - Remove binary and whisper.cpp"
	@echo "  make install    - Install to /usr/local/bin (requires sudo)"
	@echo "  make test       - Run tests"
	@echo "  make fmt        - Format code"
	@echo "  make deps       - Download Go dependencies"
	@echo "  make run        - Build and run daemon"
