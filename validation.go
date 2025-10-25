package main

import (
	"fmt"
	"strings"
)

// validateProfileName validates that a profile name is safe to use
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	// Prevent path traversal attacks
	if strings.ContainsAny(name, "./\\") {
		return fmt.Errorf("invalid profile name: %s (cannot contain ., /, or \\)", name)
	}

	// Prevent other unsafe characters
	if strings.ContainsAny(name, "\n\r\t") {
		return fmt.Errorf("invalid profile name: %s (cannot contain whitespace characters)", name)
	}

	return nil
}

// validateAccessKey validates AWS access key format
func validateAccessKey(key string) error {
	if key == "" {
		return fmt.Errorf("access key cannot be empty")
	}

	// AWS access keys should start with AKIA (long-term) or ASIA (temporary)
	if !strings.HasPrefix(key, "AKIA") && !strings.HasPrefix(key, "ASIA") {
		return fmt.Errorf("access key should start with AKIA (long-term) or ASIA (temporary), got: %s", key[:4])
	}

	// AWS access keys are 20 characters long
	if len(key) != 20 {
		return fmt.Errorf("access key should be 20 characters long, got: %d", len(key))
	}

	return nil
}
