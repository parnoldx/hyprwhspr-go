# hyprwhspr-go

**Speech-to-text daemon for Hyprland, written in Go**

Single binary, fast, simple

## Features

- **Single binary** - One command for everything (daemon + control)
- **CUDA acceleration** - Automatically uses GPU when available for faster transcription
- **Language auto-detection** - Speak any language, Whisper detects it automatically
- **wtype integration** - Direct text injection, no clipboard pollution
- **Command mode** - Trigger custom scripts with voice commands
- **Fast & efficient** - ~20-30MB memory, ~100ms startup

## Quick Start

### 1. Build

```bash
# Automatic (downloads whisper.cpp, builds everything)
make build

# Or use the build script
./build.sh
```

This will:
- Clone whisper.cpp (if needed)
- Build whisper.cpp library
- Build the Go binary with CGo

### 2. Download Whisper Models

```bash
mkdir -p ~/.local/share/hyprwhspr
cd ~/.local/share/hyprwhspr
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin
```

Available models: `tiny`, `base`, `small`, `medium`, `large`

### 3. Test

```bash
# Start daemon
./bin/hyprwhspr

# In another terminal
./bin/hyprwhspr status    # Check status
./bin/hyprwhspr toggle    # Toggle recording
```

### 4. Install

```bash
# Install system-wide
make install

# Now use from anywhere
hyprwhspr              # Start daemon
hyprwhspr toggle       # Toggle recording
```

### 5. Configure Hyprland

Add to `~/.config/hypr/bindings.conf`:

```conf
# Toggle recording with SUPER+ALT+D
bind = SUPER, D, exec, hyprwhspr toggle
```

## Usage

### Single Binary Commands

```bash
# Start daemon (default if no command)
hyprwhspr
hyprwhspr daemon

# Control commands (send to running daemon)
hyprwhspr start      # Start recording
hyprwhspr stop       # Stop recording
hyprwhspr toggle     # Toggle on/off
hyprwhspr status     # Check status

# Other
hyprwhspr help       # Show help
hyprwhspr version    # Show version
```

### Workflow

1. Press `SUPER+D` to start recording
2. Speak in any language (English, German, etc.)
3. Press `SUPER+D` again to stop
4. Whisper auto-detects language and transcribes
5. Text is injected instantly (no clipboard pollution with wtype!)

## Configuration

Config file: `~/.config/hyprwhspr/config.json`

```json
{
  "model": "base",
  "word_overrides": {},
  "whisper_prompt": "Transcribe with proper capitalization, including sentence beginnings, proper nouns, titles, and standard capitalization rules.",
  "audio_feedback": true,
  "command_mode": false,
  "commands": {}
}
```

### Configuration Options

- **model** - Whisper model to use (`tiny`, `base`, `small`, `medium`, `large`)
- **threads** - Number of CPU threads for transcription
- **language** - Force specific language (e.g., `"en"`, `"de"`), or `null` for auto-detection
- **command_mode** - Enable voice command mode (see below)
- **commands** - Map of voice commands to script paths

## Command Mode

Command Mode allows you to trigger custom scripts based on the first word of your transcribed speech.

### Enable Command Mode

Add to your `~/.config/hyprwhspr/config.json`:

```json
{
  "command_mode": true,
  "commands": {
    "note": "/path/to/scripts/note.sh",
    "workspace": "/path/to/scripts/workspace.sh"
  }
}
```

### Example Usage

- **Say:** `"note remember to buy milk"` → Appends note with timestamp
- **Say:** `"workspace 3"` → Switches to Hyprland workspace 3
- **Say:** `"Hello world"` → Types "Hello world" (no command triggered)

### Writing Custom Scripts

Scripts receive the remaining text (after the command word) as the first argument:

```bash
#!/bin/bash
# my-command.sh

TEXT="$1"

# Do something with $TEXT
echo "Received: $TEXT"

# Optional: Show notification
notify-send "My Command" "$TEXT"
```

**Requirements:**
1. Must be executable (`chmod +x script.sh`)
2. Must have shebang (`#!/bin/bash`)
3. Use absolute paths in config


## Waybar: Hyprwhspr Status Indicator

  Makes the Omarchy logo in Waybar turn green (#A1CB6C) when hyprwhspr is recording.

  Setup

  1. Create the status script:

  ~/.local/bin/omarchy-hyprwhspr-status.sh:
  #!/bin/bash
  # Returns CSS class for Waybar based on hyprwhspr status
  [ "$(hyprwhspr status 2>/dev/null || echo 0)" = "1" ] && echo '{"class":"active"}' ||
  echo '{"class":""}'

  chmod +x ~/.local/bin/omarchy-hyprwhspr-status.sh

  2. Update Waybar config:

  In ~/.config/waybar/config.jsonc, modify the custom/omarchy module:
  "custom/omarchy": {
    "format": "<span font='omarchy'>\ue900</span>",
    "exec": "~/.local/bin/omarchy-hyprwhspr-status.sh",
    "return-type": "json",
    "signal": 9,
    "on-click": "omarchy-menu",
    "on-click-right": "omarchy-launch-terminal",
    "tooltip-format": "Omarchy Menu\n\nSuper + Alt + Space"
  }

  3. Add CSS styling:

  In ~/.config/waybar/style.css:
  #custom-omarchy.active {
    color: #A1CB6C;
  }

  4. Reload Waybar:
  omarchy-restart-waybar

## Dependencies

### Required
- **Go 1.21+** - [golang.org](https://golang.org)
- **CGo** - C compiler (gcc/clang)
- **wtype** - Wayland text tool (for injection)

### Build Dependencies
- **make** - Build tool
- **git** - To clone whisper.cpp
- **cmake** - Build whisper.cpp

### Optional (for GPU acceleration)
- **CUDA Toolkit** - NVIDIA CUDA for GPU acceleration
- **nvidia-drivers** - NVIDIA GPU drivers

### Install Dependencies

```bash
# Arch Linux (CPU only)
sudo pacman -S go gcc make git wtype cmake

# Arch Linux (with CUDA support)
sudo pacman -S go gcc make git wtype cmake cuda
```

## CUDA Acceleration

The build system automatically detects CUDA and builds with GPU acceleration when available:

- **CUDA detected** → Builds with GPU support (faster transcription)
- **No CUDA** → Builds with CPU only (still fast!)

### Requirements for CUDA
- NVIDIA GPU (Compute Capability 3.5+)
- CUDA Toolkit installed (`nvcc` in PATH)
- NVIDIA drivers installed

### Verifying CUDA Support

After building, the daemon will log which backend is being used:

```
[whisper] Acceleration: CUDA (GPU)    # GPU acceleration enabled
[whisper] Acceleration: CPU only      # CPU-only build
```

### Performance Comparison

Typical transcription times for 5 seconds of audio:

| Model  | CPU (4 threads) | CUDA (GPU) |
|--------|-----------------|------------|
| tiny   | ~200ms          | ~50ms      |
| base   | ~500ms          | ~100ms     |
| small  | ~1.5s           | ~300ms     |
| medium | ~3.5s           | ~700ms     |

*Times are approximate and vary by hardware*

## Building

### Option 1: Makefile (Recommended)

```bash
make build        # Build everything
make install      # Install to /usr/local/bin
make clean        # Clean bin/
make clean-all    # Clean bin/ and whisper.cpp/
```

### Option 2: Build Script

```bash
./build.sh
```

## Troubleshooting

### Build fails with whisper.cpp errors

```bash
# Clean and rebuild
make clean-all
make build
```

### CUDA not detected but I have it installed

```bash
# Check if nvcc is in PATH
which nvcc

# If not found, add CUDA to PATH
export PATH=/usr/local/cuda/bin:$PATH

# Check if nvidia-smi works
nvidia-smi

# Rebuild
make clean-all
make build
```

### Build fails with CUDA errors

```bash
# Build without CUDA (CPU only)
rm -rf whisper.cpp/build
cmake -S whisper.cpp -B whisper.cpp/build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF
cmake --build whisper.cpp/build --target whisper
CGO_ENABLED=1 go build -o bin/hyprwhspr .
```

### Can't find whisper model

```bash
# Download model
mkdir -p ~/.local/share/hyprwhspr
cd ~/.local/share/hyprwhspr
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin
```

### Daemon won't start

```bash
# Check if socket exists
ls -la ~/.config/hyprwhspr/hyprwhspr.sock

# Remove old socket
rm ~/.config/hyprwhspr/hyprwhspr.sock

# Try again
./bin/hyprwhspr
```

### wtype not found

```bash
# Arch
sudo pacman -S wtype

# Or build from source
git clone https://github.com/atx/wtype
cd wtype
meson build && ninja -C build
sudo ninja -C build install
```

### Command not triggering (Command Mode)

1. Check `command_mode: true` in config
2. Verify command words match exactly (case-insensitive)
3. Use absolute paths for scripts
4. Ensure scripts are executable (`chmod +x`)

## Resources

- [Go documentation](https://golang.org/doc/)
- [whisper.cpp](https://github.com/ggerganov/whisper.cpp)
- [whisper.cpp Go bindings](https://github.com/ggerganov/whisper.cpp/tree/master/bindings/go)
- [malgo audio library](https://github.com/gen2brain/malgo)
- [wtype](https://github.com/atx/wtype)

## Summary

You now have a **single, self-contained binary** for speech-to-text on Hyprland:

- One command: `hyprwhspr`
- whisper.cpp fully integrated via CGo
- **Automatic CUDA acceleration** when GPU is available
- Language auto-detection working
- Command mode for voice-triggered scripts
- No Python, no pip, no venv
- ~12MB binary (CPU) or ~15MB (with CUDA)

Just build and run!

