#!/bin/bash

# Test cross-platform build capabilities
echo "🧪 Testing cross-platform build capabilities..."

# Check Docker buildx
echo "Checking Docker buildx..."
if ! docker buildx version &> /dev/null; then
    echo "❌ Docker buildx not available"
    exit 1
fi

echo "✅ Docker buildx available"

# Check available platforms
echo "Available builder platforms:"
docker buildx inspect --bootstrap | grep Platforms

# Test a simple cross-platform build
echo "Testing simple cross-platform build..."
docker buildx build --platform linux/amd64 -t test-cross-platform - <<EOF
FROM --platform=linux/amd64 alpine:latest
RUN echo "Building for AMD64"
EOF

if [ $? -eq 0 ]; then
    echo "✅ Cross-platform build test successful"
    docker rmi test-cross-platform 2>/dev/null
else
    echo "❌ Cross-platform build test failed"
    exit 1
fi

echo "🎉 Cross-platform build capabilities verified!"
