#!/bin/bash

# Simple Health Check for SSTS
# Quick verification that all services are working

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔍 SSTS Health Check"
echo "===================="

# Check if Docker is running
if ! docker ps >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    exit 1
fi

# Check container status
echo "📊 Container Status:"
docker-compose ps --format table

echo
echo "🌐 Service Connectivity:"

# Test each service
check_url() {
    local name="$1"
    local url="$2"
    
    if curl -s -f --max-time 5 "$url" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✅ $name${NC} - $url"
        return 0
    else
        echo -e "  ${RED}❌ $name${NC} - $url"
        return 1
    fi
}

# Check all services
all_good=true

check_url "SSTS Application" "http://localhost:8080/health" || all_good=false
check_url "Grafana Dashboard" "http://localhost:3000/api/health" || all_good=false
check_url "Prometheus" "http://localhost:9090/-/healthy" || all_good=false
check_url "InfluxDB" "http://localhost:8086/health" || all_good=false

# Check databases
echo
echo "🗄️  Database Connectivity:"

if docker-compose exec -T postgres pg_isready -U ssts >/dev/null 2>&1; then
    echo -e "  ${GREEN}✅ PostgreSQL${NC}"
else
    echo -e "  ${RED}❌ PostgreSQL${NC}"
    all_good=false
fi

if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
    echo -e "  ${GREEN}✅ Redis${NC}"
else
    echo -e "  ${RED}❌ Redis${NC}"
    all_good=false
fi

echo
if [ "$all_good" = true ]; then
    echo -e "${GREEN}🎉 All services are healthy!${NC}"
    echo
    echo "🚀 Access your SSTS system:"
    echo "   📊 Main Dashboard: http://localhost:8080"
    echo "   📈 Grafana:        http://localhost:3000 (admin/admin)"
    echo "   📊 Prometheus:     http://localhost:9090"
    echo "   🗄️  InfluxDB:       http://localhost:8086"
    echo
    echo "✨ Connection issues resolved! Your system is ready to use."
else
    echo -e "${RED}⚠️  Some services are not responding${NC}"
    echo
    echo "🔧 Troubleshooting:"
    echo "   1. View logs: docker-compose logs [service-name]"
    echo "   2. Restart:   ./quick-deploy.sh restart"
    echo "   3. Status:    docker-compose ps"
    exit 1
fi