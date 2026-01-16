#!/bin/bash
# Create /usr/local/bin/nvidia-smi symlink on host if it doesn't exist
# This helps NVIDIA Container Toolkit mount it correctly

set -e

echo "=== Creating nvidia-smi symlink on host ==="

# Check if /usr/bin/nvidia-smi exists
if [ ! -f "/usr/bin/nvidia-smi" ]; then
    echo "ERROR: /usr/bin/nvidia-smi not found on host"
    echo "Please ensure NVIDIA drivers are installed"
    exit 1
fi

echo "✓ Found /usr/bin/nvidia-smi"

# Create /usr/local/bin directory if it doesn't exist
if [ ! -d "/usr/local/bin" ]; then
    echo "Creating /usr/local/bin directory..."
    sudo mkdir -p /usr/local/bin
fi

# Check what /usr/local/bin/nvidia-smi is (file, symlink, directory, or doesn't exist)
if [ -e "/usr/local/bin/nvidia-smi" ]; then
    if [ -L "/usr/local/bin/nvidia-smi" ]; then
        echo "✓ /usr/local/bin/nvidia-smi is already a symlink"
        ls -la /usr/local/bin/nvidia-smi
    elif [ -d "/usr/local/bin/nvidia-smi" ]; then
        echo "⚠ WARNING: /usr/local/bin/nvidia-smi is a directory (this causes the mount error!)"
        echo "Removing directory and creating symlink..."
        sudo rm -rf /usr/local/bin/nvidia-smi
        sudo ln -sf /usr/bin/nvidia-smi /usr/local/bin/nvidia-smi
        echo "✓ Directory removed and symlink created"
        ls -la /usr/local/bin/nvidia-smi
    elif [ -f "/usr/local/bin/nvidia-smi" ]; then
        echo "⚠ WARNING: /usr/local/bin/nvidia-smi exists as a regular file"
        echo "Backing up and creating symlink..."
        sudo mv /usr/local/bin/nvidia-smi /usr/local/bin/nvidia-smi.backup
        sudo ln -sf /usr/bin/nvidia-smi /usr/local/bin/nvidia-smi
        echo "✓ File backed up and symlink created"
        ls -la /usr/local/bin/nvidia-smi
    fi
else
    echo "Creating symlink: /usr/local/bin/nvidia-smi -> /usr/bin/nvidia-smi"
    sudo ln -sf /usr/bin/nvidia-smi /usr/local/bin/nvidia-smi
    echo "✓ Symlink created"
    ls -la /usr/local/bin/nvidia-smi
fi

echo ""
echo "=== Verification ==="
if [ -L "/usr/local/bin/nvidia-smi" ]; then
    echo "✓ /usr/local/bin/nvidia-smi symlink is working"
    /usr/local/bin/nvidia-smi --version | head -1
else
    echo "✗ /usr/local/bin/nvidia-smi symlink could not be created"
    exit 1
fi

echo ""
echo "=== Next Steps ==="
echo "1. Restart Docker containers:"
echo "   cd ~/transcode-gpu"
echo "   docker-compose -f docker/docker-compose.prod.yml down"
echo "   docker-compose -f docker/docker-compose.prod.yml up -d"
echo ""
echo "2. Test GPU access:"
echo "   docker exec -it cashbacktv-backend nvidia-smi --list-gpus"
