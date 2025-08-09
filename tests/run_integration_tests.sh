#!/bin/bash

# run_integration_tests.sh - Script to run integration tests for the Go-ORMX module

set -e  # Exit on any error

# --- Configuration ---
MODULE_PATH="go-ormx"
INTEGRATION_TEST_DIR="./tests/integration"
COVERAGE_FILE="integration_coverage.out"
COVERAGE_HTML="integration_coverage.html"

# Test configuration
DEFAULT_POSTGRES_HOST="localhost"
DEFAULT_POSTGRES_PORT="5432"
DEFAULT_POSTGRES_USER="postgres"
DEFAULT_POSTGRES_PASSWORD="password"
DEFAULT_POSTGRES_DB="go-ormx_integration_test"

# --- Colors for output ---
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# --- Helper Functions ---
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_header() {
    echo -e "${BLUE}${BOLD}=== $1 ===${NC}"
}

log_subheader() {
    echo -e "${BLUE}--- $1 ---${NC}"
}

# --- Environment Setup ---
setup_environment() {
    log_header "Setting up Test Environment"
    
    # Set default environment variables if not already set
    export TEST_POSTGRES_HOST="${TEST_POSTGRES_HOST:-$DEFAULT_POSTGRES_HOST}"
    export TEST_POSTGRES_PORT="${TEST_POSTGRES_PORT:-$DEFAULT_POSTGRES_PORT}"
    export TEST_POSTGRES_USER="${TEST_POSTGRES_USER:-$DEFAULT_POSTGRES_USER}"
    export TEST_POSTGRES_PASSWORD="${TEST_POSTGRES_PASSWORD:-$DEFAULT_POSTGRES_PASSWORD}"
    export TEST_POSTGRES_DB="${TEST_POSTGRES_DB:-$DEFAULT_POSTGRES_DB}"
    
    log_info "Test Database Configuration:"
    log_info "  Host: $TEST_POSTGRES_HOST"
    log_info "  Port: $TEST_POSTGRES_PORT"
    log_info "  User: $TEST_POSTGRES_USER"
    log_info "  Database: $TEST_POSTGRES_DB"
    
    # Check for optional MySQL configuration
    if [ "$TEST_MYSQL_ENABLED" = "true" ]; then
        log_info "MySQL testing enabled"
        export TEST_MYSQL_HOST="${TEST_MYSQL_HOST:-localhost}"
        export TEST_MYSQL_PORT="${TEST_MYSQL_PORT:-3306}"
        export TEST_MYSQL_USER="${TEST_MYSQL_USER:-root}"
        export TEST_MYSQL_PASSWORD="${TEST_MYSQL_PASSWORD:-password}"
        export TEST_MYSQL_DB="${TEST_MYSQL_DB:-go-ormx_test}"
    fi
}

# --- Database Health Check ---
check_database_health() {
    log_header "Checking Database Health"
    
    # Check PostgreSQL connection
    log_info "Checking PostgreSQL connection..."
    if command -v psql >/dev/null 2>&1; then
        if PGPASSWORD="$TEST_POSTGRES_PASSWORD" psql -h "$TEST_POSTGRES_HOST" -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -d postgres -c "SELECT 1;" >/dev/null 2>&1; then
            log_info "✓ PostgreSQL connection successful"
        else
            log_warn "⚠ PostgreSQL connection failed - tests may fail"
            log_info "To start PostgreSQL with Docker:"
            log_info "  docker run --name go-ormx-postgres -e POSTGRES_PASSWORD=$TEST_POSTGRES_PASSWORD -p $TEST_POSTGRES_PORT:5432 -d postgres:13"
        fi
    else
        log_warn "psql not found - skipping PostgreSQL health check"
    fi
    
    # Check MySQL connection if enabled
    if [ "$TEST_MYSQL_ENABLED" = "true" ]; then
        log_info "Checking MySQL connection..."
        if command -v mysql >/dev/null 2>&1; then
            if mysql -h "$TEST_MYSQL_HOST" -P "$TEST_MYSQL_PORT" -u "$TEST_MYSQL_USER" -p"$TEST_MYSQL_PASSWORD" -e "SELECT 1;" >/dev/null 2>&1; then
                log_info "✓ MySQL connection successful"
            else
                log_warn "⚠ MySQL connection failed - MySQL tests will be skipped"
            fi
        else
            log_warn "mysql client not found - skipping MySQL health check"
        fi
    fi
}

# --- Test Execution Functions ---
run_test_suite() {
    local suite_name="$1"
    local test_path="$2"
    local test_flags="$3"
    
    log_subheader "Running $suite_name Tests"
    
    if [ ! -d "$test_path" ]; then
        log_warn "Test directory $test_path not found - skipping"
        return 0
    fi
    
    local start_time=$(date +%s)
    
    if go test "$test_path" $test_flags; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_info "✓ $suite_name tests completed in ${duration}s"
        return 0
    else
        log_error "✗ $suite_name tests failed"
        return 1
    fi
}

# --- Main Test Execution ---
run_all_integration_tests() {
    log_header "Running Integration Tests"
    
    local overall_status=0
    local test_flags="-v"
    
    # Add race detection if not disabled
    if [ "$DISABLE_RACE_DETECTION" != "true" ]; then
        test_flags="$test_flags -race"
    fi
    
    # Add coverage if requested
    if [ "$ENABLE_COVERAGE" = "true" ]; then
        test_flags="$test_flags -coverprofile=$COVERAGE_FILE"
    fi
    
    # Add short flag if requested
    if [ "$RUN_SHORT_TESTS" = "true" ]; then
        test_flags="$test_flags -short"
        log_info "Running short tests only (long-running tests will be skipped)"
    fi
    
    # Test suites in order of dependency/importance
    local test_suites=(
        "Database Configuration:$INTEGRATION_TEST_DIR/config"
        "Database Connections:$INTEGRATION_TEST_DIR/database"
        "Repository Operations:$INTEGRATION_TEST_DIR/repositories"
        "Transaction Management:$INTEGRATION_TEST_DIR/transactions"
        "Migration Tests:$INTEGRATION_TEST_DIR/migrations"
        "Error Handling:$INTEGRATION_TEST_DIR/errors"
        "Security & Validation:$INTEGRATION_TEST_DIR/security"
        "Performance Tests:$INTEGRATION_TEST_DIR/performance"
        "Logging & Observability:$INTEGRATION_TEST_DIR/logging"
    )
    
    for suite in "${test_suites[@]}"; do
        IFS=':' read -r suite_name test_path <<< "$suite"
        
        if ! run_test_suite "$suite_name" "$test_path" "$test_flags"; then
            overall_status=1
            if [ "$FAIL_FAST" = "true" ]; then
                log_error "Stopping on first failure (FAIL_FAST=true)"
                break
            fi
        fi
        
        echo # Add spacing between test suites
    done
    
    # Run all tests together if no specific suite failed and ALL_TESTS is enabled
    if [ $overall_status -eq 0 ] && [ "$RUN_ALL_TESTS" = "true" ]; then
        log_subheader "Running All Integration Tests Together"
        if ! go test "$INTEGRATION_TEST_DIR/..." $test_flags; then
            overall_status=1
        fi
    fi
    
    return $overall_status
}

# --- Performance Benchmarks ---
run_performance_benchmarks() {
    log_header "Running Performance Benchmarks"
    
    if [ ! -d "$INTEGRATION_TEST_DIR/performance" ]; then
        log_warn "Performance test directory not found - skipping benchmarks"
        return 0
    fi
    
    log_info "Running performance benchmarks..."
    if go test "$INTEGRATION_TEST_DIR/performance" -bench=. -benchmem -run=^$ -benchtime=5s; then
        log_info "✓ Performance benchmarks completed"
        return 0
    else
        log_error "✗ Performance benchmarks failed"
        return 1
    fi
}

# --- Stress Tests ---
run_stress_tests() {
    log_header "Running Stress Tests"
    
    export STRESS_TEST_ENABLED=true
    export PERF_TEST_CONCURRENCY="${PERF_TEST_CONCURRENCY:-50}"
    export PERF_TEST_DURATION="${PERF_TEST_DURATION:-30s}"
    
    log_info "Stress test configuration:"
    log_info "  Concurrency: $PERF_TEST_CONCURRENCY"
    log_info "  Duration: $PERF_TEST_DURATION"
    
    local stress_tests=(
        "$INTEGRATION_TEST_DIR/repositories"
        "$INTEGRATION_TEST_DIR/transactions"
        "$INTEGRATION_TEST_DIR/performance"
    )
    
    for test_path in "${stress_tests[@]}"; do
        if [ -d "$test_path" ]; then
            log_subheader "Stress testing $(basename "$test_path")"
            if ! go test "$test_path" -v -run=".*Stress.*|.*Concurrent.*" -timeout=2m; then
                log_error "Stress test failed for $(basename "$test_path")"
                return 1
            fi
        fi
    done
    
    log_info "✓ All stress tests completed"
    return 0
}

# --- Coverage Report ---
generate_coverage_report() {
    if [ "$ENABLE_COVERAGE" = "true" ] && [ -f "$COVERAGE_FILE" ]; then
        log_header "Generating Coverage Report"
        
        log_info "Generating HTML coverage report..."
        if go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"; then
            log_info "✓ Coverage report generated: $COVERAGE_HTML"
            
            # Show coverage summary
            log_info "Coverage summary:"
            go tool cover -func="$COVERAGE_FILE" | tail -1
        else
            log_warn "Failed to generate HTML coverage report"
        fi
        
        # Show coverage percentage
        local coverage_pct=$(go tool cover -func="$COVERAGE_FILE" | tail -1 | awk '{print $3}')
        log_info "Total coverage: $coverage_pct"
    fi
}

# --- Cleanup ---
cleanup() {
    log_header "Cleanup"
    
    # Clean up temporary files
    if [ -f "$COVERAGE_FILE" ] && [ "$KEEP_COVERAGE_FILE" != "true" ]; then
        rm -f "$COVERAGE_FILE"
        log_info "Cleaned up coverage file"
    fi
    
    # Additional cleanup can be added here
    log_info "Cleanup completed"
}

# --- Help ---
show_help() {
    cat << EOF
Go-ORMX Integration Test Runner

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -s, --short             Run short tests only (skip long-running tests)
    -c, --coverage          Enable coverage reporting
    -b, --benchmarks        Run performance benchmarks
    -t, --stress            Run stress tests
    -f, --fail-fast         Stop on first test failure
    -a, --all               Run all tests together after individual suites
    --no-race               Disable race detection
    --keep-coverage         Keep coverage files after completion

ENVIRONMENT VARIABLES:
    TEST_POSTGRES_HOST      PostgreSQL host (default: localhost)
    TEST_POSTGRES_PORT      PostgreSQL port (default: 5432)
    TEST_POSTGRES_USER      PostgreSQL user (default: postgres)
    TEST_POSTGRES_PASSWORD  PostgreSQL password (default: password)
    TEST_POSTGRES_DB        PostgreSQL database (default: go-ormx_integration_test)
    
    TEST_MYSQL_ENABLED      Enable MySQL tests (default: false)
    TEST_MYSQL_HOST         MySQL host (default: localhost)
    TEST_MYSQL_USER         MySQL user (default: root)
    TEST_MYSQL_PASSWORD     MySQL password (default: password)
    
    PERF_TEST_CONCURRENCY   Concurrency for stress tests (default: 50)
    PERF_TEST_DURATION      Duration for stress tests (default: 30s)

EXAMPLES:
    # Run all integration tests
    $0
    
    # Run short tests with coverage
    $0 --short --coverage
    
    # Run benchmarks only
    $0 --benchmarks
    
    # Run stress tests
    $0 --stress
    
    # Quick smoke test
    $0 --short --fail-fast

EOF
}

# --- Main Script Logic ---
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -s|--short)
                export RUN_SHORT_TESTS=true
                shift
                ;;
            -c|--coverage)
                export ENABLE_COVERAGE=true
                shift
                ;;
            -b|--benchmarks)
                export RUN_BENCHMARKS=true
                shift
                ;;
            -t|--stress)
                export RUN_STRESS_TESTS=true
                shift
                ;;
            -f|--fail-fast)
                export FAIL_FAST=true
                shift
                ;;
            -a|--all)
                export RUN_ALL_TESTS=true
                shift
                ;;
            --no-race)
                export DISABLE_RACE_DETECTION=true
                shift
                ;;
            --keep-coverage)
                export KEEP_COVERAGE_FILE=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # Ensure we are in the correct directory
    SCRIPT_DIR=$(dirname "$0")
    cd "$SCRIPT_DIR/.." || exit 1 # Go up to the go-ormx root
    
    log_header "Go-ORMX Integration Test Runner"
    log_info "Module: $MODULE_PATH"
    log_info "Test Directory: $INTEGRATION_TEST_DIR"
    
    # Setup and health checks
    setup_environment
    check_database_health
    
    local overall_status=0
    
    # Run main integration tests
    if [ "$RUN_BENCHMARKS" != "true" ] && [ "$RUN_STRESS_TESTS" != "true" ]; then
        if ! run_all_integration_tests; then
            overall_status=1
        fi
    fi
    
    # Run benchmarks if requested
    if [ "$RUN_BENCHMARKS" = "true" ]; then
        if ! run_performance_benchmarks; then
            overall_status=1
        fi
    fi
    
    # Run stress tests if requested
    if [ "$RUN_STRESS_TESTS" = "true" ]; then
        if ! run_stress_tests; then
            overall_status=1
        fi
    fi
    
    # Generate coverage report
    generate_coverage_report
    
    # Cleanup
    cleanup
    
    # Final status
    if [ $overall_status -eq 0 ]; then
        log_header "🎉 All Tests Passed!"
        log_info "Integration tests completed successfully"
    else
        log_header "❌ Some Tests Failed"
        log_error "Integration tests completed with failures"
    fi
    
    exit $overall_status
}

# Run main function with all arguments
main "$@"