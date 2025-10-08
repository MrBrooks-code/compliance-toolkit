# Docker Quick Start Guide

## 🚀 From Project Root

```bash
# Start the server
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Stop the server
docker-compose -f docker/docker-compose.yml down
```

## 📁 From Docker Directory

```bash
cd docker

# Start the server
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the server
docker-compose down
```

## 🔄 Rebuild After Code Changes

```bash
# From project root
docker-compose -f docker/docker-compose.yml down
docker-compose -f docker/docker-compose.yml build
docker-compose -f docker/docker-compose.yml up -d

# Or from docker directory
cd docker
docker-compose down
docker-compose build
docker-compose up -d
```

## 🌐 Access

- **Server**: http://localhost:8080
- **Dashboard**: http://localhost:8080/dashboard

## 📊 Useful Commands

```bash
# Check container status
docker-compose -f docker/docker-compose.yml ps

# View last 50 log lines
docker-compose -f docker/docker-compose.yml logs --tail=50

# Follow logs in real-time
docker-compose -f docker/docker-compose.yml logs -f

# Restart container
docker-compose -f docker/docker-compose.yml restart

# Rebuild without cache
docker-compose -f docker/docker-compose.yml build --no-cache
```

## ⚙️ Configuration

Environment variables can be set in `docker/.env` file:

```bash
# Copy example
cp docker/.env.example docker/.env

# Edit configuration
nano docker/.env
```

Key variables:
- `SERVER_PORT` - Server port (default: 8080)
- `DB_PATH` - Database file path (default: data/compliance.db)
- `AUTH_ENABLED` - Enable authentication (default: true)
- `DASHBOARD_ENABLED` - Enable web dashboard (default: true)
- `LOGGING_LEVEL` - Log level: debug, info, warn, error (default: info)

## 🐛 Troubleshooting

**Container won't start:**
```bash
docker-compose -f docker/docker-compose.yml logs
```

**Port already in use:**
```bash
# Change port in docker/.env
echo "HOST_PORT=9080" >> docker/.env
docker-compose -f docker/docker-compose.yml up -d
```

**Database issues:**
```bash
# Stop and remove volumes
docker-compose -f docker/docker-compose.yml down -v
# Start fresh
docker-compose -f docker/docker-compose.yml up -d
```
