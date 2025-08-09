#!/bin/bash

# Test runner script for Go-ORMX unit tests
# This script runs all unit tests with proper reporting

set -e

echo "======================================"
echo "Go-ORMX Unit Test Suite"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to run tests for a specific package
run_package_tests() {
    local package=$1
    local name=$2
    
    echo -e "\n${BLUE}Testing $name package...${NC}"
    echo "----------------------------------------"
    
    if go test -v -race -cover ./tests/unit/$package/...; then
        echo -e "${GREEN}✓ $name tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $name tests failed${NC}"
        return 1
    fi
}

# Function to run benchmarks
run_benchmarks() {
    local package=$1
    local name=$2
    
    echo -e "\n${YELLOW}Benchmarking $name package...${NC}"
    echo "----------------------------------------"
    
    go test -bench=. -benchmem ./tests/unit/$package/... || true
}

# Main test execution
main() {
    local failed_packages=()
    local total_packages=0
    
    # Ensure we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -d "tests/unit" ]]; then
        echo -e "${RED}Error: Please run this script from the go-ormx root directory${NC}"
        exit 1
    fi
    
    echo "Starting unit tests..."
    echo "Date: $(date)"
    echo "Go version: $(go version)"
    echo ""
    
    # Test each package individually for better reporting
    packages=(
        "config:Configuration Management"
        "logging:Structured Logging"
        "security:Security Utilities"
        "errors:Error Handling"
        "models:Data Models"
        "db:Database Layer"
        "repositories:Repository Pattern"
    )
    
    for package_info in "${packages[@]}"; do
        IFS=':' read -r package name <<< "$package_info"
        total_packages=$((total_packages + 1))
        
        if ! run_package_tests "$package" "$name"; then
            failed_packages+=("$name")
        fi
    done
    
    # Run all tests together for coverage report
    echo -e "\n${BLUE}Generating coverage report...${NC}"
    echo "----------------------------------------"
    
    if go test -race -coverprofile=coverage.out ./tests/unit/...; then
        go tool cover -html=coverage.out -o coverage.html
        echo -e "${GREEN}✓ Coverage report generated: coverage.html${NC}"
        
        # Show coverage summary
        go tool cover -func=coverage.out | tail -1
    else
        echo -e "${RED}✗ Failed to generate coverage report${NC}"
    fi
    
    # Run benchmarks if requested
    if [[ "$1" == "--bench" ]] || [[ "$1" == "-b" ]]; then
        echo -e "\n${YELLOW}Running benchmarks...${NC}"
        echo "======================================"
        
        for package_info in "${packages[@]}"; do
            IFS=':' read -r package name <<< "$package_info"
            run_benchmarks "$package" "$name"
        done
    fi
    
    # Final summary
    echo -e "\n======================================"
    echo -e "${BLUE}Test Summary${NC}"
    echo "======================================"
    
    local passed_packages=$((total_packages - ${#failed_packages[@]}))
    
    echo "Total packages tested: $total_packages"
    echo -e "Passed: ${GREEN}$passed_packages${NC}"
    echo -e "Failed: ${RED}${#failed_packages[@]}${NC}"
    
    if [[ ${#failed_packages[@]} -eq 0 ]]; then
        echo -e "\n${GREEN}🎉 All tests passed!${NC}"
        echo -e "${GREEN}The Go-ORMX package is ready for use.${NC}"
        exit 0
    else
        echo -e "\n${RED}❌ Some tests failed:${NC}"
        for failed_package in "${failed_packages[@]}"; do
            echo -e "${RED}  - $failed_package${NC}"
        done
        echo -e "\n${YELLOW}Please fix the failing tests before using the package.${NC}"
        exit 1
    fi
}

# Show help
show_help() {
    echo "Go-ORMX Unit Test Runner"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -b, --bench    Run benchmarks after tests"
    echo ""
    echo "Examples:"
    echo "  $0              # Run all unit tests"
    echo "  $0 --bench      # Run tests and benchmarks"
    echo ""
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -b|--bench)
        main --bench
        ;;
    "")
        main
        ;;
    *)
        echo -e "${RED}Error: Unknown option '$1'${NC}"
        show_help
        exit 1
        ;;
esac