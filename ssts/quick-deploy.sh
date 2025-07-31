#!/bin/bash

# SSTS Quick Deploy Script - Fixes Common Connection Issues
# This script provides a reliable deployment with proper error handling

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Pre-deployment checks
pre_checks() {
    log_info "Running pre-deployment checks..."
    
    # Check Docker
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker Desktop and try again."
        exit 1
    fi
    log_success "Docker is running"
    
    # Check Docker Compose
    if ! docker-compose --version >/dev/null 2>&1; then
        log_error "Docker Compose is not available"
        exit 1
    fi
    log_success "Docker Compose is available"
    
    # Create necessary directories
    mkdir -p data logs/nginx
    log_success "Directories created"
}

# Clean up existing containers
cleanup() {
    log_info "Cleaning up existing containers..."
    docker-compose down --remove-orphans >/dev/null 2>&1 || true
    
    # Remove any dangling containers
    docker container prune -f >/dev/null 2>&1 || true
    
    log_success "Cleanup completed"
}

# Build and start services
deploy() {
    log_info "Building and starting SSTS services..."
    
    # Pull latest images first
    log_info "Pulling base images..."
    docker-compose pull postgres redis influxdb grafana prometheus
    
    # Build our application
    log_info "Building SSTS application..."
    docker-compose build --no-cache ssts
    
    # Start services in order
    log_info "Starting core services..."
    docker-compose up -d postgres redis influxdb
    
    # Wait for databases to be ready
    log_info "Waiting for databases to be ready..."
    sleep 15
    
    # Check database health
    local retries=30
    while [ $retries -gt 0 ]; do
        if docker-compose exec -T postgres pg_isready -U ssts >/dev/null 2>&1 && \
           docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
            log_success "Databases are ready"
            break
        fi
        retries=$((retries-1))
        log_info "Waiting for databases... ($retries attempts left)"
        sleep 2
    done
    
    if [ $retries -eq 0 ]; then
        log_error "Databases failed to start properly"
        exit 1
    fi
    
    # Start application services
    log_info "Starting application services..."
    docker-compose up -d ssts grafana prometheus
    
    # Wait for application to be ready
    log_info "Waiting for SSTS application to be ready..."
    sleep 10
    
    local app_retries=30
    while [ $app_retries -gt 0 ]; do
        if curl -f -s http://localhost:8080/health >/dev/null 2>&1; then
            log_success "SSTS application is ready"
            break
        fi
        app_retries=$((app_retries-1))
        log_info "Waiting for SSTS application... ($app_retries attempts left)"
        sleep 2
    done
    
    # Start nginx last
    # log_info "Starting reverse proxy..."
    # docker-compose up -d nginx
    
    log_success "All services started successfully!"
}

# Verify deployment
verify() {
    log_info "Verifying deployment..."
    
    local all_healthy=true
    
    # Check each service
    services=("ssts:8080/health" "grafana:3000/api/health" "prometheus:9090/-/healthy" "influxdb:8086/health")
    
    for service in "${services[@]}"; do
        name="${service%%:*}"
        url="http://localhost:${service#*:}"
        
        if curl -f -s --max-time 10 "$url" >/dev/null 2>&1; then
            log_success "$name is healthy"
        else
            log_error "$name is not responding"
            all_healthy=false
        fi
    done
    
    # Check database connectivity
    if docker-compose exec -T postgres pg_isready -U ssts >/dev/null 2>&1; then
        log_success "PostgreSQL is accessible"
    else
        log_error "PostgreSQL is not accessible"
        all_healthy=false
    fi
    
    if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
        log_success "Redis is accessible"
    else
        log_error "Redis is not accessible"
        all_healthy=false
    fi
    
    if [ "$all_healthy" = true ]; then
        echo
        log_success "🎉 SSTS deployment is fully operational!"
        echo
        echo "Access your services:"
        echo "  📊 SSTS Dashboard:  http://localhost:8080"
        echo "  📈 Grafana:         http://localhost:3000 (admin/admin)"
        echo "  📊 Prometheus:      http://localhost:9090"
        echo "  🗄️  InfluxDB:        http://localhost:8086"
        echo "  🔧 Health Check:    http://localhost:8080/health"
        echo
        echo "Troubleshooting commands:"
        echo "  📋 View logs:       docker-compose logs -f"
        echo "  📊 Service status:  docker-compose ps"
        echo "  🔄 Restart:         ./quick-deploy.sh"
        echo
    else
        log_error "Some services are not healthy. Check logs with: docker-compose logs"
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "SSTS Quick Deploy - Reliable Deployment Script"
    echo
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  deploy    Deploy SSTS (default)"
    echo "  stop      Stop all services"
    echo "  restart   Restart all services"
    echo "  status    Show service status"
    echo "  logs      Show logs"
    echo "  clean     Clean up everything"
    echo "  help      Show this help"
    echo
}

# Main execution
main() {
    local command="${1:-deploy}"
    
    case "$command" in
        "deploy"|"start"|"up"|"")
            echo "🚀 SSTS Quick Deploy Starting..."
            echo
            pre_checks
            cleanup
            deploy
            verify
            ;;
        "stop"|"down")
            log_info "Stopping SSTS services..."
            docker-compose down
            log_success "Services stopped"
            ;;
        "restart")
            log_info "Restarting SSTS services..."
            docker-compose down
            sleep 2
            main deploy
            ;;
        "status")
            echo "Service Status:"
            docker-compose ps
            echo
            echo "Container Health:"
            docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
            ;;
        "logs")
            docker-compose logs -f
            ;;
        "clean")
            log_warning "This will remove all containers and data. Are you sure? (y/N)"
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                docker-compose down -v --remove-orphans
                docker system prune -f
                log_success "Cleanup completed"
            fi
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"