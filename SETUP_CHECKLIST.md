# Local Setup Checklist

## Phase 1: Database Installation (In Progress)
- [ ] MySQL 8 installation via Homebrew
- [ ] Redis 7 installation via Homebrew
- [ ] Start MySQL service: `brew services start mysql`
- [ ] Start Redis service: `brew services start redis`

## Phase 2: Database Configuration
- [ ] Initialize database schema: `mysql -u root < scripts/init.sql`
- [ ] Verify MySQL connection: `mysql -u root -e "SHOW DATABASES;"`
- [ ] Verify Redis connection: `redis-cli ping` (should respond PONG)

## Phase 3: Application Configuration
- [ ] Copy and edit `configs/config.yaml`:
  ```bash
  cp configs/config.yaml configs/config.yaml.local
  ```
- [ ] Generate 32-byte base64 secret key:
  ```bash
  openssl rand -base64 32
  ```
- [ ] Update `crypto.secret_key` in config with generated key
- [ ] Verify MySQL connection settings:
  - host: localhost
  - port: 3306
  - username: root
  - password: (empty by default on macOS)
  - database: exotel_monitoring
- [ ] Verify Redis connection settings:
  - host: localhost
  - port: 6379
  - password: (empty)
  - db: 0

## Phase 4: Application Startup
- [ ] Build server: `go build ./cmd/server`
- [ ] Start server: `go run cmd/server/main.go`
- [ ] Expected output: "HTTP server listening on :8080"

## Phase 5: Health Verification
- [ ] Check health endpoint: `curl http://localhost:8080/health`
- [ ] Expected response:
  ```json
  {"status":"UP","mysql":"CONNECTED","redis":"CONNECTED"}
  ```

## Phase 6: Optional - Slack Integration
- [ ] Create Slack webhook: https://api.slack.com/messaging/webhooks
- [ ] Update `configs/config.yaml`:
  ```yaml
  slack:
    webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    enabled: true
  ```

## Key Files Modified for Bug Fixes
- `internal/snapshots/generator.go`
  - Added `strconv` import for duration parsing
  - Fixed `UpdateCallMetricsSnapshot` function signature
  - Corrected duration parsing logic with proper error handling

## Troubleshooting Commands

### Check Service Status
```bash
brew services list
```

### View MySQL Logs
```bash
tail -f /usr/local/var/log/mysql/$(hostname).err
```

### View Redis Logs
```bash
tail -f /usr/local/var/log/redis.log
```

### Reset MySQL (if needed)
```bash
brew services stop mysql
rm -rf /usr/local/var/mysql
mysql.server start
```

### Verify Build
```bash
go build -o ./server ./cmd/server
./server
```

## Estimated Time
- MySQL + Redis installation: 5-10 minutes
- Database initialization: 1-2 minutes
- Configuration setup: 2-3 minutes
- Server startup: <1 minute
- **Total: ~10-15 minutes**

## Next Steps After Successful Setup
1. Explore API endpoints at http://localhost:8080
2. Check active alerts: `curl http://localhost:8080/api/v1/alerts/active`
3. View dashboard summary: `curl http://localhost:8080/api/v1/dashboard/summary`
4. Configure monitoring jobs in `scheduler/scheduler.go`
5. Set up Exotel API credentials in config for real data ingestion
