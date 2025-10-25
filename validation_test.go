package main

import (
	"strings"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid simple name", "production", false},
		{"valid with dash", "prod-aws", false},
		{"valid with underscore", "prod_aws", false},
		{"valid with number", "prod123", false},
		{"empty name", "", true},
		{"dot", "prod.test", true},
		{"forward slash", "prod/test", true},
		{"backslash", "prod\\test", true},
		{"dot dot", "..", true},
		{"newline", "prod\ntest", true},
		{"tab", "prod\ttest", true},
		{"carriage return", "prod\rtest", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfileName(tt.input)
			if tt.wantError && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error for input %q, got: %v", tt.input, err)
			}
		})
	}
}

func TestValidateAccessKey(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid AKIA key", "AKIAIOSFODNN7EXAMPLE", false},
		{"valid ASIA key", "ASIATESTACCESSKEY123", false},
		{"empty key", "", true},
		{"too short", "AKIA123", true},
		{"too long", "AKIAIOSFODNN7EXAMPLETOOLONG", true},
		{"wrong prefix", "XXIAIOSFODNN7EXAMPLE", true},
		{"lowercase", "akiaiosfodnn7example", true},
		{"exactly 20 chars but wrong prefix", "12345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAccessKey(tt.input)
			if tt.wantError && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error for input %q, got: %v", tt.input, err)
			}
		})
	}
}

func TestValidateAccessKeyErrorMessages(t *testing.T) {
	// Test empty key error message
	err := validateAccessKey("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected 'cannot be empty' in error, got: %v", err)
	}

	// Test wrong length error message
	err = validateAccessKey("AKIA123")
	if err == nil {
		t.Fatal("expected error for short key")
	}
	if !strings.Contains(err.Error(), "20 characters") {
		t.Errorf("expected '20 characters' in error, got: %v", err)
	}

	// Test wrong prefix error message
	err = validateAccessKey("XXIA1234567890123456")
	if err == nil {
		t.Fatal("expected error for wrong prefix")
	}
	if !strings.Contains(err.Error(), "should start with") {
		t.Errorf("expected 'should start with' in error, got: %v", err)
	}
}
