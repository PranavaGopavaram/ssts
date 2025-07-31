# SSTS Deployment Guide

## Overview

This guide covers the deployment of the System Stress Testing Suite (SSTS) in various environments using Docker Compose and Kubernetes.

## Prerequisites

### For Docker Deployment
- Docker 20.10+
- Docker Compose 2.0+
- 4GB+ RAM available
- 10GB+ disk space

### For Kubernetes Deployment
- Kubernetes cluster 1.20+
- kubectl configured
- Helm 3.0+ (optional)
- Persistent storage available

## Quick Start (Docker)

1. **Clone and navigate to the project:**
   ```bash
   git clone <repository-url>
   cd ssts
   ```

2. **Start all services:**
   ```bash
   ./scripts/deploy.sh up
   ```

3. **Check service health:**
   ```bash
   ./scripts/health-check.sh
   ```

4. **Access the application:**
   - SSTS UI: http://localhost:8080
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090

## Docker Deployment

### Architecture

The Docker deployment includes:
- **SSTS Application**: Main stress testing service
- **PostgreSQL**: Primary database for test data
- **InfluxDB**: Time-series database for metrics
- **Redis**: Caching and session storage
- **Grafana**: Metrics visualization
- **Prometheus**: Monitoring and alerting
- **Nginx**: Reverse proxy and load balancer

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SSTS_LOG_LEVEL` | Log level (debug, info, warn, error) | info |
| `SSTS_DATABASE_TYPE` | Database type (postgres, sqlite) | postgres |
| `SSTS_DATABASE_HOST` | Database host | postgres |
| `SSTS_DATABASE_USERNAME` | Database username | ssts |
| `SSTS_DATABASE_PASSWORD` | Database password | ssts_password |
| `SSTS_INFLUXDB_URL` | InfluxDB URL | http://influxdb:8086 |
| `SSTS_INFLUXDB_TOKEN` | InfluxDB authentication token | admin-token |
| `SSTS_REDIS_ADDRESS` | Redis connection string | redis:6379 |

### Volume Mounts

- `./data`: Application data directory
- `./logs`: Application and Nginx logs
- `postgres_data`: PostgreSQL data
- `influxdb_data`: InfluxDB data
- `redis_data`: Redis data
- `grafana_data`: Grafana configuration and dashboards

### Commands

```bash
# Start services
./scripts/deploy.sh up

# Stop services
./scripts/deploy.sh down

# Restart services
./scripts/deploy.sh restart

# Build application
./scripts/deploy.sh build

# View logs
./scripts/deploy.sh logs [service]

# Check status
./scripts/deploy.sh status

# Clean up
./scripts/deploy.sh clean
```

## Kubernetes Deployment

### Architecture

The Kubernetes deployment uses:
- **Namespace**: Isolated environment per deployment
- **Deployments**: For stateless services (SSTS app, Redis)
- **StatefulSets**: For stateful services (PostgreSQL, InfluxDB)
- **Services**: For service discovery
- **ConfigMaps**: For configuration
- **Secrets**: For sensitive data
- **PVCs**: For persistent storage
- **Ingress**: For external access

### Environment-Specific Deployments

#### Development
```bash
./scripts/deploy.sh -e dev -m k8s up
```

#### Staging
```bash
./scripts/deploy.sh -e staging -m k8s up
```

#### Production
```bash
./scripts/deploy.sh -e prod -m k8s up
```

### Monitoring Resources

Monitor your deployment:
```bash
# Check pods
kubectl get pods -n ssts-prod

# Check services
kubectl get services -n ssts-prod

# Check ingress
kubectl get ingress -n ssts-prod

# View logs
kubectl logs -f deployment/ssts-app -n ssts-prod
```

## Configuration

### Application Configuration

The main configuration file is `ssts.yaml`. Key sections:

```yaml
server:
  address: "0.0.0.0"
  port: 8080
  
database:
  type: "postgres"
  host: "postgres-service"
  
influxdb:
  url: "http://influxdb-service:8086"
  org: "ssts"
  bucket: "metrics"
  
safety:
  global_limits:
    max_cpu_percent: 80.0
    max_memory_percent: 70.0
```

### Database Migration

Database schema is automatically created on startup. For manual migration:

```bash
# Docker
docker-compose exec ssts ./ssts migrate

# Kubernetes
kubectl exec -it deployment/ssts-app -n ssts-prod -- ./ssts migrate
```

## Security Considerations

### Docker Security
- Non-root user in containers
- Read-only root filesystem where possible
- Security scanning with Trivy
- Network isolation with custom bridge

### Kubernetes Security
- RBAC enabled
- Network policies for pod-to-pod communication
- Security contexts for containers
- Secrets management for sensitive data

### General Security
- TLS termination at ingress/proxy level
- Authentication and authorization (when enabled)
- Regular security updates
- Audit logging

## Backup and Recovery

### Database Backup
```bash
# Create backup
./scripts/deploy.sh backup

# Restore from backup
./scripts/deploy.sh restore
```

### Disaster Recovery
1. Regular automated backups
2. Infrastructure as Code for quick recreation
3. Monitoring and alerting for early detection
4. Documented recovery procedures

## Performance Tuning

### Resource Allocation
- **SSTS App**: 512Mi memory, 500m CPU (prod)
- **PostgreSQL**: 512Mi memory, 250m CPU
- **InfluxDB**: 1Gi memory, 500m CPU
- **Redis**: 256Mi memory, 200m CPU

### Database Optimization
- Connection pooling
- Query optimization
- Index tuning
- Regular VACUUM operations

### Application Optimization
- Caching strategies
- Async processing
- Connection pooling
- Resource limits

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 8080, 3000, 5432, 6379, 8086, 9090 are available
2. **Memory issues**: Increase Docker memory allocation
3. **Permission issues**: Check file permissions on mounted volumes
4. **Network issues**: Verify Docker network configuration

### Debugging Commands

```bash
# Check container status
docker-compose ps

# View detailed logs
docker-compose logs -f [service]

# Execute commands in container
docker-compose exec [service] [command]

# Check network connectivity
docker-compose exec ssts curl -f http://postgres:5432
```

### Health Checks

The health check script provides comprehensive status:
```bash
./scripts/health-check.sh
```

## CI/CD Integration

### GitHub Actions

The included CI/CD pipeline provides:
- Automated testing on pull requests
- Security scanning
- Multi-stage builds
- Environment-specific deployments

### Pipeline Stages
1. **Test**: Unit tests, integration tests
2. **Lint**: Code quality checks
3. **Security**: Vulnerability scanning
4. **Build**: Docker image creation
5. **Deploy**: Environment-specific deployment

## Monitoring and Alerting

### Metrics
- Application metrics via Prometheus
- System metrics via Node Exporter
- Database metrics via specific exporters
- Custom business metrics

### Dashboards
- Grafana dashboards for visualization
- Real-time monitoring
- Historical data analysis
- Alert visualization

### Alerting Rules
- High CPU/memory usage
- Service downtime
- Database connection issues
- High error rates
- Disk space warnings

## Support

For issues and questions:
1. Check the troubleshooting guide
2. Review logs for error messages
3. Verify configuration settings
4. Check resource availability
5. Contact the development team