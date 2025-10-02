#!/bin/bash

# Test script for drun-cli completion functionality
echo "Testing drun-cli completion functionality..."

# Source the completion script
source /tmp/drun_completion.bash

# Test completion function directly
echo "Testing recipe completion..."
COMP_WORDS=("drun-cli" "")
COMP_CWORD=1
COMP_LINE="drun-cli "
COMP_POINT=8

# Call the completion function
_drun-cli

echo "Completions available: ${COMPREPLY[@]}"

# Test with partial recipe name
echo -e "\nTesting partial recipe completion..."
COMP_WORDS=("drun-cli" "rel")
COMP_CWORD=1
COMP_LINE="drun-cli rel"
COMP_POINT=11

# Call the completion function
_drun-cli

echo "Completions for 'rel': ${COMPREPLY[@]}"
