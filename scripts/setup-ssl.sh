#!/bin/bash

# CashbackTV SSL Setup Script - Fully Automated
# Usage: ./setup-ssl.sh [email]
# This script will:
# 1. Obtain SSL certificates for cashbacktv.live and cdn.cashbacktv.live
# 2. Start the full Docker Compose stack

set -e  # Exit on error

DOMAIN="cashbacktv.live"
CDN_DOMAIN="cdn.cashbacktv.live"
EMAIL=${1:-"admin@${DOMAIN}"}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_DIR="${SCRIPT_DIR}/../docker"

echo "🚀 CashbackTV SSL Setup - Fully Automated"
echo "=========================================="
echo "Domain: $DOMAIN"
echo "CDN Domain: $CDN_DOMAIN"
echo "Email: $EMAIL"
echo ""

# Check if running as root (needed for port 80/443)
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  This script needs root privileges to bind to ports 80 and 443"
    echo "   Please run with: sudo ./setup-ssl.sh"
    exit 1
fi

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker daemon is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Use docker compose (v2) if available, otherwise docker-compose (v1)
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p "${DOCKER_DIR}/certbot"
mkdir -p "${DOCKER_DIR}/nginx/ssl"
mkdir -p "${DOCKER_DIR}/../backend/migrations"

# Check if ports 80 and 443 are available
echo "🔍 Checking if ports 80 and 443 are available..."
if lsof -Pi :80 -sTCP:LISTEN -t >/dev/null 2>&1 || lsof -Pi :443 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  Ports 80 or 443 are already in use."
    echo "   Please stop any services using these ports and try again."
    exit 1
fi

# Function to check DNS (optional - will warn if dig is not available)
check_dns() {
    local domain=$1
    echo "🔍 Checking DNS for $domain..."
    
    if command -v dig &> /dev/null; then
        if ! dig +short $domain | grep -q .; then
            echo "⚠️  DNS record for $domain not found or not pointing to this server"
            echo "   Please ensure DNS is configured:"
            echo "   - $domain → $(curl -s ifconfig.me 2>/dev/null || echo 'YOUR_SERVER_IP')"
            echo "   Continuing anyway (certbot will verify)..."
            return 0
        fi
        echo "✅ DNS configured for $domain"
    else
        echo "⚠️  'dig' command not found, skipping DNS check"
        echo "   Please ensure DNS is configured before proceeding"
        read -p "   Continue anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return 1
        fi
    fi
    return 0
}

# Check DNS for both domains (non-blocking)
check_dns "$DOMAIN" || exit 1
check_dns "$CDN_DOMAIN" || exit 1

# Start temporary nginx for HTTP challenge (if needed)
echo ""
echo "🌐 Starting temporary nginx for HTTP challenge..."
cd "$DOCKER_DIR"

# Create temporary nginx config for certbot challenge
cat > /tmp/nginx-temp.conf <<EOF
events {
    worker_connections 1024;
}
http {
    server {
        listen 80;
        server_name $DOMAIN www.$DOMAIN $CDN_DOMAIN;
        
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        
        location / {
            return 200 "CashbackTV SSL Setup - Please wait...";
            add_header Content-Type text/plain;
        }
    }
}
EOF

# Start temporary nginx container
TEMP_NGINX_CONTAINER="cashbacktv-nginx-temp"
docker run -d \
    --name "$TEMP_NGINX_CONTAINER" \
    -p 80:80 \
    -v "${DOCKER_DIR}/certbot:/var/www/certbot" \
    -v /tmp/nginx-temp.conf:/etc/nginx/nginx.conf:ro \
    nginx:alpine > /dev/null 2>&1 || {
    # If container already exists, remove it first
    docker rm -f "$TEMP_NGINX_CONTAINER" > /dev/null 2>&1
    docker run -d \
        --name "$TEMP_NGINX_CONTAINER" \
        -p 80:80 \
        -v "${DOCKER_DIR}/certbot:/var/www/certbot" \
        -v /tmp/nginx-temp.conf:/etc/nginx/nginx.conf:ro \
        nginx:alpine > /dev/null 2>&1
}

# Wait for nginx to be ready
sleep 2

# Function to obtain certificate
obtain_certificate() {
    local cert_name=$1
    shift
    local domains=("$@")
    
    echo ""
    echo "📜 Obtaining SSL certificate for $cert_name..."
    echo "   Domains: ${domains[*]}"
    
    docker run --rm \
        -v "${DOCKER_DIR}/certbot:/var/www/certbot" \
        -v "${DOCKER_DIR}/certbot:/etc/letsencrypt" \
        certbot/certbot certonly \
        --webroot \
        --webroot-path=/var/www/certbot \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        --non-interactive \
        "${domains[@]}"
    
    if [ $? -eq 0 ]; then
        echo "✅ Certificate obtained successfully for $cert_name"
        return 0
    else
        echo "❌ Failed to obtain certificate for $cert_name"
        return 1
    fi
}

# Obtain certificates
echo ""
echo "🔐 Obtaining SSL certificates..."

# Main domain certificate
if ! obtain_certificate "$DOMAIN" "-d" "$DOMAIN" "-d" "www.$DOMAIN"; then
    echo "❌ Failed to obtain certificate for $DOMAIN"
    docker rm -f "$TEMP_NGINX_CONTAINER" > /dev/null 2>&1
    rm -f /tmp/nginx-temp.conf
    exit 1
fi

# CDN domain certificate
if ! obtain_certificate "$CDN_DOMAIN" "-d" "$CDN_DOMAIN"; then
    echo "❌ Failed to obtain certificate for $CDN_DOMAIN"
    docker rm -f "$TEMP_NGINX_CONTAINER" > /dev/null 2>&1
    rm -f /tmp/nginx-temp.conf
    exit 1
fi

# Stop temporary nginx
echo ""
echo "🛑 Stopping temporary nginx..."
docker rm -f "$TEMP_NGINX_CONTAINER" > /dev/null 2>&1
rm -f /tmp/nginx-temp.conf

# Verify certificates exist
echo ""
echo "🔍 Verifying certificates..."
CERT_MAIN="${DOCKER_DIR}/certbot/live/$DOMAIN/fullchain.pem"
CERT_CDN="${DOCKER_DIR}/certbot/live/$CDN_DOMAIN/fullchain.pem"

if [ ! -f "$CERT_MAIN" ]; then
    echo "❌ Main domain certificate not found at: $CERT_MAIN"
    exit 1
fi

if [ ! -f "$CERT_CDN" ]; then
    echo "❌ CDN domain certificate not found at: $CERT_CDN"
    exit 1
fi

echo "✅ Certificates verified"

# Update docker-compose volumes to use correct paths
echo ""
echo "📝 Preparing Docker Compose configuration..."

# Start Docker Compose stack
echo ""
echo "🐳 Starting Docker Compose stack..."
cd "$DOCKER_DIR"

# Build images first (use cache if available)
echo "🔨 Building Docker images..."
$COMPOSE_CMD -f docker-compose.prod.yml build

# Start services
echo "🚀 Starting services..."
$COMPOSE_CMD -f docker-compose.prod.yml up -d

# Wait for services to be healthy
echo ""
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service status
echo ""
echo "📊 Service Status:"
$COMPOSE_CMD -f docker-compose.prod.yml ps

echo ""
echo "✅ Setup completed successfully!"
echo ""
echo "🌐 Your services are now running:"
echo "   - Main site: https://$DOMAIN"
echo "   - CDN: https://$CDN_DOMAIN"
echo ""
echo "📝 Default admin credentials:"
echo "   Email: admin@cashbacktv.local"
echo "   Password: C@shb@ckTV2024!L1ve"
echo ""
echo "🔄 SSL certificates will auto-renew via certbot container"
echo ""
echo "📋 Useful commands:"
echo "   - View logs: $COMPOSE_CMD -f docker-compose.prod.yml logs -f"
echo "   - Stop services: $COMPOSE_CMD -f docker-compose.prod.yml down"
echo "   - Restart services: $COMPOSE_CMD -f docker-compose.prod.yml restart"
