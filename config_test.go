package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigSettings(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `[default]
region = us-west-2

[profile production]
region = us-east-1
mfa_serial = arn:aws:iam::123456789012:mfa/user

[profile staging]
region = eu-west-1

[profile minimal]
# This profile has no settings
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Set test mode to use our temp config
	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	tests := []struct {
		name           string
		profile        string
		wantRegion     string
		wantMFASerial  string
	}{
		{
			name:          "default profile",
			profile:       "default",
			wantRegion:    "us-west-2",
			wantMFASerial: "",
		},
		{
			name:          "production with MFA",
			profile:       "production",
			wantRegion:    "us-east-1",
			wantMFASerial: "arn:aws:iam::123456789012:mfa/user",
		},
		{
			name:          "staging without MFA",
			profile:       "staging",
			wantRegion:    "eu-west-1",
			wantMFASerial: "",
		},
		{
			name:          "minimal profile",
			profile:       "minimal",
			wantRegion:    "",
			wantMFASerial: "",
		},
		{
			name:          "nonexistent profile",
			profile:       "nonexistent",
			wantRegion:    "",
			wantMFASerial: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := getConfigSettings(tt.profile)
			if err != nil {
				t.Fatalf("getConfigSettings failed: %v", err)
			}

			if settings.Region != tt.wantRegion {
				t.Errorf("region: got %q, want %q", settings.Region, tt.wantRegion)
			}
			if settings.MFASerial != tt.wantMFASerial {
				t.Errorf("mfa_serial: got %q, want %q", settings.MFASerial, tt.wantMFASerial)
			}
		})
	}
}

func TestProfileExistsInConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `[default]
region = us-west-2

[profile production]
region = us-east-1
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	tests := []struct {
		name    string
		profile string
		want    bool
	}{
		{"default exists", "default", true},
		{"production exists", "production", true},
		{"nonexistent", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := profileExistsInConfig(tt.profile)
			if err != nil {
				t.Fatalf("profileExistsInConfig failed: %v", err)
			}
			if exists != tt.want {
				t.Errorf("got %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestProfileExistsInConfigNoFile(t *testing.T) {
	// Use a directory with no config file
	tmpDir := t.TempDir()

	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	exists, err := profileExistsInConfig("anyprofile")
	if err != nil {
		t.Fatalf("expected no error when config doesn't exist, got: %v", err)
	}
	if exists {
		t.Error("expected false when config file doesn't exist")
	}
}

func TestCreateConfigProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	// Create a profile in a new config
	if err := createConfigProfile("production"); err != nil {
		t.Fatalf("createConfigProfile failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}

	contentStr := string(content)
	if !containsLine(contentStr, "[profile production]") {
		t.Errorf("config should contain '[profile production]', got:\n%s", contentStr)
	}

	// Create another profile
	if err := createConfigProfile("staging"); err != nil {
		t.Fatalf("createConfigProfile second profile failed: %v", err)
	}

	// Verify both profiles exist
	content, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config after second profile: %v", err)
	}

	contentStr = string(content)
	if !containsLine(contentStr, "[profile production]") {
		t.Error("config should still contain '[profile production]'")
	}
	if !containsLine(contentStr, "[profile staging]") {
		t.Error("config should contain '[profile staging]'")
	}
}

func TestCreateConfigProfileDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	// Create default profile (special case)
	if err := createConfigProfile("default"); err != nil {
		t.Fatalf("createConfigProfile for default failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}

	contentStr := string(content)
	// Default profile should use [default], not [profile default]
	if !containsLine(contentStr, "[default]") {
		t.Errorf("config should contain '[default]', got:\n%s", contentStr)
	}
	if containsLine(contentStr, "[profile default]") {
		t.Error("config should not contain '[profile default]' for default profile")
	}
}

func TestGetConfigSettingsWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `# This is a comment
[profile production]
# Region comment
region = us-east-1
; Another comment style
mfa_serial = arn:aws:iam::123456789012:mfa/user
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	oldTestDir := os.Getenv("CAWS_TEST_DIR")
	os.Setenv("CAWS_TEST_DIR", tmpDir)
	defer os.Setenv("CAWS_TEST_DIR", oldTestDir)

	settings, err := getConfigSettings("production")
	if err != nil {
		t.Fatalf("getConfigSettings failed: %v", err)
	}

	if settings.Region != "us-east-1" {
		t.Errorf("region: got %q, want %q", settings.Region, "us-east-1")
	}
	if settings.MFASerial != "arn:aws:iam::123456789012:mfa/user" {
		t.Errorf("mfa_serial: got %q, want %q", settings.MFASerial, "arn:aws:iam::123456789012:mfa/user")
	}
}

// Helper function to check if a string contains a line
func containsLine(content, line string) bool {
	lines := splitLines(content)
	for _, l := range lines {
		if l == line {
			return true
		}
	}
	return false
}

// Helper function to split content into lines
func splitLines(content string) []string {
	var lines []string
	current := ""
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else if ch != '\r' {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
