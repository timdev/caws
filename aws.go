package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// AWSCredentials represents AWS access credentials
type AWSCredentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token,omitempty"`
	Expiration      time.Time `json:"expiration,omitempty"`
	Region          string    `json:"region,omitempty"`
	MFASerial       string    `json:"mfa_serial,omitempty"`
}

// STSCredentials represents temporary STS credentials
type STSCredentials struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
	Region          string    `json:"Region,omitempty"`
}

// AssumeRole calls AWS STS to get temporary credentials
func AssumeRole(creds *AWSCredentials, duration int32, mfaCode string) (*STSCredentials, error) {
	// Set up environment with base credentials
	env := os.Environ()
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey))
	if creds.Region != "" {
		env = append(env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", creds.Region))
	}

	// Build AWS CLI command
	args := []string{
		"sts", "get-session-token",
		"--duration-seconds", fmt.Sprintf("%d", duration),
		"--output", "json",
	}

	// Add MFA if required
	if creds.MFASerial != "" {
		if mfaCode == "" {
			return nil, fmt.Errorf("MFA code required but not provided")
		}
		args = append(args, "--serial-number", creds.MFASerial)
		args = append(args, "--token-code", mfaCode)
	}

	cmd := exec.Command("aws", args...)
	cmd.Env = env
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get session token: %w", err)
	}

	var response struct {
		Credentials STSCredentials `json:"Credentials"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse STS response: %w", err)
	}

	// Add region to credentials for caching
	response.Credentials.Region = creds.Region

	return &response.Credentials, nil
}

// GetCachedCredentials retrieves cached credentials if still valid
func GetCachedCredentials(profile string) (*STSCredentials, error) {
	cacheDir := getCacheDir()
	cachePath := fmt.Sprintf("%s/%s.json", cacheDir, profile)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var creds STSCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	// Check if expired (with 5 minute buffer)
	if time.Now().Add(5 * time.Minute).After(creds.Expiration) {
		return nil, fmt.Errorf("cached credentials expired")
	}

	return &creds, nil
}

// CacheCredentials saves credentials to cache
func CacheCredentials(profile string, creds *STSCredentials) error {
	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := fmt.Sprintf("%s/%s.json", cacheDir, profile)

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// getCacheDir returns the cache directory path
func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/caws-cache"
	}
	return fmt.Sprintf("%s/.caws/cache", homeDir)
}

// SetEnvVars sets AWS environment variables
func SetEnvVars(profile string, creds *STSCredentials, region string) []string {
	env := os.Environ()

	// Remove existing AWS env vars (including AWS_PROFILE)
	filtered := []string{}
	for _, e := range env {
		if !isAWSEnvVar(e) {
			filtered = append(filtered, e)
		}
	}

	// Add new credentials
	filtered = append(filtered, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID))
	filtered = append(filtered, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey))
	filtered = append(filtered, fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken))

	// Add AWS_VAULT for shell prompt integration (matches aws-vault behavior)
	filtered = append(filtered, fmt.Sprintf("AWS_VAULT=%s", profile))

	// Add credential expiration timestamp
	filtered = append(filtered, fmt.Sprintf("AWS_CREDENTIAL_EXPIRATION=%s", creds.Expiration.Format(time.RFC3339)))

	if region != "" {
		filtered = append(filtered, fmt.Sprintf("AWS_DEFAULT_REGION=%s", region))
		filtered = append(filtered, fmt.Sprintf("AWS_REGION=%s", region))
	}

	return filtered
}

// isAWSEnvVar checks if an environment variable is AWS-related
func isAWSEnvVar(envVar string) bool {
	awsPrefixes := []string{
		"AWS_ACCESS_KEY_ID=",
		"AWS_SECRET_ACCESS_KEY=",
		"AWS_SESSION_TOKEN=",
		"AWS_SECURITY_TOKEN=",
		"AWS_DEFAULT_REGION=",
		"AWS_REGION=",
		"AWS_PROFILE=", // Filter this out - we don't set it
		"AWS_VAULT=",    // Filter old value
		"AWS_CREDENTIAL_EXPIRATION=", // Filter old value
	}

	for _, prefix := range awsPrefixes {
		if len(envVar) >= len(prefix) && envVar[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
