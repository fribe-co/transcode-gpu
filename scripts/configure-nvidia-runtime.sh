#!/bin/bash
# NVIDIA Container Toolkit Runtime Configuration Script
# This script configures Docker to use NVIDIA runtime

set -e

echo "=== NVIDIA Container Toolkit Runtime Configuration ==="

# Check if nvidia-ctk is installed
if ! command -v nvidia-ctk &> /dev/null; then
    echo "ERROR: nvidia-ctk command not found. Please install NVIDIA Container Toolkit first."
    echo "Run: scripts/install-nvidia-container-toolkit.sh"
    exit 1
fi

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo "ERROR: docker command not found. Please install Docker first."
    exit 1
fi

echo "1. Configuring NVIDIA runtime using nvidia-ctk..."
sudo nvidia-ctk runtime configure --runtime=docker

echo "2. Checking Docker daemon.json configuration..."
if [ -f /etc/docker/daemon.json ]; then
    echo "Docker daemon.json exists. Content:"
    cat /etc/docker/daemon.json
else
    echo "WARNING: /etc/docker/daemon.json does not exist. nvidia-ctk should create it."
fi

echo "3. Restarting Docker service..."
sudo systemctl restart docker

echo "4. Verifying NVIDIA runtime is configured..."
if docker info | grep -i nvidia &> /dev/null; then
    echo "✓ NVIDIA runtime is configured in Docker"
    docker info | grep -i nvidia
else
    echo "⚠ WARNING: NVIDIA runtime not found in Docker info. Please check the configuration."
fi

echo ""
echo "=== Configuration Complete ==="
echo "Next steps:"
echo "1. Verify GPU access: docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi"
echo "2. Restart your docker-compose services: docker-compose -f docker/docker-compose.prod.yml up -d"
