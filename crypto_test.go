package main

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	password := "test-password-123"
	original := &VaultData{
		Profiles: map[string]ProfileData{
			"test-profile": {
				AccessKey: "AKIAIOSFODNN7EXAMPLE",
				SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			"prod-profile": {
				AccessKey: "AKIAI44QH8DHBEXAMPLE",
				SecretKey: "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY",
			},
		},
	}

	// Encrypt
	encrypted, err := encryptVault(password, original)
	if err != nil {
		t.Fatalf("encryptVault failed: %v", err)
	}

	// Verify encrypted structure
	if encrypted.Version != vaultVersion {
		t.Errorf("expected version %d, got %d", vaultVersion, encrypted.Version)
	}
	if encrypted.Salt == "" {
		t.Error("salt should not be empty")
	}
	if encrypted.Nonce == "" {
		t.Error("nonce should not be empty")
	}
	if encrypted.Data == "" {
		t.Error("encrypted data should not be empty")
	}

	// Decrypt
	decrypted, err := decryptVault(password, encrypted)
	if err != nil {
		t.Fatalf("decryptVault failed: %v", err)
	}

	// Verify round-trip
	if len(decrypted.Profiles) != len(original.Profiles) {
		t.Errorf("expected %d profiles, got %d", len(original.Profiles), len(decrypted.Profiles))
	}

	for name, origProfile := range original.Profiles {
		decProfile, exists := decrypted.Profiles[name]
		if !exists {
			t.Errorf("profile %s missing after round-trip", name)
			continue
		}
		if decProfile.AccessKey != origProfile.AccessKey {
			t.Errorf("profile %s: access key mismatch", name)
		}
		if decProfile.SecretKey != origProfile.SecretKey {
			t.Errorf("profile %s: secret key mismatch", name)
		}
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	password := "correct-password"
	wrongPassword := "wrong-password"

	data := &VaultData{
		Profiles: map[string]ProfileData{
			"test": {
				AccessKey: "AKIAIOSFODNN7EXAMPLE",
				SecretKey: "secretKey123",
			},
		},
	}

	encrypted, err := encryptVault(password, data)
	if err != nil {
		t.Fatalf("encryptVault failed: %v", err)
	}

	// Try to decrypt with wrong password
	_, err = decryptVault(wrongPassword, encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong password")
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("expected 'decryption failed' error, got: %v", err)
	}
}

func TestEncryptEmptyVault(t *testing.T) {
	password := "test-password"
	emptyData := &VaultData{
		Profiles: make(map[string]ProfileData),
	}

	encrypted, err := encryptVault(password, emptyData)
	if err != nil {
		t.Fatalf("encryptVault failed on empty vault: %v", err)
	}

	decrypted, err := decryptVault(password, encrypted)
	if err != nil {
		t.Fatalf("decryptVault failed: %v", err)
	}

	if len(decrypted.Profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(decrypted.Profiles))
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	password := "test-password"
	salt := []byte("01234567890123456789012345678901") // Exactly 32 bytes

	if len(salt) != saltSize {
		t.Fatalf("test salt should be %d bytes, got %d", saltSize, len(salt))
	}

	// Derive key twice with same inputs
	key1 := deriveKey(password, salt)
	key2 := deriveKey(password, salt)

	// Should be identical
	if len(key1) != keySize {
		t.Errorf("expected key size %d, got %d", keySize, len(key1))
	}
	if len(key2) != keySize {
		t.Errorf("expected key size %d, got %d", keySize, len(key2))
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("derived keys should be identical for same inputs")
			break
		}
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	password := "test-password"
	salt1 := []byte("salt1111111111111111111111111111")
	salt2 := []byte("salt2222222222222222222222222222")

	key1 := deriveKey(password, salt1)
	key2 := deriveKey(password, salt2)

	// Keys should be different
	identical := true
	for i := range key1 {
		if key1[i] != key2[i] {
			identical = false
			break
		}
	}

	if identical {
		t.Error("different salts should produce different keys")
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	password := "test-password"
	data := &VaultData{
		Profiles: map[string]ProfileData{
			"test": {
				AccessKey: "AKIAIOSFODNN7EXAMPLE",
				SecretKey: "secretKey123",
			},
		},
	}

	// Encrypt same data twice
	encrypted1, err := encryptVault(password, data)
	if err != nil {
		t.Fatalf("encryptVault 1 failed: %v", err)
	}

	encrypted2, err := encryptVault(password, data)
	if err != nil {
		t.Fatalf("encryptVault 2 failed: %v", err)
	}

	// Salt and nonce should be different (randomly generated)
	if encrypted1.Salt == encrypted2.Salt {
		t.Error("encrypted vaults should have different salts")
	}
	if encrypted1.Nonce == encrypted2.Nonce {
		t.Error("encrypted vaults should have different nonces")
	}
	if encrypted1.Data == encrypted2.Data {
		t.Error("encrypted data should be different due to different salts/nonces")
	}

	// But both should decrypt correctly
	decrypted1, err := decryptVault(password, encrypted1)
	if err != nil {
		t.Fatalf("decryptVault 1 failed: %v", err)
	}
	decrypted2, err := decryptVault(password, encrypted2)
	if err != nil {
		t.Fatalf("decryptVault 2 failed: %v", err)
	}

	// And produce same plaintext
	if decrypted1.Profiles["test"].AccessKey != decrypted2.Profiles["test"].AccessKey {
		t.Error("decrypted data should be identical")
	}
}

func TestUnsupportedVaultVersion(t *testing.T) {
	password := "test-password"
	data := &VaultData{
		Profiles: map[string]ProfileData{
			"test": {
				AccessKey: "AKIAIOSFODNN7EXAMPLE",
				SecretKey: "secretKey123",
			},
		},
	}

	encrypted, err := encryptVault(password, data)
	if err != nil {
		t.Fatalf("encryptVault failed: %v", err)
	}

	// Modify version to unsupported value
	encrypted.Version = 999

	// Should fail on decrypt
	_, err = decryptVault(password, encrypted)
	if err == nil {
		t.Error("expected error for unsupported vault version")
	}
	if !strings.Contains(err.Error(), "unsupported vault version") {
		t.Errorf("expected 'unsupported vault version' error, got: %v", err)
	}
}
