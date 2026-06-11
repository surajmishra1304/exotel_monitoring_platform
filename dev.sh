#!/usr/bin/env bash
# dev.sh — start or stop the Exotel Monitoring Platform locally
#
#   ./dev.sh start   — ensure MySQL/Redis are up, build & run backend + frontend
#   ./dev.sh stop    — cleanly kill backend and frontend

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_DIR="$SCRIPT_DIR/.pids"
BACKEND_PID="$PID_DIR/backend.pid"
FRONTEND_PID="$PID_DIR/frontend.pid"
BACKEND_LOG="$SCRIPT_DIR/logs/backend.log"
FRONTEND_LOG="$SCRIPT_DIR/logs/frontend.log"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
info() { echo -e "${YELLOW}[INFO]${NC}  $*"; }
fail() { echo -e "${RED}[FAIL]${NC}  $*" >&2; exit 1; }

# ── helpers ─────────────────────────────────────────────────────────────────

pid_running() { [[ -f "$1" ]] && kill -0 "$(cat "$1")" 2>/dev/null; }

stop_pid() {
  local label="$1" pidfile="$2"
  if pid_running "$pidfile"; then
    kill "$(cat "$pidfile")" 2>/dev/null && ok "Stopped $label (pid $(cat "$pidfile"))"
    rm -f "$pidfile"
  else
    info "$label was not running"
    rm -f "$pidfile"
  fi
}

wait_for_port() {
  local port="$1" label="$2" attempts=0
  while ! curl -s "http://localhost:$port" >/dev/null 2>&1; do
    ((attempts++))
    [[ $attempts -ge 20 ]] && fail "$label did not come up on :$port after 20s"
    sleep 1
  done
}

# ── start ───────────────────────────────────────────────────────────────────

cmd_start() {
  mkdir -p "$PID_DIR" "$SCRIPT_DIR/logs"

  # Guard: already running?
  if pid_running "$BACKEND_PID"; then
    fail "Backend already running (pid $(cat "$BACKEND_PID")). Run './dev.sh stop' first."
  fi

  # 1. MySQL
  info "Checking MySQL..."
  if ! brew services list | grep -q "mysql.*started"; then
    brew services start mysql && sleep 3
  fi
  mysql -u root -proot -e "SELECT 1" >/dev/null 2>&1 || fail "Cannot connect to MySQL (root/root). Check credentials."
  ok "MySQL"

  # 2. Redis
  info "Checking Redis..."
  if ! brew services list | grep -q "redis.*started"; then
    brew services start redis && sleep 2
  fi
  redis-cli ping >/dev/null 2>&1 || fail "Cannot reach Redis on localhost:6379"
  ok "Redis"

  # 3. Build backend
  info "Building backend..."
  cd "$SCRIPT_DIR"
  go build -o "$SCRIPT_DIR/server" ./cmd/server 2>&1 || fail "Go build failed"
  ok "Backend built"

  # 4. Start backend
  info "Starting backend..."
  "$SCRIPT_DIR/server" >> "$BACKEND_LOG" 2>&1 &
  echo $! > "$BACKEND_PID"
  wait_for_port 8080 "Backend"
  ok "Backend running  → http://localhost:8080  (log: logs/backend.log)"

  # 5. Start frontend
  info "Starting frontend..."
  cd "$SCRIPT_DIR/exotel-monitoring-frontend"
  npm run dev >> "$FRONTEND_LOG" 2>&1 &
  echo $! > "$FRONTEND_PID"
  wait_for_port 5173 "Frontend"
  ok "Frontend running → http://localhost:5173  (log: logs/frontend.log)"

  echo ""
  echo -e "${GREEN}════════════════════════════════════════${NC}"
  echo -e "${GREEN}  Platform is UP${NC}"
  echo -e "${GREEN}════════════════════════════════════════${NC}"
  echo -e "  API     → http://localhost:8080/health"
  echo -e "  UI      → http://localhost:5173"
  echo -e "  Stop    → ./dev.sh stop"
  echo -e "${GREEN}════════════════════════════════════════${NC}"
}

# ── stop ────────────────────────────────────────────────────────────────────

cmd_stop() {
  stop_pid "Frontend" "$FRONTEND_PID"
  stop_pid "Backend"  "$BACKEND_PID"

  # Belt-and-suspenders: kill anything still holding the ports
  lsof -ti :8080 | xargs kill -9 2>/dev/null || true
  lsof -ti :5173 | xargs kill -9 2>/dev/null || true

  # Remove compiled binary
  rm -f "$SCRIPT_DIR/server"

  ok "Platform stopped"
}

# ── dispatch ─────────────────────────────────────────────────────────────────

case "${1:-}" in
  start) cmd_start ;;
  stop)  cmd_stop  ;;
  *) echo "Usage: $0 {start|stop}" >&2; exit 1 ;;
esac
