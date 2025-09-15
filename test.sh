#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=70
VERBOSE=false
COVERAGE=false
RACE=false
BENCH=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -b|--bench)
            BENCH=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose    Enable verbose test output"
            echo "  -c, --coverage   Generate coverage report"
            echo "  -r, --race       Enable race detection"
            echo "  -b, --bench      Run benchmarks"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Run basic tests"
            echo "  $0 -c                 # Run tests with coverage"
            echo "  $0 -v -c -r           # Verbose tests with coverage and race detection"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✅${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}❌${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BLUE}===${NC} $1 ${BLUE}===${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_section "Checking Prerequisites"
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $GO_VERSION"
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]]; then
        log_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi
    
    PROJECT_NAME=$(grep "^module" go.mod | awk '{print $2}')
    log_info "Project: $PROJECT_NAME"
    
    log_success "Prerequisites check passed"
}

# Clean previous artifacts
clean_artifacts() {
    log_section "Cleaning Previous Artifacts"
    
    # Remove coverage files
    find . -name "*.out" -type f -delete 2>/dev/null || true
    find . -name "coverage.html" -type f -delete 2>/dev/null || true
    
    # Clean Go cache
    go clean -cache -testcache
    
    log_success "Cleaned previous artifacts"
}

# Install golangci-lint if not present
install_golangci_lint() {
    if ! command -v golangci-lint &> /dev/null; then
        log_info "golangci-lint not found, installing..."
        
        # Try to install golangci-lint
        if command -v go &> /dev/null; then
            # Install using go install
            if go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; then
                log_success "golangci-lint installed successfully"
            else
                log_error "Failed to install golangci-lint via go install"
                
                # Try curl installation as fallback
                log_info "Trying curl installation..."
                if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin; then
                    log_success "golangci-lint installed via curl"
                else
                    log_error "Failed to install golangci-lint. Please install it manually:"
                    log_error "  https://golangci-lint.run/usage/install/"
                    exit 1
                fi
            fi
        else
            log_error "Go not found. Cannot install golangci-lint."
            exit 1
        fi
    fi
}

# Run linting
run_linting() {
    log_section "Running Linting"
    
    # Ensure golangci-lint is installed
    install_golangci_lint
    
    # Verify golangci-lint is now available
    if ! command -v golangci-lint &> /dev/null; then
        log_error "golangci-lint is still not available after installation attempt"
        exit 1
    fi
    
    log_info "Running golangci-lint..."
    if golangci-lint run ./...; then
        log_success "Linting passed"
    else
        log_error "Linting failed - fix the issues before proceeding"
        log_info "Run 'golangci-lint run ./...' to see detailed issues"
        exit 1
    fi
}

# Build test flags
build_test_flags() {
    local flags=()
    
    if [[ "$VERBOSE" == true ]]; then
        flags+=("-v")
    fi
    
    if [[ "$RACE" == true ]]; then
        flags+=("-race")
    fi
    
    if [[ "$COVERAGE" == true ]]; then
        flags+=("-cover" "-coverprofile=coverage.out")
    fi
    
    # Handle empty array case
    if [[ ${#flags[@]} -eq 0 ]]; then
        echo ""
    else
        echo "${flags[@]}"
    fi
}

# Run unit tests
run_unit_tests() {
    log_section "Running Unit Tests"
    
    local test_flags
    test_flags=$(build_test_flags)
    
    log_info "Test flags: $test_flags"
    
    # Run tests for internal packages
    if eval "go test $test_flags ./internal/..."; then
        log_success "Unit tests passed"
    else
        log_error "Unit tests failed"
        exit 1
    fi
}

# Run integration tests (if they exist)
run_integration_tests() {
    log_section "Running Integration Tests"
    
    # Check if integration tests exist
    if find . -name "*_integration_test.go" -type f | grep -q .; then
        log_info "Found integration tests, running..."
        local test_flags
        test_flags=$(build_test_flags)
        
        if eval "go test $test_flags -tags=integration ./..."; then
            log_success "Integration tests passed"
        else
            log_error "Integration tests failed"
            exit 1
        fi
    else
        log_info "No integration tests found, skipping"
    fi
}

# Run benchmarks
run_benchmarks() {
    if [[ "$BENCH" == true ]]; then
        log_section "Running Benchmarks"
        
        if go test -bench=. -benchmem ./internal/...; then
            log_success "Benchmarks completed"
        else
            log_warning "Benchmarks had issues (continuing anyway)"
        fi
    fi
}

# Generate coverage report
generate_coverage_report() {
    if [[ "$COVERAGE" == true && -f "coverage.out" ]]; then
        log_section "Generating Coverage Report"
        
        # Calculate total coverage
        local total_coverage
        total_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        
        log_info "Total coverage: ${total_coverage}%"
        
        # Check coverage threshold
        if (( $(echo "$total_coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
            log_success "Coverage meets threshold (${COVERAGE_THRESHOLD}%)"
        else
            log_warning "Coverage below threshold: ${total_coverage}% < ${COVERAGE_THRESHOLD}%"
        fi
        
        # Generate HTML report
        go tool cover -html=coverage.out -o coverage.html
        log_info "HTML coverage report generated: coverage.html"
        
        # Show per-package coverage
        echo ""
        log_info "Per-package coverage:"
        go tool cover -func=coverage.out | grep -E "(internal/|total:)" | while read -r line; do
            if [[ $line == *"total:"* ]]; then
                echo -e "  ${GREEN}$line${NC}"
            else
                echo "  $line"
            fi
        done
    fi
}

# Test build
test_build() {
    log_section "Testing Build"
    
    log_info "Building drun binary..."
    if go build -o bin/drun ./cmd/drun; then
        log_success "Build successful"
        
        # Test basic functionality
        log_info "Testing basic functionality..."
        if ./bin/drun --version >/dev/null 2>&1; then
            log_success "Binary works correctly"
        else
            log_error "Binary execution failed"
            exit 1
        fi
        
        # Clean up
        rm -f bin/drun
    else
        log_error "Build failed"
        exit 1
    fi
}

# Test examples
test_examples() {
    log_section "Testing Example Configurations"
    
    # Build binary for example testing
    if ! go build -o bin/drun ./cmd/drun; then
        log_error "Failed to build binary for example testing"
        return 1
    fi
    
    if [[ -d "examples" ]]; then
        local example_count=0
        local success_count=0
        
        for example_file in examples/*.yml examples/*.yaml; do
            if [[ -f "$example_file" ]]; then
                ((example_count++))
                log_info "Testing $example_file..."
                
                if ./bin/drun -f "$example_file" --list >/dev/null 2>&1; then
                    ((success_count++))
                    echo "  ✅ Valid"
                else
                    echo "  ❌ Invalid"
                fi
            fi
        done
        
        if [[ $example_count -gt 0 ]]; then
            log_info "Example validation: $success_count/$example_count passed"
            if [[ $success_count -eq $example_count ]]; then
                log_success "All examples are valid"
            else
                log_warning "Some examples have issues"
            fi
        else
            log_info "No example files found"
        fi
    else
        log_info "No examples directory found"
    fi
    
    # Clean up binary
    rm -f bin/drun
}

# Main execution
main() {
    echo -e "${BLUE}🧪 drun Test Suite${NC}"
    echo "=================="
    
    local start_time
    start_time=$(date +%s)
    
    # Run all test phases
    check_prerequisites
    clean_artifacts
    run_linting
    run_unit_tests
    run_integration_tests
    run_benchmarks
    generate_coverage_report
    test_build
    test_examples
    
    # Calculate duration
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Final summary
    log_section "Test Summary"
    log_success "All tests completed successfully!"
    log_info "Total duration: ${duration}s"
    
    if [[ "$COVERAGE" == true ]]; then
        log_info "Coverage report: coverage.html"
    fi
    
    echo ""
    echo -e "${GREEN}🎉 Test suite passed!${NC}"
}

# Run main function
main "$@"
