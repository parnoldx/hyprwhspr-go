#!/bin/bash

# Hyprwhspr SystemD Installation Script
# This script builds and installs hyprwhspr as a SystemD user service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root for system-wide installation
check_permissions() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should be run as a regular user, not as root."
        log_info "The script will use sudo only when needed for system-wide installation."
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    # Check for required commands
    for cmd in go make gcc cmake git wtype; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Please install the missing dependencies:"
        echo "  Arch Linux: sudo pacman -S ${missing_deps[*]}"
        echo "  Ubuntu/Debian: sudo apt install ${missing_deps[*]}"
        exit 1
    fi
    
    log_success "All dependencies found"
}

# Build hyprwhspr
build_hyprwhspr() {
    log_info "Building hyprwhspr..."
    
    # Clean previous builds
    if [ -d "bin" ]; then
        make clean
    fi
    
    # Build the project
    make build
    
    if [ ! -f "bin/hyprwhspr" ]; then
        log_error "Build failed - binary not found"
        exit 1
    fi
    
    log_success "Build completed successfully"
}

# Install binary system-wide
install_binary() {
    log_info "Installing hyprwhspr binary to /usr/local/bin..."
    
    if sudo cp bin/hyprwhspr /usr/local/bin/; then
        sudo chmod +x /usr/local/bin/hyprwhspr
        log_success "Binary installed to /usr/local/bin/hyprwhspr"
    else
        log_error "Failed to install binary"
        exit 1
    fi
}

# Install SystemD user service
install_service() {
    log_info "Installing SystemD user service..."
    
    # Create user systemd directory
    mkdir -p ~/.config/systemd/user
    
    # Create service file
    cat > ~/.config/systemd/user/hyprwhspr.service << 'EOF'
[Unit]
Description=Hyprwhspr Speech-to-Text Daemon
After=graphical-session.target

[Service]
Type=simple
ExecStart=/usr/local/bin/hyprwhspr daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF
    
    # Reload systemd and enable service
    systemctl --user daemon-reload
    systemctl --user enable hyprwhspr.service
    
    log_success "Service installed and enabled for auto-start"
}

# Create configuration directory
create_config() {
    local config_dir="$HOME/.config/hyprwhspr"
    local share_dir="$HOME/.local/share/hyprwhspr"
    
    log_info "Creating configuration directories..."
    
    mkdir -p "$config_dir"
    mkdir -p "$share_dir"
    
    # Copy example config if it doesn't exist
    if [ ! -f "$config_dir/config.json" ] && [ -f "share/config-example.json" ]; then
        cp share/config-example.json "$config_dir/config.json"
        log_info "Example configuration copied to $config_dir/config.json"
    fi
    
    log_success "Configuration directories created"
}

# Download whisper model if needed
download_model() {
    local share_dir="$HOME/.local/share/hyprwhspr"
    local model_file="$share_dir/ggml-base.bin"
    
    if [ ! -f "$model_file" ]; then
        log_info "Downloading Whisper base model..."
        log_warning "This may take a while depending on your internet connection"
        
        cd "$share_dir"
        if wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin; then
            log_success "Whisper model downloaded successfully"
        else
            log_error "Failed to download Whisper model"
            log_info "You can download it manually later with:"
            echo "  cd $share_dir"
            echo "  wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin"
        fi
        cd - > /dev/null
    else
        log_success "Whisper model already exists"
    fi
}

# Start service
start_service() {
    log_info "Starting hyprwhspr service..."
    
    if systemctl --user start hyprwhspr.service; then
        log_success "Service started successfully"
        sleep 2
        if systemctl --user is-active --quiet hyprwhspr.service; then
            log_success "Service is running!"
        else
            log_warning "Service may not be running properly"
            log_info "Check status with: ./hyprwhspr-service status"
        fi
    else
        log_error "Failed to start service"
    fi
}

# Show usage instructions
show_instructions() {
    echo
    log_success "Installation completed!"
    echo
    echo "=== Usage Instructions ==="
    echo
    echo "Service Management:"
    echo "  ./hyprwhspr-service status    # Check service status"
    echo "  ./hyprwhspr-service restart   # Restart service"
    echo "  ./hyprwhspr-service logs      # View logs"
    echo "  ./hyprwhspr-service test      # Test functionality"
    echo
    echo "Hyprwhspr Commands:"
    echo "  hyprwhspr toggle              # Toggle recording"
    echo "  hyprwhspr status              # Check status"
    echo "  hyprwhspr models              # List models"
    echo "  hyprwhspr download <model>    # Download model"
    echo "  hyprwhspr delete <model>      # Delete model"
    echo
    echo "Configuration:"
    echo "  Config file: $HOME/.config/hyprwhspr/config.json"
    echo "  Models directory: $HOME/.local/share/hyprwhspr/"
    echo
    echo "Hyprland Integration:"
    echo "  Add this to ~/.config/hypr/bindings.conf:"
    echo "  bind = SUPER, D, exec, hyprwhspr toggle"
    echo
    echo "For more information, see the README.md file."
}

# Main installation function
main() {
    echo "=== Hyprwhspr SystemD Installation ==="
    echo
    
    check_permissions
    check_dependencies
    
    echo
    read -p "This will install hyprwhspr as a SystemD service. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Installation cancelled"
        exit 0
    fi
    
    build_hyprwhspr
    install_binary
    install_service
    create_config
    download_model
    start_service
    show_instructions
}

# Run main function
main "$@"