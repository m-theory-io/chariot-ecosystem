#!/bin/bash

# Sync script for Option A: StudioSSD as Primary Development Environment
# This syncs changes FROM StudioSSD back TO Jetson internal SSD for git operations

set -e

echo "🔄 Syncing chariot-ecosystem from StudioSSD to Jetson internal SSD..."

SOURCE="/media/nvidia/StudioSSD/go/src/github.com/bhouse1273/chariot-ecosystem/"
DEST="/home/nvidia/go/src/github.com/bhouse1273/chariot-ecosystem/"

echo "📂 Source: $SOURCE"
echo "📂 Destination: $DEST"

# Sync files, excluding build artifacts and temporary files
rsync -av \
    --exclude='bin/' \
    --exclude='build/' \
    --exclude='*.log' \
    --exclude='node_modules/' \
    --exclude='.git/' \
    --delete \
    "$SOURCE" "$DEST"

echo "✅ Sync complete!"
echo ""
echo "💡 Next steps:"
echo "   cd /home/nvidia/go/src/github.com/bhouse1273/chariot-ecosystem"
echo "   git add ."
echo "   git commit -m 'Your commit message'"
echo "   git push"
