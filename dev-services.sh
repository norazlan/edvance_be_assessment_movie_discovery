#!/bin/bash

# Development Script for Movie Discovery Platform
# Start and stop all services using go run

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Service definitions
SERVICES=(
    "movie-service:8081"
    "user-preference-service:8082"
    "recommendation-service:8083"
    "api-gateway:8080"
)

# PID directory for process tracking
PID_DIR="/tmp/movie-discovery-pids"
mkdir -p "$PID_DIR"

# Functions to display colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to start a service
start_service() {
    local service_dir=$1
    local port=$2
    local service_name=$(basename "$service_dir")
    local pid_file="$PID_DIR/${service_name}.pid"

    # Check if service is already running
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_warning "$service_name is already running (PID: $pid)"
            return 0
        else
            # PID file exists but process is not running
            rm -f "$pid_file"
        fi
    fi

    print_info "Starting $service_name on port $port..."
    
    # Go to service directory and run go run in background
    cd "$service_dir"
    nohup go run cmd/main.go > "/tmp/${service_name}.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$pid_file"
    
    cd - > /dev/null

    # Wait a moment and check if process is still running
    sleep 2
    if ps -p "$pid" > /dev/null 2>&1; then
        print_success "$service_name started successfully (PID: $pid)"
        return 0
    else
        print_error "$service_name failed to start. Check log: /tmp/${service_name}.log"
        rm -f "$pid_file"
        return 1
    fi
}

# Kill all processes listening on a given port
kill_port_processes() {
    local port=$1
    local pids
    pids=$(lsof -ti:"$port" 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill -9 2>/dev/null
        return 0
    fi
    return 1
}

# Function to stop a service
stop_service() {
    local service_name=$1
    local port=$2
    local pid_file="$PID_DIR/${service_name}.pid"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "Stopping $service_name (PID: $pid)..."

            # Kill all child processes first (go run spawns a child binary)
            pkill -P "$pid" 2>/dev/null
            kill "$pid" 2>/dev/null

            # Wait for process to stop gracefully
            local count=0
            while ps -p "$pid" > /dev/null 2>&1 && [ $count -lt 5 ]; do
                sleep 1
                count=$((count + 1))
            done

            # Force kill if still running
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null
                pkill -9 -P "$pid" 2>/dev/null
            fi
        fi
        rm -f "$pid_file"
    fi

    # Fallback: kill any process still listening on the port
    if [ -n "$port" ] && lsof -ti:"$port" > /dev/null 2>&1; then
        print_info "Cleaning up remaining processes on port $port..."
        kill_port_processes "$port"
        sleep 1
    fi

    # Verify port is free
    if [ -n "$port" ] && lsof -ti:"$port" > /dev/null 2>&1; then
        print_error "Failed to free port $port for $service_name"
        return 1
    else
        print_success "$service_name stopped successfully (port $port is free)"
        return 0
    fi
}

# Function to display service status
show_status() {
    echo ""
    echo "════════════════════════════════════════════════════════"
    echo "  SERVICE STATUS - Movie Discovery Platform"
    echo "═══════════════════════════════════════════════════════="
    echo ""
    printf "%-30s %-10s %-10s\n" "SERVICE" "PORT" "STATUS"
    echo "────────────────────────────────────────────────────────"
    
    for service_info in "${SERVICES[@]}"; do
        IFS=':' read -r service_dir port <<< "$service_info"
        local service_name=$(basename "$service_dir")
        local pid_file="$PID_DIR/${service_name}.pid"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                printf "%-30s %-10s ${GREEN}%-10s${NC}\n" "$service_name" ":$port" "RUNNING"
            else
                printf "%-30s %-10s ${RED}%-10s${NC}\n" "$service_name" ":$port" "STOPPED"
            fi
        else
            printf "%-30s %-10s ${RED}%-10s${NC}\n" "$service_name" ":$port" "STOPPED"
        fi
    done
    
    echo "════════════════════════════════════════════════════════"
    echo ""
}

# Function to start all services
start_all() {
    print_info "Starting all services..."
    echo ""
    
    local workspace_root="/workspaces/movie_discovery_platform"
    local failed=0
    
    for service_info in "${SERVICES[@]}"; do
        IFS=':' read -r service_dir port <<< "$service_info"
        if ! start_service "$workspace_root/$service_dir" "$port"; then
            failed=1
        fi
        echo ""
    done
    
    if [ $failed -eq 0 ]; then
        echo ""
        print_success "All services started successfully!"
        echo ""
        sleep 3
        show_status
    else
        echo ""
        print_error "Some services failed to start. Check logs in /tmp/"
    fi
}

# Function to stop all services
stop_all() {
    print_info "Stopping all services..."
    echo ""
    
    local failed=0
    
    # Stop in reverse order (gateway first, then services)
    for ((idx=${#SERVICES[@]}-1 ; idx>=0 ; idx--)); do
        service_info="${SERVICES[idx]}"
        IFS=':' read -r service_dir port <<< "$service_info"
        local service_name=$(basename "$service_dir")
        
        if ! stop_service "$service_name" "$port"; then
            failed=1
        fi
        echo ""
    done
    
    if [ $failed -eq 0 ]; then
        print_success "All services stopped successfully!"
    else
        print_error "Some services failed to stop"
    fi
}

# Function to restart all services
restart_all() {
    print_info "Restarting all services..."
    echo ""
    stop_all
    echo ""
    sleep 2
    start_all
}

# Function to display service logs
show_logs() {
    local service_name=$1
    local log_file="/tmp/${service_name}.log"
    
    if [ -f "$log_file" ]; then
        print_info "Displaying logs for $service_name..."
        echo ""
        tail -f "$log_file"
    else
        print_error "Log file not found: $log_file"
    fi
}

# Function to display usage
show_usage() {
    cat << EOF

Usage: $0 {start|stop|restart|status|logs}

Commands:
  start        Start all services and API gateway
  stop         Stop all services and API gateway
  restart      Restart all services and API gateway
  status       Display status of all services
  logs [name]  Display service logs (example: logs movie-service)

Examples:
  $0 start              # Start all services
  $0 stop               # Stop all services
  $0 status             # Check service status
  $0 logs api-gateway   # Display API gateway logs

Available services:
  - movie-service (port 8081)
  - user-preference-service (port 8082)
  - recommendation-service (port 8083)
  - api-gateway (port 8080)

EOF
}

# Main script
case "${1:-}" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        restart_all
        ;;
    status)
        show_status
        ;;
    logs)
        if [ -z "${2:-}" ]; then
            print_error "Please specify service name"
            echo "Example: $0 logs movie-service"
            exit 1
        fi
        show_logs "$2"
        ;;
    *)
        show_usage
        exit 1
        ;;
esac

exit 0
