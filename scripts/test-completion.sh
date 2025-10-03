#!/bin/bash

# Test script for xdrun completion functionality
echo "Testing xdrun completion functionality..."

# Source the completion script
source /tmp/xdrun_completion.bash

# Test completion function directly
echo "Testing recipe completion..."
COMP_WORDS=("xdrun" "")
COMP_CWORD=1
COMP_LINE="xdrun "
COMP_POINT=8

# Call the completion function
_xdrun

echo "Completions available: ${COMPREPLY[@]}"

# Test with partial recipe name
echo -e "\nTesting partial recipe completion..."
COMP_WORDS=("xdrun" "rel")
COMP_CWORD=1
COMP_LINE="xdrun rel"
COMP_POINT=11

# Call the completion function
_xdrun

echo "Completions for 'rel': ${COMPREPLY[@]}"
