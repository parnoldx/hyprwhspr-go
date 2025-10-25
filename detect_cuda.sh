#!/bin/bash
# Detect if CUDA is available on the system

# Check if nvcc (CUDA compiler) is available
if command -v nvcc &> /dev/null; then
    echo "CUDA_AVAILABLE=1"
    exit 0
fi

# Check if nvidia-smi is available (NVIDIA driver installed)
if command -v nvidia-smi &> /dev/null; then
    echo "CUDA_AVAILABLE=1"
    exit 0
fi

# Check for CUDA libraries in common locations
if [ -d "/usr/local/cuda" ] || [ -d "/opt/cuda" ]; then
    echo "CUDA_AVAILABLE=1"
    exit 0
fi

# No CUDA detected
echo "CUDA_AVAILABLE=0"
exit 0
