#!/usr/bin/env bash

set -euo pipefail

export KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT=true

# Find and execute all Python files in the current directory
for python_file in $(find . -maxdepth 1 -name "*.py"); do
    echo "Executing: $python_file"
    python3 "$python_file"
done
