#!/bin/bash

# =============================================================================
# Go AI Agents - Live Regression Test Script
# =============================================================================
# This script runs live integration tests against actual LLM providers
# DO NOT COMMIT - For local testing only

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$TEST_DIR")"
LOG_FILE="$TEST_DIR/test-results.log"
SERVER_PID=""
SERVER_PORT=18080

# Cleanup function
cleanup() {
    if [[ -n "$SERVER_PID" ]]; then
        echo -e "${YELLOW}Stopping test server (PID: $SERVER_PID)...${NC}"
        kill -TERM "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

# Set up cleanup on exit
trap cleanup EXIT

# Helper functions
log() {
    echo -e "$1" | tee -a "$LOG_FILE"
}

success() {
    log "${GREEN}✓ $1${NC}"
}

warning() {
    log "${YELLOW}⚠ $1${NC}"
}

error() {
    log "${RED}✗ $1${NC}"
}

info() {
    log "${BLUE}ℹ $1${NC}"
}

check_env() {
    local var_name="$1"
    local var_value="${!var_name}"
    
    if [[ -z "$var_value" ]]; then
        error "Environment variable $var_name is not set"
        return 1
    fi
    
    # Check if it looks like a placeholder
    if [[ "$var_value" == *"your_"* ]] || [[ "$var_value" == *"_here" ]]; then
        error "Environment variable $var_name appears to be a placeholder: $var_value"
        return 1
    fi
    
    success "Environment variable $var_name is set"
    return 0
}

# Start of script
log "${BLUE}================================================================${NC}"
log "${BLUE}Go AI Agents - Live Regression Test${NC}"
log "${BLUE}Started at: $(date)${NC}"
log "${BLUE}================================================================${NC}"

# Initialize log file
echo "Go AI Agents Live Regression Test - $(date)" > "$LOG_FILE"

# Change to root directory
cd "$ROOT_DIR"

# Check if .env file exists
if [[ ! -f ".env" ]]; then
    error ".env file not found. Please copy .env.example to .env and configure it."
    exit 1
fi

# Load environment variables
set -o allexport
source .env
set +o allexport

info "Loaded environment variables from .env"

# Check required environment variables
info "Checking environment variables..."

ENV_OK=true

if ! check_env "OPENAI_API_KEY"; then
    ENV_OK=false
fi

if ! check_env "ANTHROPIC_API_KEY"; then
    ENV_OK=false
fi

if [[ "$ENV_OK" != "true" ]]; then
    error "Environment validation failed. Please check your .env file."
    exit 1
fi

success "All required environment variables are set"

# Build the test server
info "Building test server..."
if ! go build -o "$TEST_DIR/test-server" "$TEST_DIR/main.go"; then
    error "Failed to build test server"
    exit 1
fi
success "Test server built successfully"

# Start the test server
info "Starting test server on port $SERVER_PORT..."
HTTP_PORT="$SERVER_PORT" "$TEST_DIR/test-server" > "$TEST_DIR/server.log" 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Check if server is running
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    error "Test server failed to start. Check $TEST_DIR/server.log"
    cat "$TEST_DIR/server.log"
    exit 1
fi

# Wait for server to be ready
info "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s "http://localhost:$SERVER_PORT/health" > /dev/null 2>&1; then
        success "Test server is ready"
        break
    fi
    if [[ $i -eq 30 ]]; then
        error "Server did not become ready in time"
        exit 1
    fi
    sleep 1
done

# Run the tests
info "Running live regression tests..."

# Test health endpoint
info "Testing health endpoint..."
if curl -sf "http://localhost:$SERVER_PORT/health" > /dev/null; then
    success "Health endpoint working"
else
    error "Health endpoint failed"
    exit 1
fi

# Run Go test with live integration tag
info "Running Go integration tests..."
if HTTP_PORT="$SERVER_PORT" go test -v -tags=integration ./regression-test-backend/...; then
    success "Go integration tests passed"
else
    error "Go integration tests failed"
    exit 1
fi

success "All regression tests completed successfully!"
log "${GREEN}================================================================${NC}"
log "${GREEN}Regression test completed successfully at: $(date)${NC}"
log "${GREEN}================================================================${NC}"