package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getAWSConfigPath returns the path to ~/.aws/config
func getAWSConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".aws", "config"), nil
}

// profileExistsInConfig checks if a profile exists in ~/.aws/config
func profileExistsInConfig(profile string) (bool, error) {
	configPath, err := getAWSConfigPath()
	if err != nil {
		return false, err
	}

	// If config file doesn't exist, profile doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Look for [profile <name>] or [default] (if profile is "default")
	targetSection := fmt.Sprintf("[profile %s]", profile)
	if profile == "default" {
		targetSection = "[default]"
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == targetSection {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// createConfigProfile creates a minimal profile section in ~/.aws/config
func createConfigProfile(profile string) error {
	configPath, err := getAWSConfigPath()
	if err != nil {
		return err
	}

	// Create ~/.aws directory if it doesn't exist
	awsDir := filepath.Dir(configPath)
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return fmt.Errorf("failed to create ~/.aws directory: %w", err)
	}

	// Determine section name
	sectionName := fmt.Sprintf("[profile %s]", profile)
	if profile == "default" {
		sectionName = "[default]"
	}

	// Open file for append (create if doesn't exist)
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Check if file is empty or ends with newline
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	content := ""
	if stat.Size() > 0 {
		// File has content, ensure it ends with newline
		content = "\n"
	}

	// Append profile section
	content += fmt.Sprintf("%s\n", sectionName)

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	return nil
}

// ConfigSettings represents settings read from ~/.aws/config
type ConfigSettings struct {
	Region    string
	MFASerial string
}

// getConfigSettings reads region and mfa_serial from ~/.aws/config for a profile
func getConfigSettings(profile string) (*ConfigSettings, error) {
	configPath, err := getAWSConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return empty settings (no error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &ConfigSettings{}, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Look for the profile section
	targetSection := fmt.Sprintf("[profile %s]", profile)
	if profile == "default" {
		targetSection = "[default]"
	}

	settings := &ConfigSettings{}
	inTargetSection := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check if we entered our target section
		if line == targetSection {
			inTargetSection = true
			continue
		}

		// Check if we entered a different section (stop reading)
		if strings.HasPrefix(line, "[") {
			inTargetSection = false
			continue
		}

		// If we're in our target section, parse key-value pairs
		if inTargetSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "region":
					settings.Region = value
				case "mfa_serial":
					settings.MFASerial = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}
