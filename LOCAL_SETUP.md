# Local Setup Guide - Exotel Monitoring Platform

## Prerequisites

- Go 1.22+ (confirmed installed)
- MySQL 8 (will install via Homebrew)
- Redis 7 (will install via Homebrew)

## Installation Steps

### 1. Install MySQL and Redis via Homebrew

```bash
# Install MySQL
brew install mysql

# Install Redis
brew install redis

# Start MySQL
brew services start mysql

# Start Redis
brew services start redis
```

### 2. Initialize Database

```bash
# Connect to MySQL and run init script
mysql -u root < scripts/init.sql
```

The init script will create:
- Database: `exotel_monitoring`
- Tables: accounts, exophones, heartbeat_metrics, transactions, alerts, snapshots

### 3. Configure Application

Edit `configs/config.yaml`:

```yaml
mysql:
  host: localhost
  port: 3306
  username: root
  password: ""  # Default MySQL password on macOS
  database: exotel_monitoring

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

# Required: Generate a 32-byte base64 key
# openssl rand -base64 32
crypto:
  secret_key: "YOUR_32_BYTE_BASE64_KEY_HERE"

# Optional: Configure Slack alerts
slack:
  webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  enabled: false
```

### 4. Run Application

```bash
# Terminal 1: Start the server
go run cmd/server/main.go

# The app will listen on http://localhost:8080
```

### 5. Verify Health

```bash
# In another terminal:
curl http://localhost:8080/health

# Expected response:
# {"status":"UP","mysql":"CONNECTED","redis":"CONNECTED"}
```

## Available Endpoints

- `GET /health` - Infrastructure health check
- `GET /api/v1/dashboard/summary` - Account-level KPI summary
- `GET /api/v1/alerts/active` - Active alerts
- `GET /api/v1/transactions` - Job transaction log

## Stopping Services

```bash
# Stop MySQL
brew services stop mysql

# Stop Redis
brew services stop redis
```

## Troubleshooting

### MySQL Connection Issues
- Check if MySQL is running: `brew services list`
- Default password is empty on macOS. Update `configs/config.yaml` if needed.
- Verify database created: `mysql -u root -e "SHOW DATABASES;"`

### Redis Connection Issues
- Check if Redis is running: `brew services list`
- Test Redis connection: `redis-cli ping`

### Build Errors
- Run `go mod tidy` to resolve dependencies
- Check Go version: `go version`

## Architecture

```
Local MySQL (3306)
    ↓
Exotel Monitoring Platform (8080)
    ↓
Local Redis (6379)
```
