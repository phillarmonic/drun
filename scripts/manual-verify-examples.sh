#!/bin/bash

# Manual verification script for examples in numerical order
# Usage: ./manual-verify-examples.sh [start_number]
# This script runs examples in order from 01 to 38, or starts from a specific number

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse command line arguments
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo -e "${BLUE}Manual Verification of Examples${NC}"
    echo "================================"
    echo
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0                    # Start from example 1"
    echo "  $0 [number]          # Start from specific example number"
    echo "  $0 -h, --help        # Show this help"
    echo
    echo -e "${YELLOW}Interactive Controls:${NC}"
    echo "  Enter                # Continue to next example"
    echo "  q, quit, exit        # Quit the verification"
    echo "  s, skip              # Skip to next example"
    echo
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0                   # Test all examples from 1-38"
    echo "  $0 10                # Start testing from example 10"
    echo "  $0 3                 # Start testing from example 3 (interpolation)"
    echo
    exit 0
fi

START_FROM=${1:-1}

# Validate start number
if ! [[ "$START_FROM" =~ ^[0-9]+$ ]] || [ "$START_FROM" -lt 1 ] || [ "$START_FROM" -gt 38 ]; then
    echo -e "${RED}❌ Invalid start number: $START_FROM${NC}"
    echo -e "${YELLOW}Please provide a number between 1 and 38${NC}"
    echo -e "${BLUE}Use $0 --help for usage information${NC}"
    exit 1
fi

echo -e "${BLUE}🔍 Manual Verification of Examples (Starting from #${START_FROM})${NC}"
echo "================================================================"
echo -e "${YELLOW}Tip: Use 'q' to quit, 's' to skip, Enter to continue${NC}"
echo

# Build fresh binary
echo -e "${BLUE}📦 Building fresh drun binary...${NC}"
go build -o drun ./cmd/drun
echo -e "${GREEN}✅ Build completed${NC}"
echo

# Function to run and display a command
run_example() {
    local example_num="$1"
    local file="$2"
    local task="$3"
    local params="$4"
    local description="$5"
    
    local filename=$(basename "$file")
    local command="./drun-cli -f $file"
    
    if [ -n "$task" ]; then
        command="$command \"$task\""
    fi
    
    if [ -n "$params" ]; then
        command="$command $params"
    fi
    
    echo -e "${CYAN}🧪 Example ${example_num}: ${filename}${NC}"
    echo -e "${YELLOW}Command: ${command}${NC}"
    if [ -n "$description" ]; then
        echo -e "${BLUE}Expected: ${description}${NC}"
    fi
    echo -e "${GREEN}Output:${NC}"
    
    # Run the command and indent output
    eval "$command" 2>&1 | sed 's/^/  /'
    
    echo
    echo -e "${YELLOW}Press Enter to continue (or 'q' to quit, 's' to skip to next)...${NC}"
    read -r input
    case "$input" in
        q|Q|quit|exit)
            echo -e "${BLUE}Exiting manual verification.${NC}"
            exit 0
            ;;
        s|S|skip)
            echo -e "${YELLOW}Skipping to next example...${NC}"
            ;;
        *)
            ;;
    esac
    echo
}

# Function to get the first task from a file
get_first_task() {
    local file="$1"
    local first_task=$(./drun-cli -f "$file" -l 2>/dev/null | grep -E "^  " | head -1 | sed 's/^  //' | sed 's/  *[A-Z].*//')
    echo "$first_task"
}

# Function to get suggested parameters for common task patterns
get_suggested_params() {
    local task="$1"
    local file="$2"
    
    case "$task" in
        *greet*|*hello*)
            echo "name=TestUser"
            ;;
        *deploy*)
            echo "environment=dev"
            ;;
        *backup*)
            echo "source_path=/tmp/test"
            ;;
        *build*)
            echo ""
            ;;
        *)
            # Try to detect required parameters from the file
            if grep -q "requires.*name" "$file" 2>/dev/null; then
                echo "name=TestUser"
            elif grep -q "requires.*environment" "$file" 2>/dev/null; then
                echo "environment=dev"
            else
                echo ""
            fi
            ;;
    esac
}

# Get all example files in numerical order
example_files=($(ls examples/[0-9][0-9]-*.drun 2>/dev/null | sort -V))

if [ ${#example_files[@]} -eq 0 ]; then
    echo -e "${RED}❌ No example files found in examples/ directory${NC}"
    exit 1
fi

echo -e "${BLUE}Found ${#example_files[@]} example files${NC}"
echo

# Track tested examples
tested_examples=()

# Process each example file starting from the specified number
for file in "${example_files[@]}"; do
    # Extract example number from filename
    filename=$(basename "$file")
    example_num=$(echo "$filename" | sed 's/^\([0-9][0-9]\)-.*/\1/' | sed 's/^0*//')
    
    # Skip if before start number
    if [ "$example_num" -lt "$START_FROM" ]; then
        continue
    fi
    
    # Get the first task
    first_task=$(get_first_task "$file")
    
    if [ -z "$first_task" ]; then
        echo -e "${YELLOW}⚠️  Skipping ${filename} - No tasks found${NC}"
        continue
    fi
    
    # Get suggested parameters
    suggested_params=$(get_suggested_params "$first_task" "$file")
    
    # Create description based on filename
    description=""
    case "$filename" in
        *hello-world*)
            description="Should show simple hello message"
            ;;
        *parameters*)
            description="Should demonstrate parameter usage"
            ;;
        *interpolation*)
            description="Should show variable interpolation working"
            ;;
        *docker*)
            description="Should demonstrate Docker operations"
            ;;
        *builtin*)
            description="Should show builtin functions working"
            ;;
        *shell*)
            description="Should demonstrate shell command execution"
            ;;
        *)
            description="Should execute without errors"
            ;;
    esac
    
    # Run the example
    run_example "$example_num" "$file" "$first_task" "$suggested_params" "$description"
    
    # Track this example as tested
    tested_examples+=("$example_num")
done

echo -e "${GREEN}🎉 Manual verification completed!${NC}"
if [ ${#tested_examples[@]} -gt 0 ]; then
    echo -e "${BLUE}Tested ${#tested_examples[@]} examples: ${tested_examples[*]}${NC}"
else
    echo -e "${YELLOW}No examples were tested (starting number $START_FROM may be too high)${NC}"
fi
