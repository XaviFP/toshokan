# Production Deployment Guide

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
docker compose -f docker-compose.prod.yaml up -d
```
