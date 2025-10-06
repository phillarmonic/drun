#!/bin/bash

# Test all drun examples for regressions
# This script runs through all .drun files and tests them

# set -e  # Don't exit on errors, continue testing all files

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Arrays to track results
PASSED_FILES=()
FAILED_FILES=()
SKIPPED_FILES=()

echo -e "${BLUE}🧪 Testing all drun examples for regressions...${NC}"
echo "=================================================="

# Build fresh binary
echo -e "${BLUE}📦 Building fresh drun binary...${NC}"
cd /Users/andy/repos/phillarmonic/drun
go build -o xdrun ./cmd/drun
echo -e "${GREEN}✅ Build completed${NC}"
echo

# Function to test a single file
test_file() {
    local file="$1"
    local filename=$(basename "$file")
    
    echo -e "${BLUE}🔍 Testing: ${filename}${NC}"
    
    # First, try to list tasks to see if file parses
    if ! ./xdrun -f "$file" -l > /dev/null 2>&1; then
        echo -e "${RED}❌ FAILED: ${filename} - Parse error${NC}"
        FAILED_FILES+=("$filename (parse error)")
        ((FAILED_TESTS++))
        return 1
    fi
    
    # Get the first task from the file (handle multi-word task names)
    # Task list format: "  task_name           description"
    # We need to extract everything before the description (2+ spaces separator)
    local first_task=$(./xdrun -f "$file" -l 2>/dev/null | grep -E "^  " | head -1 | sed 's/^  //' | sed -E 's/  +.*//' | xargs)
    
    if [ -z "$first_task" ]; then
        echo -e "${YELLOW}⚠️  SKIPPED: ${filename} - No tasks found${NC}"
        SKIPPED_FILES+=("$filename (no tasks)")
        ((SKIPPED_TESTS++))
        return 0
    fi
    
    # Store task name for execution
    local task_names=(
        "$first_task"
    )
    
    local success=false
    for task_name in "${task_names[@]}"; do
        if [ -n "$task_name" ]; then
            # Try to run the task in dry-run mode
            if ./xdrun -f "$file" "$task_name" --dry-run > /dev/null 2>&1; then
                echo -e "${GREEN}✅ PASSED: ${filename} (task: ${task_name})${NC}"
                PASSED_FILES+=("$filename")
                ((PASSED_TESTS++))
                success=true
                break
            else
                # If it failed, try with some common parameters
                local param_attempts=(
                    "name=test"
                    "environment=dev"
                    "items=test1,test2"
                    "source_path=/tmp/test"
                    "name=test environment=dev"
                    "name=test title=Mr"
                )
                
                for params in "${param_attempts[@]}"; do
                    if ./xdrun -f "$file" "$task_name" $params --dry-run > /dev/null 2>&1; then
                        echo -e "${GREEN}✅ PASSED: ${filename} (task: ${task_name} with params: ${params})${NC}"
                        PASSED_FILES+=("$filename")
                        ((PASSED_TESTS++))
                        success=true
                        break 2
                    fi
                done
            fi
        fi
    done
    
    if [ "$success" = false ]; then
        echo -e "${RED}❌ FAILED: ${filename} - Execution error${NC}"
        FAILED_FILES+=("$filename (execution error)")
        ((FAILED_TESTS++))
        return 1
    fi
    
    return 0
}

# Test all .drun files in examples directory
for file in examples/*.drun; do
    if [ -f "$file" ]; then
        ((TOTAL_TESTS++))
        test_file "$file"
        echo
    fi
done

# Print summary
echo "=================================================="
echo -e "${BLUE}📊 Test Summary${NC}"
echo "=================================================="
echo -e "Total files tested: ${TOTAL_TESTS}"
echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"
echo -e "${YELLOW}Skipped: ${SKIPPED_TESTS}${NC}"

if [ ${#PASSED_FILES[@]} -gt 0 ]; then
    echo
    echo -e "${GREEN}✅ Passed files:${NC}"
    for file in "${PASSED_FILES[@]}"; do
        echo "  - $file"
    done
fi

if [ ${#FAILED_FILES[@]} -gt 0 ]; then
    echo
    echo -e "${RED}❌ Failed files:${NC}"
    for file in "${FAILED_FILES[@]}"; do
        echo "  - $file"
    done
fi

if [ ${#SKIPPED_FILES[@]} -gt 0 ]; then
    echo
    echo -e "${YELLOW}⚠️  Skipped files:${NC}"
    for file in "${SKIPPED_FILES[@]}"; do
        echo "  - $file"
    done
fi

echo
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! No regressions detected.${NC}"
    exit 0
else
    echo -e "${RED}💥 ${FAILED_TESTS} test(s) failed. Regressions detected!${NC}"
    exit 1
fi
