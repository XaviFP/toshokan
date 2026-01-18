# Production Deployment Guide

**No local dependencies required!** Everything is built inside Docker containers.

## Prerequisites

- Docker and Docker Compose installed
- Domain names configured (DNS A records pointing to server IP)
- Ports 80 and 443 open on firewall

## Quick Start

### 1. Create .env.prod from template

```bash
cp .env.prod.example .env.prod
```

### 2. Edit .env.prod

Generate secrets and fill in your values:

```bash
# Generate passwords/secrets
openssl rand -base64 32  # For DB_PASSWORD, ADMIN_HEADER_SECRET
openssl rand -base64 24  # For GRAFANA_ADMIN_PASSWORD

# Generate JWT keypair
openssl genpkey -algorithm ed25519 -out private.pem
openssl pkey -in private.pem -pubout -out public.pem
cat private.pem   # Copy to JWT_PRIVATE_KEY
cat public.pem    # Copy to JWT_PUBLIC_KEY
rm private.pem public.pem
```

### 3. Deploy

```bash
# Build and start the database first
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d db cache

# Wait for database to be healthy
docker compose --env-file .env.prod -f docker-compose.prod.yaml ps

# Run migrations
docker build -f migrations.Dockerfile -t toshokan-migrations .
docker run --rm --network toshokan_internal \
  -e DB_HOST=db \
  -e DB_PORT=5432 \
  -e DB_USER=${DB_USER} \
  -e DB_PASSWORD=${DB_PASSWORD} \
  -e SERVICE=all \
  toshokan-migrations

# Start all services
docker compose --env-file .env.prod -f docker-compose.prod.yaml up -d
```

Or use the Makefile shortcut:

```bash
make prod
```

### 4. Verify Deployment

```bash
# Check all services are healthy
docker compose --env-file .env.prod -f docker-compose.prod.yaml ps

# Check logs
docker compose --env-file .env.prod -f docker-compose.prod.yaml logs -f gate
```
