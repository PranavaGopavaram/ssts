# SSTS Operations Runbook

## Service Management

### Starting Services

#### Docker
```bash
# Start all services
./scripts/deploy.sh up

# Start specific service
docker-compose up -d [service-name]
```

#### Kubernetes
```bash
# Deploy to environment
./scripts/deploy.sh -e [env] -m k8s up

# Scale deployment
kubectl scale deployment/ssts-app --replicas=3 -n ssts-prod
```

### Stopping Services

#### Docker
```bash
# Stop all services
./scripts/deploy.sh down

# Stop specific service
docker-compose stop [service-name]
```

#### Kubernetes
```bash
# Remove deployment
./scripts/deploy.sh -e [env] -m k8s down

# Scale down
kubectl scale deployment/ssts-app --replicas=0 -n ssts-prod
```

## Monitoring and Alerting

### Health Checks

#### Automated Health Check
```bash
./scripts/health-check.sh
```

#### Manual Health Checks

**SSTS Application:**
```bash
curl -f http://localhost:8080/health
```

**PostgreSQL:**
```bash
docker-compose exec postgres pg_isready -U ssts
# or
kubectl exec -it statefulset/postgres -n ssts-prod -- pg_isready -U ssts
```

**Redis:**
```bash
docker-compose exec redis redis-cli ping
# or
kubectl exec -it deployment/redis -n ssts-prod -- redis-cli ping
```

**InfluxDB:**
```bash
curl -f http://localhost:8086/health
# or
kubectl port-forward service/influxdb-service 8086:8086 -n ssts-prod
```

### Monitoring Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **InfluxDB UI**: http://localhost:8086

### Key Metrics to Monitor

1. **Application Metrics:**
   - Response time (95th percentile < 1s)
   - Error rate (< 1%)
   - Active connections
   - Test execution count
   - Test failure rate

2. **System Metrics:**
   - CPU usage (< 80%)
   - Memory usage (< 85%)
   - Disk usage (< 90%)
   - Network I/O

3. **Database Metrics:**
   - Connection pool usage
   - Query execution time
   - Lock wait time
   - Cache hit ratio

## Backup and Recovery

### Database Backup

#### Automated Backup
```bash
./scripts/deploy.sh backup
```

#### Manual Backup

**PostgreSQL:**
```bash
# Docker
docker-compose exec postgres pg_dump -U ssts ssts > backup_$(date +%Y%m%d).sql

# Kubernetes
kubectl exec -it statefulset/postgres -n ssts-prod -- pg_dump -U ssts ssts > backup_$(date +%Y%m%d).sql
```

**InfluxDB:**
```bash
# Docker
docker-compose exec influxdb influx backup /tmp/backup
docker-compose cp influxdb:/tmp/backup ./influxdb_backup_$(date +%Y%m%d)

# Kubernetes
kubectl exec -it statefulset/influxdb -n ssts-prod -- influx backup /tmp/backup
kubectl cp ssts-prod/influxdb-0:/tmp/backup ./influxdb_backup_$(date +%Y%m%d)
```

### Database Recovery

#### Automated Recovery
```bash
./scripts/deploy.sh restore
```

#### Manual Recovery

**PostgreSQL:**
```bash
# Docker
docker-compose exec -i postgres psql -U ssts ssts < backup_file.sql

# Kubernetes
kubectl exec -i statefulset/postgres -n ssts-prod -- psql -U ssts ssts < backup_file.sql
```

## Log Management

### Viewing Logs

#### Docker
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f ssts

# Last N lines
docker-compose logs --tail=100 ssts
```

#### Kubernetes
```bash
# Application logs
kubectl logs -f deployment/ssts-app -n ssts-prod

# Previous container logs
kubectl logs deployment/ssts-app --previous -n ssts-prod

# All pods with label
kubectl logs -f -l app=ssts-app -n ssts-prod
```

### Log Locations

#### Docker
- Application logs: `./logs/`
- Nginx logs: `./logs/nginx/`
- Container logs: `docker-compose logs`

#### Kubernetes
- Pod logs: `kubectl logs`
- Node logs: `/var/log/pods/`
- System logs: `journalctl`

### Log Rotation

Configure log rotation to prevent disk space issues:

```bash
# Add to /etc/logrotate.d/ssts
/path/to/ssts/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 root root
}
```

## Performance Optimization

### Resource Scaling

#### Docker
```bash
# Update docker-compose.yml resource limits
services:
  ssts:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

#### Kubernetes
```bash
# Scale replicas
kubectl scale deployment/ssts-app --replicas=5 -n ssts-prod

# Update resource limits
kubectl patch deployment ssts-app -n ssts-prod -p '{"spec":{"template":{"spec":{"containers":[{"name":"ssts","resources":{"limits":{"cpu":"1000m","memory":"1Gi"}}}]}}}}'
```

### Database Optimization

#### PostgreSQL
```sql
-- Check slow queries
SELECT query, mean_time, calls, total_time 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Analyze table statistics
ANALYZE;

-- Vacuum tables
VACUUM ANALYZE;
```

#### InfluxDB
```bash
# Check retention policies
influx -execute "SHOW RETENTION POLICIES"

# Compact shards
influx -execute "COMPACT SHARDS"
```

## Security Operations

### Certificate Management

#### Generate TLS certificates
```bash
# Self-signed for development
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt

# Update configuration
# Docker: Mount certificates and update nginx config
# Kubernetes: Create TLS secret
kubectl create secret tls ssts-tls --cert=tls.crt --key=tls.key -n ssts-prod
```

### Security Scanning

#### Container Scanning
```bash
# Scan with Trivy
trivy image ssts:latest

# Scan filesystem
trivy fs .
```

#### Dependency Scanning
```bash
# Go dependencies
go list -m -f '{{if not (or .Main .Indirect)}}{{.Path}} {{.Version}}{{end}}' all

# NPM dependencies
npm audit
```

## Incident Response

### Incident Classification

1. **P0 - Critical**: Complete service outage
2. **P1 - High**: Major functionality impacted
3. **P2 - Medium**: Minor functionality impacted  
4. **P3 - Low**: Cosmetic or minor issues

### Incident Response Steps

1. **Detection**: Monitor alerts, user reports
2. **Assessment**: Determine impact and severity
3. **Response**: Immediate mitigation actions
4. **Investigation**: Root cause analysis
5. **Resolution**: Permanent fix implementation
6. **Documentation**: Post-incident review

### Common Incident Scenarios

#### Service Completely Down
```bash
# Check service status
./scripts/health-check.sh

# Check container/pod status
docker-compose ps
kubectl get pods -n ssts-prod

# Check logs for errors
docker-compose logs ssts
kubectl logs deployment/ssts-app -n ssts-prod

# Restart services
./scripts/deploy.sh restart
```

#### High CPU/Memory Usage
```bash
# Check resource usage
docker stats
kubectl top pods -n ssts-prod

# Check for memory leaks
docker-compose exec ssts ps aux
kubectl exec -it deployment/ssts-app -n ssts-prod -- ps aux

# Scale if needed
kubectl scale deployment/ssts-app --replicas=3 -n ssts-prod
```

#### Database Connection Issues
```bash
# Check database connectivity
docker-compose exec ssts telnet postgres 5432
kubectl exec -it deployment/ssts-app -n ssts-prod -- telnet postgres-service 5432

# Check connection pool
# Look for "too many connections" errors in logs

# Restart database if needed
docker-compose restart postgres
kubectl rollout restart statefulset/postgres -n ssts-prod
```

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily
- Check service health
- Review monitoring dashboards
- Check disk space
- Review error logs

#### Weekly
- Database backup verification
- Security updates
- Performance review
- Log rotation cleanup

#### Monthly
- Full system backup
- Capacity planning review
- Security audit
- Documentation updates

### Maintenance Windows

Plan maintenance during low-usage periods:

1. **Notification**: Inform users 24-48 hours in advance
2. **Preparation**: Test procedures in staging
3. **Execution**: Follow maintenance checklist
4. **Verification**: Confirm all services working
5. **Rollback**: Have rollback plan ready

### Update Procedures

#### Application Updates
```bash
# Docker
docker-compose build --no-cache
docker-compose up -d

# Kubernetes
kubectl set image deployment/ssts-app ssts=ssts:new-version -n ssts-prod
kubectl rollout status deployment/ssts-app -n ssts-prod
```

#### Database Updates
```bash
# Backup before update
./scripts/deploy.sh backup

# Update database
docker-compose exec ssts ./ssts migrate

# Verify update
./scripts/health-check.sh
```

## Contact Information

### Escalation Matrix

| Severity | First Contact | Escalation |
|----------|---------------|------------|
| P0/P1 | On-call Engineer | Team Lead |
| P2 | Team Member | Team Lead |
| P3 | Team Member | Next Business Day |

### Key Contacts

- **On-call**: [on-call-phone]
- **Team Lead**: [team-lead-email]
- **DevOps**: [devops-email]
- **Security**: [security-email]

## Useful Commands Reference

### Docker
```bash
# View resource usage
docker stats

# Clean up unused resources
docker system prune -f

# Update and restart specific service
docker-compose up -d --force-recreate ssts

# Execute command in container
docker-compose exec ssts /bin/sh
```

### Kubernetes
```bash
# Get cluster info
kubectl cluster-info

# Describe resource
kubectl describe pod [pod-name] -n ssts-prod

# Port forward for debugging
kubectl port-forward service/ssts-service 8080:8080 -n ssts-prod

# Get events
kubectl get events --sort-by=.metadata.creationTimestamp -n ssts-prod
```