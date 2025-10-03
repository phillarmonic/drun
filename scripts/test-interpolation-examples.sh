#!/bin/bash

# Comprehensive test script for variable interpolation examples
# This script tests specific interpolation scenarios to ensure they work correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}🧪 Testing Variable Interpolation Examples${NC}"
echo "=============================================="
echo

# Build fresh binary
echo -e "${BLUE}📦 Building fresh drun binary...${NC}"
go build -o drun ./cmd/drun
echo -e "${GREEN}✅ Build completed${NC}"
echo

# Function to run a test and check output
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_patterns=("${@:3}")
    
    ((TOTAL_TESTS++))
    echo -e "${CYAN}🔍 Test: ${test_name}${NC}"
    echo -e "${YELLOW}Command: ${command}${NC}"
    
    # Capture output
    local output
    if output=$(eval "$command" 2>&1); then
        echo -e "${BLUE}Output:${NC}"
        echo "$output" | sed 's/^/  /'
        
        # Check each expected pattern
        local all_patterns_found=true
        for pattern in "${expected_patterns[@]}"; do
            if echo "$output" | grep -q "$pattern"; then
                echo -e "${GREEN}  ✅ Found expected pattern: ${pattern}${NC}"
            else
                echo -e "${RED}  ❌ Missing expected pattern: ${pattern}${NC}"
                all_patterns_found=false
            fi
        done
        
        if [ "$all_patterns_found" = true ]; then
            echo -e "${GREEN}✅ PASSED: ${test_name}${NC}"
            ((PASSED_TESTS++))
        else
            echo -e "${RED}❌ FAILED: ${test_name} - Missing expected patterns${NC}"
            ((FAILED_TESTS++))
        fi
    else
        echo -e "${RED}❌ FAILED: ${test_name} - Command failed${NC}"
        echo -e "${RED}Error output:${NC}"
        echo "$output" | sed 's/^/  /'
        ((FAILED_TESTS++))
    fi
    
    echo
}

# Test 1: Basic interpolation with default values
run_test "Basic interpolation with defaults" \
    "./xdrun -f examples/03-interpolation.drun greet name=Andy" \
    "Hello, friend Andy!" \
    "Processing greeting for Andy" \
    "Greeting completed for friend Andy!"

# Test 2: Custom parameter values
run_test "Custom parameter values" \
    "./xdrun -f examples/03-interpolation.drun greet name=Bob title=buddy" \
    "Hello, buddy Bob!" \
    "Processing greeting for Bob" \
    "Greeting completed for buddy Bob!"

# Test 3: Deploy task with constraints
run_test "Deploy task with constraints" \
    "./xdrun -f examples/03-interpolation.drun deploy environment=staging app_version=v1.2.3" \
    "Deploying version v1.2.3 to staging" \
    "Environment: staging" \
    "Version: v1.2.3" \
    "Deployment to staging completed!"

# Test 4: Deploy with default version
run_test "Deploy with default version" \
    "./xdrun -f examples/03-interpolation.drun deploy environment=dev" \
    "Deploying version latest to dev" \
    "Environment: dev" \
    "Version: latest" \
    "Deployment to dev completed!"

# Test 5: Backup task with default name
run_test "Backup with default name" \
    "./xdrun -f examples/03-interpolation.drun backup source_path=/home/user/data" \
    "Creating backup: backup-2024-01-01" \
    "Source: /home/user/data" \
    "Backup: backup-2024-01-01" \
    "Backup created: backup-2024-01-01"

# Test 6: Backup with custom name
run_test "Backup with custom name" \
    "./xdrun -f examples/03-interpolation.drun backup source_path=/tmp/data backup_name=my-backup" \
    "Creating backup: my-backup" \
    "Source: /tmp/data" \
    "Backup: my-backup" \
    "Backup created: my-backup"

# Test 7: Edge case - empty parameter value
echo -e "${CYAN}🔍 Test: Empty parameter values${NC}"
cat > /tmp/test-empty.drun << 'EOF'
version: 2.0

task "empty":
  requires $name
  given $title defaults to ""
  
  info "Name: '{$name}', Title: '{$title}'"
EOF

run_test "Empty parameter values" \
    "./xdrun -f /tmp/test-empty.drun empty name=" \
    "Name: '', Title: ''"

# Test 8: Edge case - special characters
echo -e "${CYAN}🔍 Test: Special characters in parameters${NC}"
cat > /tmp/test-special.drun << 'EOF'
version: 2.0

task "special":
  requires $message
  
  info "Message: {$message}"
EOF

run_test "Special characters in parameters" \
    "./xdrun -f /tmp/test-special.drun special \"message=Hello World!\"" \
    "Message: Hello World!"

# Test 9: Multiple same variable in one string
echo -e "${CYAN}🔍 Test: Multiple same variable${NC}"
cat > /tmp/test-repeat.drun << 'EOF'
version: 2.0

task "repeat":
  requires $word
  
  info "{$word} {$word} {$word}!"
EOF

run_test "Multiple same variable" \
    "./xdrun -f /tmp/test-repeat.drun repeat word=echo" \
    "echo echo echo!"

# Test 10: Undefined variable should remain as placeholder
echo -e "${CYAN}🔍 Test: Undefined variables${NC}"
cat > /tmp/test-undefined.drun << 'EOF'
version: 2.0

task "undefined":
  requires $name
  
  info "Hello {$name}, undefined: {$undefined_var}"
EOF

run_test "Undefined variables remain as placeholders" \
    "./xdrun -f /tmp/test-undefined.drun undefined name=Alice" \
    "Hello Alice, undefined: {\$undefined_var}"

# Clean up temporary files
rm -f /tmp/test-*.drun

# Print summary
echo "=============================================="
echo -e "${BLUE}📊 Test Summary${NC}"
echo "=============================================="
echo -e "Total tests: ${TOTAL_TESTS}"
echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"

echo
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 All interpolation tests passed!${NC}"
    echo -e "${GREEN}Variable interpolation is working correctly across all scenarios.${NC}"
    exit 0
else
    echo -e "${RED}💥 ${FAILED_TESTS} test(s) failed!${NC}"
    echo -e "${RED}Variable interpolation has issues that need to be addressed.${NC}"
    exit 1
fi
