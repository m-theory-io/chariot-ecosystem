#!/bin/bash

# Check status of all Chariot Ecosystem services
echo "🔍 Checking Chariot Ecosystem Services Status..."
echo "==============================================="

# Function to check service health
check_service() {
    local service_name=$1
    local url=$2
    local expected_pattern=$3
    
    echo -n "🔧 $service_name: "
    
    if curl -s --max-time 5 "$url" | grep -q "$expected_pattern" 2>/dev/null; then
        echo "✅ Running"
    else
        echo "❌ Not ready"
    fi
}

# Check databases
echo ""
echo "📊 Database Services:"
check_service "MySQL" "http://localhost:3306" "mysql" || echo "   MySQL: ❌ Not ready (connection refused is expected)"
check_service "Couchbase" "http://localhost:8091/ui/index.html" "Couchbase"

echo ""
echo "🚀 Application Services:"
check_service "Charioteer" "http://localhost:8080/health" "OK"
check_service "Go-Chariot" "http://localhost:9080/health" "OK"
check_service "Visual DSL" "http://localhost/" "html"

echo ""
echo "📈 Container Status:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep chariot

echo ""
echo "💡 Service URLs:"
echo "  🎨 Visual DSL Frontend: http://localhost/"
echo "  🎯 Charioteer: http://localhost:8080/"
echo "  ⚡ Go-Chariot API: http://localhost:9080/"
echo "  📊 Couchbase Admin: http://localhost:8091/"
echo "  🗄️  MySQL: localhost:3306"
