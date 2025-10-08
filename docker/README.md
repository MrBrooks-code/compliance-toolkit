# Compliance Server Docker Deployment

This directory contains Docker configuration files for deploying the Compliance Server in a containerized environment.

## 📋 Quick Start

### Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+

### Starting from Project Root

```bash
# From D:\golang-labs\ComplianceToolkit (or wherever your project is)

# Build and start the server
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Stop the server
docker-compose -f docker/docker-compose.yml down
```

The server will be available at `http://localhost:8080`

### Starting from Docker Directory

```bash
# Navigate to docker directory
cd docker

# Build and start the server
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the server
docker-compose down
```

The server will be available at `http://localhost:8080`

### How It Works

The deployment uses a **multi-stage Docker build**:
1. **Builder stage**: Compiles the Go binary from source in a Linux environment
2. **Runtime stage**: Creates a minimal Alpine Linux image with only the binary and dependencies

This approach works perfectly on Windows because Docker handles the Linux cross-compilation automatically.


## 🔧 Configuration

### Environment Variables

All configuration is done through environment variables in the `.env` file:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | `8080` | Internal server port |
| `HOST_PORT` | `8080` | External host port |
| `TLS_ENABLED` | `false` | Enable HTTPS |
| `TLS_CERT_FILE` | `certs/server.crt` | TLS certificate path |
| `TLS_KEY_FILE` | `certs/server.key` | TLS private key path |
| `DB_TYPE` | `sqlite` | Database type |
| `DB_PATH` | `data/compliance.db` | SQLite database path |
| `AUTH_ENABLED` | `true` | Enable authentication |
| `AUTH_REQUIRE_KEY` | `false` | Require API keys |
| `USE_HASHED_KEYS` | `false` | Use bcrypt-hashed API keys |
| `DASHBOARD_ENABLED` | `true` | Enable web dashboard |
| `DASHBOARD_PATH` | `/dashboard` | Dashboard URL path |
| `DASHBOARD_LOGIN_MESSAGE` | `Welcome...` | Login page message |
| `LOGGING_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `LOGGING_FORMAT` | `json` | Log format (json/text) |
| `LOGGING_OUTPUT` | `stdout` | Log output destination |
| `TZ` | `UTC` | Container timezone |

### Volume Mounts

The compose file mounts several directories for data persistence:

- `./data` → `/app/data` - Database storage
- `./certs` → `/app/certs` - TLS certificates (read-only)
- `./logs` → `/app/logs` - Log files
- `./server.yaml` → `/app/server.yaml` - Custom config (optional)

## 🚀 Advanced Usage

### Enable HTTPS

1. Generate TLS certificates:

```bash
# Create certs directory
mkdir -p certs

# Generate self-signed certificate (for testing)
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"

# Update .env
echo "TLS_ENABLED=true" >> .env
echo "HOST_TLS_PORT=8443" >> .env
```

2. Restart the server:

```bash
docker-compose down
docker-compose up -d
```

Access via `https://localhost:8443`

### Use Custom Configuration

Instead of environment variables, you can mount a custom `server.yaml`:

1. Create `server.yaml` in the docker directory

2. Ensure the compose file has this volume mount:
   ```yaml
   - ./server.yaml:/app/server.yaml:ro
   ```

3. Restart: `docker-compose up -d`

### Enable API Key Authentication

```bash
# Update .env
echo "AUTH_REQUIRE_KEY=true" >> .env
echo "USE_HASHED_KEYS=true" >> .env

# Restart
docker-compose restart
```

Then generate API keys through the web UI at Settings page.

### View Logs

```bash
# Follow logs in real-time
docker-compose logs -f

# View last 100 lines
docker-compose logs --tail=100

# View specific service
docker-compose logs compliance-server
```

### Database Backup

```bash
# Backup SQLite database
docker cp compliance-server:/app/data/compliance.db ./backup-$(date +%Y%m%d).db

# Restore database
docker cp ./backup.db compliance-server:/app/data/compliance.db
docker-compose restart
```

### Resource Limits

Edit `docker-compose.yml` to adjust resource limits:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 1G
```

## 🔍 Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs

# Check container status
docker-compose ps

# Rebuild image
docker-compose build --no-cache
docker-compose up -d
```

### Port Already in Use

```bash
# Change host port in .env
echo "HOST_PORT=9080" >> .env
docker-compose up -d
```

### Permission Denied Errors

The container runs as non-root user (UID 1000). Ensure host directories have correct permissions:

```bash
chmod -R 755 data logs
```

### Database Locked

```bash
# Stop all clients
# Restart server
docker-compose restart
```

## 📊 Health Checks

The container includes a health check that runs every 30 seconds:

```bash
# Check health status
docker inspect compliance-server | grep -A5 Health

# Manual health check
docker exec compliance-server wget -qO- http://localhost:8080/ | jq
```

## 🔐 Security Best Practices

1. **Enable TLS** - Always use HTTPS in production
2. **Use Hashed API Keys** - Set `USE_HASHED_KEYS=true`
3. **Restrict Network Access** - Use firewall rules or docker networks
4. **Regular Backups** - Backup database and certificates
5. **Update Images** - Rebuild regularly for security patches
6. **Change Default Credentials** - Create admin account and change password
7. **Limit Resources** - Set CPU/memory limits in production

## 🐳 Docker Commands Reference

```bash
# Build image
docker-compose build

# Start in detached mode
docker-compose up -d

# Start in foreground (see logs)
docker-compose up

# Stop services
docker-compose stop

# Stop and remove containers
docker-compose down

# Remove all data (CAUTION)
docker-compose down -v

# Rebuild and restart
docker-compose up -d --build

# Scale services (if needed)
docker-compose up -d --scale compliance-server=3

# Execute command in container
docker exec -it compliance-server sh

# View resource usage
docker stats compliance-server
```

## 📝 Example Deployments

### Development

```bash
cp .env.example .env
# Use defaults
docker-compose up
```

### Production with HTTPS

```bash
cp .env.example .env
# Edit .env:
# - TLS_ENABLED=true
# - AUTH_REQUIRE_KEY=true
# - USE_HASHED_KEYS=true
# - LOGGING_FORMAT=json
docker-compose up -d
```

### Behind Reverse Proxy (Nginx/Traefik)

```bash
# .env
TLS_ENABLED=false  # Proxy handles TLS
HOST_PORT=8080
# Add to compose: labels for proxy auto-discovery
```

## 🛠️ Manual Docker Build

If you need to build the Docker image manually without docker-compose:

### From Project Root

```bash
# Build the image
docker build -f docker/Dockerfile.multistage -t compliance-server:latest .

# Run manually
docker run -d \
  -p 8080:8080 \
  -v ./docker/data:/app/data \
  -e SERVER_PORT=8080 \
  --name compliance-server \
  compliance-server:latest

# View logs
docker logs -f compliance-server

# Stop container
docker stop compliance-server
docker rm compliance-server
```

### Rebuilding After Code Changes

```bash
# From project root
docker-compose -f docker/docker-compose.yml down
docker-compose -f docker/docker-compose.yml build --no-cache
docker-compose -f docker/docker-compose.yml up -d
```

Or from docker directory:

```bash
cd docker
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## 📚 Additional Resources

- [Main Documentation](../docs/README.md)
- [API Documentation](../docs/API.md)
- [Security Guide](../docs/security/API_KEY_SECURITY.md)
- [Docker Hub](https://hub.docker.com) (if published)

## 🐛 Support

For issues or questions:
1. Check logs: `docker-compose logs`
2. Review this README
3. Check main project documentation
4. File an issue on GitHub
