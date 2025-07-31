# SSTS Troubleshooting Guide

## Quick Diagnosis

### Health Check Script
Always start with the automated health check:
```bash
./scripts/health-check.sh
```

### Service Status
Check if all services are running:
```bash
# Docker
docker-compose ps

# Kubernetes
kubectl get pods -n ssts-[env]
```

## Common Issues and Solutions

### 1. Connection Refused (localhost:8080)

**Symptoms:**
- Browser shows "connection refused" 
- `curl http://localhost:8080` fails
- Health check fails for SSTS application

**Diagnosis:**
```bash
# Check if container is running
docker-compose ps ssts

# Check container logs
docker-compose logs ssts

# Check port binding
netstat -tlnp | grep 8080
```

**Common Causes & Solutions:**

#### Container Not Started
```bash
# Start the container
docker-compose up -d ssts

# If it fails to start, check logs
docker-compose logs ssts
```

#### Port Already in Use
```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process or change port in docker-compose.yml
```

#### Configuration Issues
```bash
# Check configuration file
cat ssts.yaml

# Verify environment variables
docker-compose exec ssts env | grep SSTS_
```

#### Build Issues
```bash
# Rebuild the image
docker-compose build --no-cache ssts
docker-compose up -d ssts
```

### 2. Database Connection Issues

**Symptoms:**
- "connection refused" to database
- "too many connections" errors
- Database timeout errors

**Diagnosis:**
```bash
# Check PostgreSQL status
docker-compose exec postgres pg_isready -U ssts

# Check connection from app container
docker-compose exec ssts telnet postgres 5432

# Check connection pool settings
docker-compose logs ssts | grep -i connection
```

**Solutions:**

#### PostgreSQL Not Ready
```bash
# Wait for PostgreSQL to fully start (can take 30-60 seconds)
# Check initialization logs
docker-compose logs postgres

# Restart if stuck
docker-compose restart postgres
```

#### Connection Pool Exhausted
```bash
# Check active connections
docker-compose exec postgres psql -U ssts -c "SELECT count(*) FROM pg_stat_activity;"

# Restart application to reset pool
docker-compose restart ssts
```

#### Network Issues
```bash
# Verify containers can communicate
docker-compose exec ssts ping postgres

# Check network configuration
docker network ls
docker network inspect ssts_ssts-network
```

### 3. Build Failures

**Symptoms:**
- Docker build fails
- Go compilation errors
- Web build failures

**Common Build Issues:**

#### Go Dependencies
```bash
# Go module issues
ERROR: failed to solve: failed to compute cache key

# Solution: Clean and rebuild
rm -f go.sum
docker-compose build --no-cache ssts
```

#### Web Build Issues
```bash
# NPM/Node issues in web container
# Check web directory structure
ls -la web/

# Rebuild web dependencies
cd web && npm ci && npm run build
```

#### Docker Layer Caching
```bash
# Clear Docker build cache
docker system prune -af
docker-compose build --no-cache
```

### 4. Permission Issues

**Symptoms:**
- "permission denied" errors
- Container fails to write to volumes
- Configuration files not readable

**Solutions:**

#### Volume Mount Permissions
```bash
# Check directory ownership
ls -la data/ logs/

# Fix permissions
chmod -R 755 data logs
chown -R $USER:$USER data logs
```

#### Container User Issues
```bash
# Check container user
docker-compose exec ssts id

# If running as root when shouldn't be, check Dockerfile
```

### 5. Memory Issues

**Symptoms:**
- Container killed/restarted frequently
- Out of memory errors
- Slow performance

**Diagnosis:**
```bash
# Check container memory usage
docker stats

# Check system memory
free -h

# Check container limits
docker-compose exec ssts cat /sys/fs/cgroup/memory/memory.limit_in_bytes
```

**Solutions:**

#### Increase Docker Memory
```bash
# Update docker-compose.yml
services:
  ssts:
    deploy:
      resources:
        limits:
          memory: 1G
```

#### Memory Leaks
```bash
# Monitor memory usage over time
watch -n 5 'docker stats --no-stream'

# Check for memory leaks in application
docker-compose exec ssts ps aux
```

### 6. Network Connectivity Issues

**Symptoms:**
- Services can't communicate
- External API calls fail
- WebSocket connections drop

**Diagnosis:**
```bash
# Test container-to-container connectivity
docker-compose exec ssts ping postgres
docker-compose exec ssts ping influxdb

# Check network configuration
docker network inspect ssts_ssts-network

# Test external connectivity
docker-compose exec ssts curl -I https://google.com
```

**Solutions:**

#### Network Recreation
```bash
# Recreate network
docker-compose down
docker network prune -f
docker-compose up -d
```

#### DNS Issues
```bash
# Check DNS resolution
docker-compose exec ssts nslookup postgres
docker-compose exec ssts cat /etc/resolv.conf
```

### 7. Web UI Issues

**Symptoms:**
- White screen/blank page
- JavaScript errors
- Assets not loading

**Diagnosis:**
```bash
# Check web build
ls -la web/build/

# Check nginx logs
docker-compose logs nginx

# Check browser developer console
```

**Solutions:**

#### Rebuild Web Assets
```bash
# Clean and rebuild web
rm -rf web/build/ web/node_modules/
cd web && npm ci && npm run build

# Rebuild container
docker-compose build --no-cache ssts
```

#### Static Asset Issues
```bash
# Check nginx configuration
docker-compose exec nginx nginx -t

# Check asset paths
curl -I http://localhost:8080/static/js/main.js
```

### 8. Performance Issues

**Symptoms:**
- Slow response times
- High CPU/memory usage
- Timeouts

**Diagnosis:**
```bash
# Check resource usage
docker stats

# Check application metrics
curl http://localhost:8080/metrics

# Check database performance
docker-compose exec postgres psql -U ssts -c "SELECT * FROM pg_stat_activity;"
```

**Solutions:**

#### Resource Optimization
```bash
# Scale up resources
# Edit docker-compose.yml resource limits

# Scale horizontally (Kubernetes)
kubectl scale deployment/ssts-app --replicas=3
```

#### Database Optimization
```bash
# Analyze database performance
docker-compose exec postgres psql -U ssts -c "ANALYZE;"

# Check slow queries
docker-compose exec postgres psql -U ssts -c "SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

### 9. SSL/TLS Issues

**Symptoms:**
- Certificate errors
- HTTPS not working
- Mixed content warnings

**Solutions:**

#### Certificate Problems
```bash
# Check certificate validity
openssl x509 -in certificate.crt -text -noout

# Generate new self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt
```

#### Mixed Content
```bash
# Ensure all resources use HTTPS
# Check nginx configuration for proper redirects
```

### 10. Monitoring/Metrics Issues

**Symptoms:**
- Grafana shows no data
- Prometheus targets down
- Missing metrics

**Diagnosis:**
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check InfluxDB connectivity
curl http://localhost:8086/health

# Check metrics endpoint
curl http://localhost:8080/metrics
```

**Solutions:**

#### Service Discovery Issues
```bash
# Restart monitoring stack
docker-compose restart prometheus grafana

# Check configuration
docker-compose exec prometheus cat /etc/prometheus/prometheus.yml
```

#### Data Sources
```bash
# Verify Grafana data source configuration
# Access Grafana UI and check data source settings
```

## Advanced Troubleshooting

### Container Debugging

#### Interactive Shell Access
```bash
# Docker
docker-compose exec ssts /bin/sh

# Kubernetes
kubectl exec -it deployment/ssts-app -n ssts-prod -- /bin/sh
```

#### Debug Information
```bash
# System information
uname -a
cat /etc/os-release

# Process information
ps aux
top

# Network information
netstat -tlnp
ss -tlnp

# Disk usage
df -h
du -sh /*

# Memory information
free -h
cat /proc/meminfo
```

### Log Analysis

#### Log Aggregation
```bash
# Combine logs from all services
docker-compose logs --follow > all_logs.txt

# Search for specific errors
grep -i "error\|exception\|fail" all_logs.txt

# Analyze patterns
awk '{print $1, $2, $3}' all_logs.txt | sort | uniq -c | sort -nr
```

#### Structured Log Analysis
```bash
# Parse JSON logs
docker-compose logs ssts | jq -r 'select(.level == "error")'

# Time-based filtering
docker-compose logs --since="2h" ssts
```

### Performance Profiling

#### Application Profiling
```bash
# Enable pprof (if available)
curl http://localhost:8080/debug/pprof/profile > cpu.prof

# Memory profiling
curl http://localhost:8080/debug/pprof/heap > mem.prof

# Goroutine analysis
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

#### Database Profiling
```bash
# PostgreSQL slow query log
docker-compose exec postgres psql -U ssts -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"
docker-compose restart postgres

# Monitor active queries
watch -n 5 'docker-compose exec postgres psql -U ssts -c "SELECT query, state, query_start FROM pg_stat_activity WHERE state != '\''idle'\'';"'
```

## Emergency Procedures

### Complete System Recovery

#### Backup Recovery
```bash
# Stop all services
docker-compose down

# Restore from backup
./scripts/deploy.sh restore

# Start services
docker-compose up -d

# Verify recovery
./scripts/health-check.sh
```

#### Factory Reset
```bash
# WARNING: This will delete all data
docker-compose down -v
docker system prune -af --volumes
rm -rf data/ logs/

# Redeploy
docker-compose up -d
```

### Disaster Recovery

#### Data Recovery
```bash
# If volumes are corrupted, restore from backup
docker volume rm $(docker volume ls -q)
# Restore volume data from backup location
```

#### Service Migration
```bash
# Export data
docker-compose exec postgres pg_dumpall -U ssts > full_backup.sql

# Import to new system
docker-compose exec -T postgres psql -U ssts < full_backup.sql
```

## Getting Help

### Information to Collect

When seeking help, provide:

1. **Environment Information:**
   ```bash
   docker version
   docker-compose version
   uname -a
   ```

2. **Service Status:**
   ```bash
   docker-compose ps
   ./scripts/health-check.sh
   ```

3. **Recent Logs:**
   ```bash
   docker-compose logs --tail=100 > logs.txt
   ```

4. **Configuration:**
   ```bash
   cat docker-compose.yml
   cat ssts.yaml
   ```

5. **Resource Usage:**
   ```bash
   docker stats --no-stream
   df -h
   free -h
   ```

### Support Channels

1. Check documentation first
2. Search existing issues
3. Create detailed issue report
4. Contact development team
5. Emergency escalation (for critical issues)

### Debug Mode

Enable debug logging for more detailed information:

#### Docker
```bash
# Set debug environment variable
SSTS_LOG_LEVEL=debug docker-compose up -d ssts
```

#### Kubernetes
```bash
# Update configmap or deployment
kubectl patch configmap ssts-config -n ssts-prod -p '{"data":{"SSTS_LOG_LEVEL":"debug"}}'
kubectl rollout restart deployment/ssts-app -n ssts-prod
```

Remember: Always test solutions in a development environment before applying to production!