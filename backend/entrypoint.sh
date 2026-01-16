#!/bin/bash
set -e

# Create /usr/local/bin directory if it doesn't exist
mkdir -p /usr/local/bin

# Create symlink for nvidia-smi if /usr/bin/nvidia-smi exists and symlink doesn't
if [ -f "/usr/bin/nvidia-smi" ] && [ ! -e "/usr/local/bin/nvidia-smi" ]; then
    echo "Creating symlink: /usr/local/bin/nvidia-smi -> /usr/bin/nvidia-smi"
    ln -sf /usr/bin/nvidia-smi /usr/local/bin/nvidia-smi
fi

# Execute the main command
exec "$@"
