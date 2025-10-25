#!/bin/bash

assert_success() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "✗ FAIL: Expected success, got exit code $exit_code"
        exit 1
    fi
}

assert_failure() {
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo "✗ FAIL: Expected failure, but command succeeded"
        exit 1
    fi
}

assert_contains() {
    local output="$1"
    local expected="$2"
    if ! echo "$output" | grep -q "$expected"; then
        echo "✗ FAIL: Output does not contain '$expected'"
        echo "Output was:"
        echo "$output"
        exit 1
    fi
}

assert_not_contains() {
    local output="$1"
    local unexpected="$2"
    if echo "$output" | grep -q "$unexpected"; then
        echo "✗ FAIL: Output should not contain '$unexpected'"
        echo "Output was:"
        echo "$output"
        exit 1
    fi
}

assert_file_exists() {
    local file="$1"
    if [ ! -f "$file" ]; then
        echo "✗ FAIL: File does not exist: $file"
        exit 1
    fi
}

assert_dir_exists() {
    local dir="$1"
    if [ ! -d "$dir" ]; then
        echo "✗ FAIL: Directory does not exist: $dir"
        exit 1
    fi
}
