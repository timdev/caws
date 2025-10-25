package main

import (
	"os"
	"path/filepath"
)

// getXDGDataHome returns the XDG data directory
// Defaults to ~/.local/share if XDG_DATA_HOME is not set
func getXDGDataHome() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return xdgData
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}

	return filepath.Join(homeDir, ".local", "share")
}

// getXDGCacheHome returns the XDG cache directory
// Defaults to ~/.cache if XDG_CACHE_HOME is not set
func getXDGCacheHome() string {
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return xdgCache
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}

	return filepath.Join(homeDir, ".cache")
}
