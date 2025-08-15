#!/bin/bash

# Migration script for Chariot Ecosystem Monorepo
# Run this from the chariot-ecosystem root directory

set -e

echo "🚀 Starting Chariot Ecosystem Migration..."

# Set paths
VISUAL_DSL_SOURCE="$HOME/go/src/github.com/bhouse1273/visual-dsl"
GO_CHARIOT_SOURCE="$HOME/go/src/github.com/bhouse1273/go-chariot"  # Adjust path as needed
CHARIOTEER_SOURCE="$HOME/go/src/github.com/bhouse1273/charioteer"   # Adjust path as needed

# Verify we're in the right directory
if [ ! -d "services" ]; then
    echo "❌ Error: Run this script from the chariot-ecosystem root directory"
    exit 1
fi

echo "📁 Creating directory structure..."

# Create additional directories
mkdir -p {config,logs}/{visual-dsl,go-chariot,charioteer}
mkdir -p infrastructure/docker/{visual-dsl,go-chariot,charioteer,nginx}
mkdir -p databases/{mysql,couchbase}

echo "📋 Migrating Visual DSL core files..."

# Copy core Visual DSL application files
if [ -d "$VISUAL_DSL_SOURCE" ]; then
    cp -r "$VISUAL_DSL_SOURCE/src" services/visual-dsl/
    cp -r "$VISUAL_DSL_SOURCE/public" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/package.json" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/package-lock.json" services/visual-dsl/ 2>/dev/null || echo "No package-lock.json found"
    cp "$VISUAL_DSL_SOURCE/tsconfig.json" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/vite.config.js" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/tailwind.config.js" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/postcss.config.js" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/eslint.config.js" services/visual-dsl/
    cp "$VISUAL_DSL_SOURCE/index.html" services/visual-dsl/
    
    # Copy modified/new components
    echo "  ✅ Copied Visual DSL core files"
else
    echo "  ⚠️  Visual DSL source not found at $VISUAL_DSL_SOURCE"
fi

echo "🗄️  Setting up database configurations..."

# Copy database configurations if they exist
if [ -f "$VISUAL_DSL_SOURCE/docker/mysql/init.sql" ]; then
    cp "$VISUAL_DSL_SOURCE/docker/mysql/init.sql" databases/mysql/
    echo "  ✅ Copied MySQL init.sql"
fi

if [ -f "$VISUAL_DSL_SOURCE/docker/mysql/my.cnf" ]; then
    cp "$VISUAL_DSL_SOURCE/docker/mysql/my.cnf" databases/mysql/
    echo "  ✅ Copied MySQL config"
fi

if [ -f "$VISUAL_DSL_SOURCE/docker/couchbase/init.sh" ]; then
    cp "$VISUAL_DSL_SOURCE/docker/couchbase/init.sh" databases/couchbase/
    echo "  ✅ Copied Couchbase init script"
fi

echo "🐳 Setting up Docker configurations..."

# Copy the monorepo docker-compose file
if [ -f "$VISUAL_DSL_SOURCE/monorepo-docker-compose.yml" ]; then
    cp "$VISUAL_DSL_SOURCE/monorepo-docker-compose.yml" docker-compose.yml
    echo "  ✅ Copied main docker-compose.yml"
fi

echo "📖 Setting up documentation..."

# Copy and merge documentation
if [ -f "$VISUAL_DSL_SOURCE/MIGRATION_PLAN.md" ]; then
    cp "$VISUAL_DSL_SOURCE/MIGRATION_PLAN.md" docs/
    echo "  ✅ Copied migration plan"
fi

echo "🔧 Go-Chariot Service Migration..."

# Copy go-chariot repository if it exists
if [ -d "$GO_CHARIOT_SOURCE" ]; then
    echo "  📋 Copying go-chariot repository..."
    cp -r "$GO_CHARIOT_SOURCE/"* services/go-chariot/
    echo "  ✅ Copied go-chariot service"
else
    echo "  ⚠️  Go-Chariot source not found at $GO_CHARIOT_SOURCE"
    echo "     You'll need to manually copy the go-chariot repository"
fi

echo "⚙️  Charioteer Service Migration..."

# Copy charioteer repository if it exists
if [ -d "$CHARIOTEER_SOURCE" ]; then
    echo "  📋 Copying charioteer repository..."
    cp -r "$CHARIOTEER_SOURCE/"* services/charioteer/
    echo "  ✅ Copied charioteer service"
else
    echo "  ⚠️  Charioteer source not found at $CHARIOTEER_SOURCE"
    echo "     You'll need to manually copy the charioteer repository"
fi

echo "📝 Creating development scripts..."

# Create basic development scripts
cat > scripts/dev-start.sh << 'EOF'
#!/bin/bash
# Start the full development environment
echo "🚀 Starting Chariot Ecosystem Development Environment..."
docker compose up --build
EOF

cat > scripts/build-all.sh << 'EOF'
#!/bin/bash
# Build all services
echo "🔨 Building all Chariot services..."
docker compose build
EOF

chmod +x scripts/*.sh

echo "📄 Creating initial .gitignore..."

cat > .gitignore << 'EOF'
# Dependencies
node_modules/
*/node_modules/

# Environment files
.env
.env.local
.env.*.local

# Logs
logs/
*.log
npm-debug.log*

# Docker
.docker/

# OS files
.DS_Store
Thumbs.db

# IDE files
.vscode/settings.json
.idea/

# Build outputs
dist/
build/
target/

# Go
bin/
vendor/

# SSL certificates (generated)
certs/
*.pem
*.crt
*.key

# Database data
data/
EOF

echo "✅ Migration completed!"
echo ""
echo "Next steps:"
echo "1. Update paths in go-chariot and charioteer sources if needed"
echo "2. Create Dockerfiles in infrastructure/docker/ directories"
echo "3. Test with: docker compose up --build"
echo "4. Update documentation"
echo ""
echo "Key changes in monorepo:"
echo "- Host networking (no bridge network conflicts)"
echo "- Simplified service discovery (localhost URLs)"
echo "- Centralized Azure configuration"
echo "- Clean separation of services and infrastructure"
