#!/bin/bash

# Exotel Monitoring Platform - Local Setup and Run Script
# This script automates the setup and startup of the application locally

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[SETUP]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check prerequisites
log "Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    error "Go is not installed"
    exit 1
fi
log "Go version: $(go version)"

# Check MySQL
if ! command -v mysql &> /dev/null; then
    error "MySQL is not installed. Run: brew install mysql"
    exit 1
fi
log "MySQL found at: $(which mysql)"

# Check Redis
if ! command -v redis-cli &> /dev/null; then
    error "Redis is not installed. Run: brew install redis"
    exit 1
fi
log "Redis found at: $(which redis-cli)"

# Start MySQL if not running
log "Checking MySQL service..."
if ! brew services list | grep -q "mysql.*started"; then
    log "Starting MySQL service..."
    brew services start mysql
    sleep 3
fi
log "MySQL service is running"

# Start Redis if not running
log "Checking Redis service..."
if ! brew services list | grep -q "redis.*started"; then
    log "Starting Redis service..."
    brew services start redis
    sleep 3
fi
log "Redis service is running"

# Verify MySQL connection
log "Verifying MySQL connection..."
if ! mysql -u root -e "SELECT 1" &> /dev/null; then
    error "Cannot connect to MySQL"
    exit 1
fi
log "MySQL connection OK"

# Verify Redis connection
log "Verifying Redis connection..."
if ! redis-cli ping &> /dev/null; then
    error "Cannot connect to Redis"
    exit 1
fi
log "Redis connection OK"

# Check if database exists
log "Checking if database exists..."
if ! mysql -u root -e "USE exotel_monitoring" &> /dev/null; then
    log "Database not found. Initializing schema..."
    mysql -u root < scripts/init.sql
    log "Database schema initialized"
else
    log "Database exists"
fi

# Ensure config file exists
log "Checking configuration..."
if [ ! -f "configs/config.yaml" ]; then
    if [ -f "configs/config.yaml.example" ]; then
        log "Creating config from example..."
        cp configs/config.yaml.example configs/config.yaml
    else
        error "No config file found"
        exit 1
    fi
fi

# Check if secret key is set
if grep -q "CHANGE_ME" configs/config.yaml; then
    warn "WARNING: crypto.secret_key needs to be configured"
    warn "Generate with: openssl rand -base64 32"
    warn "Update 'crypto.secret_key' in configs/config.yaml"
fi

# Build application
log "Building application..."
go mod tidy
go build -o server ./cmd/server

log "Build complete!"

# Display startup info
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Exotel Monitoring Platform${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "MySQL:    ${GREEN}Connected (localhost:3306)${NC}"
echo -e "Redis:    ${GREEN}Connected (localhost:6379)${NC}"
echo -e "Build:    ${GREEN}OK (./server)${NC}"
echo ""
echo -e "To start the server, run:"
echo -e "  ${YELLOW}./server${NC} or ${YELLOW}go run cmd/server/main.go${NC}"
echo ""
echo -e "Health check:"
echo -e "  ${YELLOW}curl http://localhost:8080/health${NC}"
echo ""
echo -e "${GREEN}========================================${NC}"
