#!/bin/bash

# SSTS Deployment Script
# This script handles the deployment of SSTS in different environments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEFAULT_ENV="dev"
DEFAULT_MODE="docker"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
SSTS Deployment Script

Usage: $0 [OPTIONS] COMMAND

COMMANDS:
    up              Start all services
    down            Stop all services
    restart         Restart all services
    build           Build the application
    logs            Show logs
    status          Show service status
    clean           Clean up resources
    test            Run tests
    backup          Backup databases
    restore         Restore databases

OPTIONS:
    -e, --env       Environment (dev, staging, prod) [default: dev]
    -m, --mode      Deployment mode (docker, k8s) [default: docker]
    -h, --help      Show this help message

EXAMPLES:
    $0 up                           # Start services in dev environment
    $0 -e prod -m k8s up           # Start services in production using Kubernetes
    $0 down                         # Stop all services
    $0 logs ssts                    # Show logs for SSTS service
    $0 backup                       # Backup databases

EOF
}

# Parse command line arguments
parse_args() {
    ENVIRONMENT="$DEFAULT_ENV"
    MODE="$DEFAULT_MODE"
    COMMAND=""
    SERVICE=""

    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--env)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -m|--mode)
                MODE="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            up|down|restart|build|logs|status|clean|test|backup|restore)
                COMMAND="$1"
                shift
                ;;
            *)
                if [[ -z "$SERVICE" && "$COMMAND" == "logs" ]]; then
                    SERVICE="$1"
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$COMMAND" ]]; then
        log_error "No command specified"
        show_help
        exit 1
    fi
}

# Docker functions
docker_up() {
    log_info "Starting SSTS services with Docker Compose..."
    cd "$PROJECT_ROOT"
    
    # Create necessary directories
    mkdir -p data logs/nginx
    
    # Start services
    docker-compose up -d
    
    log_success "Services started successfully!"
    log_info "SSTS UI: http://localhost:8080"
    log_info "Grafana: http://localhost:3000 (admin/admin)"
    log_info "Prometheus: http://localhost:9090"
}

docker_down() {
    log_info "Stopping SSTS services..."
    cd "$PROJECT_ROOT"
    docker-compose down
    log_success "Services stopped successfully!"
}

docker_restart() {
    docker_down
    docker_up
}

docker_build() {
    log_info "Building SSTS application..."
    cd "$PROJECT_ROOT"
    docker-compose build --no-cache
    log_success "Build completed successfully!"
}

docker_logs() {
    cd "$PROJECT_ROOT"
    if [[ -n "$SERVICE" ]]; then
        docker-compose logs -f "$SERVICE"
    else
        docker-compose logs -f
    fi
}

docker_status() {
    log_info "Service status:"
    cd "$PROJECT_ROOT"
    docker-compose ps
}

docker_clean() {
    log_warning "This will remove all containers, volumes, and images. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        cd "$PROJECT_ROOT"
        docker-compose down -v --remove-orphans
        docker system prune -af --volumes
        log_success "Cleanup completed!"
    else
        log_info "Cleanup cancelled."
    fi
}

# Kubernetes functions
k8s_up() {
    log_info "Deploying SSTS to Kubernetes ($ENVIRONMENT environment)..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Apply Kubernetes manifests
    kubectl apply -k "$PROJECT_ROOT/k8s/overlays/$ENVIRONMENT"
    
    log_success "SSTS deployed to Kubernetes!"
    log_info "Use 'kubectl get pods -n ssts-$ENVIRONMENT' to check status"
}

k8s_down() {
    log_info "Removing SSTS from Kubernetes..."
    kubectl delete -k "$PROJECT_ROOT/k8s/overlays/$ENVIRONMENT" --ignore-not-found=true
    log_success "SSTS removed from Kubernetes!"
}

k8s_restart() {
    log_info "Restarting SSTS deployment..."
    kubectl rollout restart deployment/ssts-app -n "ssts-$ENVIRONMENT"
    kubectl rollout status deployment/ssts-app -n "ssts-$ENVIRONMENT"
    log_success "Deployment restarted!"
}

k8s_logs() {
    local namespace="ssts-$ENVIRONMENT"
    if [[ -n "$SERVICE" ]]; then
        kubectl logs -f -l app="$SERVICE" -n "$namespace"
    else
        kubectl logs -f -l app.kubernetes.io/name=ssts -n "$namespace"
    fi
}

k8s_status() {
    local namespace="ssts-$ENVIRONMENT"
    log_info "Kubernetes resources in namespace $namespace:"
    kubectl get all -n "$namespace"
}

# Database backup/restore functions
backup_databases() {
    log_info "Backing up databases..."
    
    local backup_dir="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    if [[ "$MODE" == "docker" ]]; then
        # Backup PostgreSQL
        docker-compose exec -T postgres pg_dump -U ssts ssts > "$backup_dir/postgres_backup.sql"
        
        # Backup InfluxDB
        docker-compose exec -T influxdb influx backup "$backup_dir/influxdb_backup"
        
        log_success "Databases backed up to $backup_dir"
    else
        log_warning "Database backup for Kubernetes mode not implemented yet"
    fi
}

restore_databases() {
    log_warning "This will restore databases from backup. Current data will be lost!"
    log_warning "Please specify the backup directory path:"
    read -r backup_path
    
    if [[ ! -d "$backup_path" ]]; then
        log_error "Backup directory does not exist: $backup_path"
        exit 1
    fi
    
    log_info "Restoring databases from $backup_path..."
    
    if [[ "$MODE" == "docker" ]]; then
        # Restore PostgreSQL
        if [[ -f "$backup_path/postgres_backup.sql" ]]; then
            docker-compose exec -T postgres psql -U ssts ssts < "$backup_path/postgres_backup.sql"
            log_success "PostgreSQL restored"
        fi
        
        # Restore InfluxDB
        if [[ -d "$backup_path/influxdb_backup" ]]; then
            docker-compose exec -T influxdb influx restore "$backup_path/influxdb_backup"
            log_success "InfluxDB restored"
        fi
    else
        log_warning "Database restore for Kubernetes mode not implemented yet"
    fi
}

# Test functions
run_tests() {
    log_info "Running tests..."
    
    cd "$PROJECT_ROOT"
    
    # Start test services
    if [[ "$MODE" == "docker" ]]; then
        docker-compose -f docker-compose.test.yml up -d postgres redis
        sleep 5
        
        # Run Go tests
        docker-compose -f docker-compose.test.yml run --rm ssts-test go test -v ./...
        
        # Run web tests
        docker-compose -f docker-compose.test.yml run --rm web-test npm test
        
        # Cleanup
        docker-compose -f docker-compose.test.yml down -v
    else
        log_warning "Tests for Kubernetes mode not implemented yet"
    fi
    
    log_success "Tests completed!"
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Environment: $ENVIRONMENT"
    log_info "Mode: $MODE"
    log_info "Command: $COMMAND"
    
    case "$MODE" in
        docker)
            case "$COMMAND" in
                up) docker_up ;;
                down) docker_down ;;
                restart) docker_restart ;;
                build) docker_build ;;
                logs) docker_logs ;;
                status) docker_status ;;
                clean) docker_clean ;;
                test) run_tests ;;
                backup) backup_databases ;;
                restore) restore_databases ;;
                *) log_error "Unknown command: $COMMAND" ;;
            esac
            ;;
        k8s)
            case "$COMMAND" in
                up) k8s_up ;;
                down) k8s_down ;;
                restart) k8s_restart ;;
                logs) k8s_logs ;;
                status) k8s_status ;;
                test) run_tests ;;
                backup) backup_databases ;;
                restore) restore_databases ;;
                *) log_error "Unknown command: $COMMAND" ;;
            esac
            ;;
        *)
            log_error "Unknown mode: $MODE"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"