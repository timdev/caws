package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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
	Type            string    `json:"Type"` // "session" or "federation"
}

// AssumeRole calls AWS STS to get temporary credentials
func AssumeRole(creds *AWSCredentials, duration int32, mfaCode string) (*STSCredentials, error) {
	ctx := context.Background()

	// Create AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				"",
			),
		),
		config.WithRegion(creds.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create STS client
	client := sts.NewFromConfig(cfg)

	// Build GetSessionToken input
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(duration),
	}

	// Add MFA if required
	if creds.MFASerial != "" {
		if mfaCode == "" {
			return nil, fmt.Errorf("MFA code required but not provided")
		}
		input.SerialNumber = aws.String(creds.MFASerial)
		input.TokenCode = aws.String(mfaCode)
	}

	// Call STS GetSessionToken
	result, err := client.GetSessionToken(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get session token: %w", err)
	}

	// Convert to our STSCredentials format
	stsCreds := &STSCredentials{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          creds.Region,
		Type:            "session",
	}

	return stsCreds, nil
}

// GetFederationToken calls AWS STS to get federation token (for console login)
func GetFederationToken(creds *AWSCredentials, duration int32, name string) (*STSCredentials, error) {
	ctx := context.Background()

	// Create AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				"",
			),
		),
		config.WithRegion(creds.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create STS client
	client := sts.NewFromConfig(cfg)

	// Build GetFederationToken input
	input := &sts.GetFederationTokenInput{
		Name:            aws.String(name),
		DurationSeconds: aws.Int32(duration),
		// Policy is optional - omitting it means the token has same permissions as the user
	}

	// Call STS GetFederationToken
	result, err := client.GetFederationToken(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get federation token: %w", err)
	}

	// Convert to our STSCredentials format
	stsCreds := &STSCredentials{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          creds.Region,
		Type:            "federation",
	}

	return stsCreds, nil
}

// GetConsoleURL generates an AWS Console federation login URL
func GetConsoleURL(creds *STSCredentials, region string) (string, error) {
	// Build session JSON for federation
	session := map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	// Request signin token from AWS federation endpoint
	federationURL := "https://signin.aws.amazon.com/federation"
	params := url.Values{}
	params.Add("Action", "getSigninToken")
	params.Add("Session", string(sessionJSON))

	resp, err := http.Get(federationURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to get signin token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("federation endpoint returned status %d", resp.StatusCode)
	}

	var tokenResp struct {
		SigninToken string `json:"SigninToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse signin token response: %w", err)
	}

	// Construct console login URL
	loginParams := url.Values{}
	loginParams.Add("Action", "login")
	loginParams.Add("Destination", "https://console.aws.amazon.com/")
	loginParams.Add("SigninToken", tokenResp.SigninToken)

	return federationURL + "?" + loginParams.Encode(), nil
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
	// Check for test mode
	if testDir := os.Getenv("CAWS_TEST_DIR"); testDir != "" {
		return filepath.Join(testDir, "cache")
	}

	// XDG-compliant path: $XDG_CACHE_HOME/caws
	cacheHome := getXDGCacheHome()
	return filepath.Join(cacheHome, "caws")
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
		"AWS_PROFILE=",               // Filter this out - we don't set it
		"AWS_VAULT=",                 // Filter old value
		"AWS_CREDENTIAL_EXPIRATION=", // Filter old value
	}

	for _, prefix := range awsPrefixes {
		if len(envVar) >= len(prefix) && envVar[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
