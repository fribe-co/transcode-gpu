#!/bin/bash
# Check nvidia-smi location on host system

echo "=== Checking nvidia-smi location on host ==="

# Check if nvidia-smi exists
if command -v nvidia-smi &> /dev/null; then
    echo "✓ nvidia-smi found in PATH"
    which nvidia-smi
    nvidia-smi --version | head -1
else
    echo "✗ nvidia-smi not found in PATH"
fi

echo ""
echo "=== Checking common nvidia-smi locations ==="

locations=(
    "/usr/bin/nvidia-smi"
    "/usr/local/bin/nvidia-smi"
    "/usr/local/cuda/bin/nvidia-smi"
    "/opt/nvidia/bin/nvidia-smi"
)

found=0
for loc in "${locations[@]}"; do
    if [ -f "$loc" ]; then
        echo "✓ Found: $loc"
        ls -lh "$loc"
        found=1
    else
        echo "✗ Not found: $loc"
    fi
done

if [ $found -eq 0 ]; then
    echo ""
    echo "WARNING: nvidia-smi not found in common locations!"
    echo "Please find nvidia-smi location manually:"
    echo "  find /usr -name nvidia-smi 2>/dev/null"
    echo "  find /opt -name nvidia-smi 2>/dev/null"
else
    echo ""
    echo "=== Recommendation ==="
    echo "Update docker-compose.prod.yml volume mount with the correct path:"
    echo "  - [NVIDIA_SMI_PATH]:/usr/bin/nvidia-smi:ro"
fi
