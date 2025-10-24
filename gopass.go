package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gopasspw/gopass/pkg/gopass"
	"github.com/gopasspw/gopass/pkg/gopass/api"
	"github.com/gopasspw/gopass/pkg/gopass/secrets"
)

const secretPrefix = "aws/"

// GopassClient handles interactions with gopass
type GopassClient struct {
	store gopass.Store
	ctx   context.Context
}

// NewGopassClient creates a new gopass client
func NewGopassClient() (*GopassClient, error) {
	ctx := context.Background()
	store, err := api.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open gopass store: %w\nHave you run 'gopass init'?", err)
	}

	return &GopassClient{
		store: store,
		ctx:   ctx,
	}, nil
}

// Close closes the gopass store
func (g *GopassClient) Close() {
	if g.store != nil {
		g.store.Close(g.ctx)
	}
}

// GetCredentials retrieves AWS credentials from gopass
func (g *GopassClient) GetCredentials(profile string) (*AWSCredentials, error) {
	secretName := secretPrefix + profile
	secret, err := g.store.Get(g.ctx, secretName, "latest")
	if err != nil {
		return nil, fmt.Errorf("profile '%s' not found in gopass", profile)
	}

	// Extract fields
	accessKey, _ := secret.Get("access_key")
	secretKey, _ := secret.Get("secret_key")
	region, _ := secret.Get("region")
	mfaSerial, _ := secret.Get("mfa_serial")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("invalid credentials for profile '%s': missing access_key or secret_key", profile)
	}

	return &AWSCredentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Region:          region,
		MFASerial:       mfaSerial,
	}, nil
}

// CreateCredentials stores AWS credentials in gopass
func (g *GopassClient) CreateCredentials(profile string, accessKey, secretKey, region, mfaSerial string) error {
	secretName := secretPrefix + profile

	// Create new secret
	sec := secrets.New()
	sec.SetPassword("AWS credentials for " + profile)
	sec.Set("access_key", accessKey)
	sec.Set("secret_key", secretKey)
	if region != "" {
		sec.Set("region", region)
	}
	if mfaSerial != "" {
		sec.Set("mfa_serial", mfaSerial)
	}

	err := g.store.Set(g.ctx, secretName, sec)
	if err != nil {
		return fmt.Errorf("failed to create profile in gopass: %w", err)
	}

	return nil
}

// ListProfiles returns all AWS profiles stored in gopass
func (g *GopassClient) ListProfiles() ([]ProfileInfo, error) {
	allSecrets, err := g.store.List(g.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	profiles := []ProfileInfo{}
	for _, name := range allSecrets {
		if !strings.HasPrefix(name, secretPrefix) {
			continue
		}

		profile := strings.TrimPrefix(name, secretPrefix)

		// Fetch the secret to get region and MFA info
		secret, err := g.store.Get(g.ctx, name, "latest")
		if err != nil {
			continue // Skip secrets we can't read
		}

		region, _ := secret.Get("region")
		mfaSerial, _ := secret.Get("mfa_serial")

		profiles = append(profiles, ProfileInfo{
			Name:      profile,
			Region:    region,
			MFASerial: mfaSerial,
		})
	}

	return profiles, nil
}

// RemoveProfile removes an AWS profile from gopass
func (g *GopassClient) RemoveProfile(profile string) error {
	secretName := secretPrefix + profile

	// Check if it exists first
	_, err := g.store.Get(g.ctx, secretName, "latest")
	if err != nil {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	err = g.store.Remove(g.ctx, secretName)
	if err != nil {
		return fmt.Errorf("failed to remove profile: %w", err)
	}

	return nil
}

// ProfileInfo contains information about an AWS profile
type ProfileInfo struct {
	Name      string
	Region    string
	MFASerial string
}
