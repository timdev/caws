package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// VaultFile represents the encrypted vault file structure
type VaultFile struct {
	Version int    `json:"version"`
	Salt    string `json:"salt"`    // base64-encoded
	Nonce   string `json:"nonce"`   // base64-encoded
	Data    string `json:"data"`    // base64-encoded encrypted JSON
}

// VaultData represents the decrypted vault contents
type VaultData struct {
	Profiles map[string]ProfileData `json:"profiles"`
}

// ProfileData represents stored AWS credentials for a profile
type ProfileData struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region,omitempty"`
	MFASerial string `json:"mfa_serial,omitempty"`
}

const (
	vaultVersion = 1
	saltSize     = 32
	nonceSize    = 12
	keySize      = 32 // AES-256

	// Argon2id parameters (balanced security/performance)
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
)

// deriveKey derives an encryption key from a password using Argon2id
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keySize)
}

// encryptVault encrypts vault data with a password
func encryptVault(password string, data *VaultData) (*VaultFile, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key from password
	key := deriveKey(password, salt)

	// Marshal vault data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vault data: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &VaultFile{
		Version: vaultVersion,
		Salt:    base64.StdEncoding.EncodeToString(salt),
		Nonce:   base64.StdEncoding.EncodeToString(nonce),
		Data:    base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// decryptVault decrypts a vault file with a password
func decryptVault(password string, vaultFile *VaultFile) (*VaultData, error) {
	// Check version
	if vaultFile.Version != vaultVersion {
		return nil, fmt.Errorf("unsupported vault version: %d", vaultFile.Version)
	}

	// Decode base64 fields
	salt, err := base64.StdEncoding.DecodeString(vaultFile.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(vaultFile.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(vaultFile.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	// Derive encryption key from password
	key := deriveKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?): %w", err)
	}

	// Unmarshal JSON
	var data VaultData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vault data: %w", err)
	}

	return &data, nil
}
