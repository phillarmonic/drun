#!/bin/bash

# Test script for drun-cli completion functionality
echo "Testing drun-cli completion functionality..."

# Source the completion script
source /tmp/drun_completion.bash

# Test completion function directly
echo "Testing recipe completion..."
COMP_WORDS=("drun" "")
COMP_CWORD=1
COMP_LINE="drun "
COMP_POINT=5

# Call the completion function
_drun

echo "Completions available: ${COMPREPLY[@]}"

# Test with partial recipe name
echo -e "\nTesting partial recipe completion..."
COMP_WORDS=("drun" "rel")
COMP_CWORD=1
COMP_LINE="drun rel"
COMP_POINT=8

# Call the completion function
_drun

echo "Completions for 'rel': ${COMPREPLY[@]}"
