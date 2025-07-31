#!/bin/bash

# SSTS Health Check Script
# Performs comprehensive health checks on all services

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
TIMEOUT=10
RETRIES=3

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Health check function
check_service() {
    local service_name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    
    log_info "Checking $service_name..."
    
    for i in $(seq 1 $RETRIES); do
        if curl -f -s --max-time $TIMEOUT "$url" > /dev/null 2>&1; then
            log_success "$service_name is healthy"
            return 0
        fi
        
        if [[ $i -lt $RETRIES ]]; then
            log_warning "$service_name not ready, retrying in 5 seconds... ($i/$RETRIES)"
            sleep 5
        fi
    done
    
    log_error "$service_name is not healthy"
    return 1
}

# Database connectivity check
check_database() {
    local service_name="$1"
    local command="$2"
    
    log_info "Checking $service_name connectivity..."
    
    if eval "$command" > /dev/null 2>&1; then
        log_success "$service_name is accessible"
        return 0
    else
        log_error "$service_name is not accessible"
        return 1
    fi
}

# Main health check
main() {
    log_info "Starting SSTS health check..."
    echo
    
    local overall_status=0
    
    # Check SSTS main application
    if ! check_service "SSTS Application" "http://localhost:8080/health"; then
        overall_status=1
    fi
    
    # Check Grafana
    if ! check_service "Grafana" "http://localhost:3000/api/health"; then
        overall_status=1
    fi
    
    # Check Prometheus
    if ! check_service "Prometheus" "http://localhost:9090/-/healthy"; then
        overall_status=1
    fi
    
    # Check PostgreSQL
    if ! check_database "PostgreSQL" "docker-compose exec -T postgres pg_isready -U ssts"; then
        overall_status=1
    fi
    
    # Check Redis
    if ! check_database "Redis" "docker-compose exec -T redis redis-cli ping"; then
        overall_status=1
    fi
    
    # Check InfluxDB
    if ! check_service "InfluxDB" "http://localhost:8086/health"; then
        overall_status=1
    fi
    
    echo
    if [[ $overall_status -eq 0 ]]; then
        log_success "All services are healthy!"
        echo
        log_info "Service URLs:"
        echo "  SSTS UI:        http://localhost:8080"
        echo "  Grafana:        http://localhost:3000 (admin/admin)"
        echo "  Prometheus:     http://localhost:9090"
        echo "  InfluxDB UI:    http://localhost:8086"
    else
        log_error "Some services are not healthy!"
        echo
        log_info "Troubleshooting:"
        echo "  1. Check service logs: docker-compose logs [service]"
        echo "  2. Restart services: ./scripts/deploy.sh restart"
        echo "  3. Check service status: docker-compose ps"
        exit 1
    fi
}

main "$@"