package main

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"
)

// ProfileInfo contains information about an AWS profile
type ProfileInfo struct {
	Name      string
	Region    string
	MFASerial string
}

// VaultClient handles interactions with the encrypted vault
type VaultClient struct {
	vaultPath string
	password  string
}

// NewVaultClient creates a new vault client and prompts for password
func NewVaultClient() (*VaultClient, error) {
	vaultPath := getVaultPath()

	// Check if vault exists
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("vault not found at %s\nRun 'caws init' to create a new vault", vaultPath)
	}

	// Prompt for password
	fmt.Print("Enter vault password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // New line after password input
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	password := string(passwordBytes)

	// Verify password by attempting to decrypt
	if err := verifyPassword(vaultPath, password); err != nil {
		return nil, fmt.Errorf("incorrect password or corrupted vault")
	}

	return &VaultClient{
		vaultPath: vaultPath,
		password:  password,
	}, nil
}

// Close is a no-op for compatibility with gopass interface
func (v *VaultClient) Close() {
	// Nothing to close, but keep for interface compatibility
}

// GetCredentials retrieves AWS credentials for a profile
func (v *VaultClient) GetCredentials(profile string) (*AWSCredentials, error) {
	data, err := v.loadVault()
	if err != nil {
		return nil, err
	}

	profileData, exists := data.Profiles[profile]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found in vault", profile)
	}

	return &AWSCredentials{
		AccessKeyID:     profileData.AccessKey,
		SecretAccessKey: profileData.SecretKey,
		Region:          profileData.Region,
		MFASerial:       profileData.MFASerial,
	}, nil
}

// CreateCredentials stores AWS credentials for a profile
func (v *VaultClient) CreateCredentials(profile string, accessKey, secretKey, region, mfaSerial string) error {
	data, err := v.loadVault()
	if err != nil {
		return err
	}

	// Initialize profiles map if needed
	if data.Profiles == nil {
		data.Profiles = make(map[string]ProfileData)
	}

	// Add or update profile
	data.Profiles[profile] = ProfileData{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		MFASerial: mfaSerial,
	}

	return v.saveVault(data)
}

// ListProfiles returns all profiles stored in the vault
func (v *VaultClient) ListProfiles() ([]ProfileInfo, error) {
	data, err := v.loadVault()
	if err != nil {
		return nil, err
	}

	profiles := []ProfileInfo{}
	for name, profileData := range data.Profiles {
		profiles = append(profiles, ProfileInfo{
			Name:      name,
			Region:    profileData.Region,
			MFASerial: profileData.MFASerial,
		})
	}

	return profiles, nil
}

// RemoveProfile removes a profile from the vault
func (v *VaultClient) RemoveProfile(profile string) error {
	data, err := v.loadVault()
	if err != nil {
		return err
	}

	if _, exists := data.Profiles[profile]; !exists {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	delete(data.Profiles, profile)

	return v.saveVault(data)
}

// loadVault reads and decrypts the vault
func (v *VaultClient) loadVault() (*VaultData, error) {
	fileData, err := os.ReadFile(v.vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	var vaultFile VaultFile
	if err := json.Unmarshal(fileData, &vaultFile); err != nil {
		return nil, fmt.Errorf("failed to parse vault file: %w", err)
	}

	data, err := decryptVault(v.password, &vaultFile)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// saveVault encrypts and writes the vault
func (v *VaultClient) saveVault(data *VaultData) error {
	vaultFile, err := encryptVault(v.password, data)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	fileData, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault file: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tempPath := v.vaultPath + ".tmp"
	if err := os.WriteFile(tempPath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	if err := os.Rename(tempPath, v.vaultPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to replace vault file: %w", err)
	}

	return nil
}

// verifyPassword checks if a password can decrypt the vault
func verifyPassword(vaultPath, password string) error {
	fileData, err := os.ReadFile(vaultPath)
	if err != nil {
		return err
	}

	var vaultFile VaultFile
	if err := json.Unmarshal(fileData, &vaultFile); err != nil {
		return err
	}

	_, err = decryptVault(password, &vaultFile)
	return err
}

// getVaultPath returns the path to the vault file
func getVaultPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/caws-vault.enc"
	}
	return fmt.Sprintf("%s/.caws/vault.enc", homeDir)
}

// InitVault creates a new encrypted vault
func InitVault() error {
	vaultPath := getVaultPath()

	// Check if vault already exists
	if _, err := os.Stat(vaultPath); err == nil {
		return fmt.Errorf("vault already exists at %s", vaultPath)
	}

	// Prompt for password twice
	fmt.Print("Enter master password: ")
	password1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	fmt.Print("Confirm password: ")
	password2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if string(password1) != string(password2) {
		return fmt.Errorf("passwords do not match")
	}

	// Create vault directory
	vaultDir := fmt.Sprintf("%s/.caws", os.Getenv("HOME"))
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Create empty vault
	emptyData := &VaultData{
		Profiles: make(map[string]ProfileData),
	}

	vaultFile, err := encryptVault(string(password1), emptyData)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	fileData, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault: %w", err)
	}

	if err := os.WriteFile(vaultPath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	fmt.Printf("✓ Vault initialized at %s\n", vaultPath)
	return nil
}
