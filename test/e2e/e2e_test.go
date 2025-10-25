package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package-level variable for the caws binary path
var cawsBin string

// TestMain builds the caws binary once before running all tests
func TestMain(m *testing.M) {
	// Build binary once for all tests
	tmpDir, err := os.MkdirTemp("", "caws-e2e-bin-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	cawsBin = filepath.Join(tmpDir, "caws")
	cmd := exec.Command("go", "build", "-o", cawsBin, "../../")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build caws: " + err.Error() + "\nOutput: " + string(output))
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// TestEnv encapsulates a test environment with isolated temp directory
type TestEnv struct {
	Dir       string   // Temp directory (auto-cleanup via t.TempDir())
	CawsBin   string   // Path to caws binary
	Env       []string // Environment variables
	AccessKey string   // Test AWS access key
	SecretKey string   // Test AWS secret key
	Mock      bool     // Using mock STS?
	t         *testing.T
}

// newTestEnv creates a new isolated test environment
func newTestEnv(t *testing.T) *TestEnv {
	tmpDir := t.TempDir() // Auto-cleanup on test end

	accessKey, secretKey, mock := getTestCredentials()

	env := []string{
		"CAWS_TEST_DIR=" + tmpDir,
		"CAWS_PASSWORD=testpass",
		"CAWS_AUTO_CONFIRM=yes",
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"), // Needed for potential home dir lookups
	}

	if mock {
		env = append(env, "CAWS_MOCK_STS=1")
	}

	return &TestEnv{
		Dir:       tmpDir,
		CawsBin:   cawsBin,
		Env:       env,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Mock:      mock,
		t:         t,
	}
}

// getTestCredentials returns test credentials (real or mock)
func getTestCredentials() (accessKey, secretKey string, mock bool) {
	accessKey = os.Getenv("CAWS_TEST_AWS_ACCESS_KEY")
	secretKey = os.Getenv("CAWS_TEST_AWS_SECRET_KEY")

	if accessKey != "" && secretKey != "" {
		return accessKey, secretKey, false // Real AWS
	}

	// Mock mode - use valid format credentials
	return "AKIAIOSFODNN7EXAMPLE",
		"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		true
}

// Run executes caws with given arguments and returns output and error
func (e *TestEnv) Run(args ...string) (string, error) {
	cmd := exec.Command(e.CawsBin, args...)
	cmd.Env = e.Env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// MustRun executes caws and fails test if it returns an error
func (e *TestEnv) MustRun(args ...string) string {
	output, err := e.Run(args...)
	require.NoError(e.t, err, "command failed: caws %s\nOutput: %s", strings.Join(args, " "), output)
	return output
}

// RunExpectError executes caws expecting an error
func (e *TestEnv) RunExpectError(args ...string) string {
	output, err := e.Run(args...)
	require.Error(e.t, err, "expected command to fail: caws %s", strings.Join(args, " "))
	return output
}

// RunWithStdin executes caws with stdin input
func (e *TestEnv) RunWithStdin(stdin string, args ...string) (string, error) {
	cmd := exec.Command(e.CawsBin, args...)
	cmd.Env = e.Env
	cmd.Stdin = strings.NewReader(stdin)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// MustRunWithStdin executes caws with stdin and fails test on error
func (e *TestEnv) MustRunWithStdin(stdin string, args ...string) string {
	output, err := e.RunWithStdin(stdin, args...)
	require.NoError(e.t, err, "command failed: caws %s\nOutput: %s", strings.Join(args, " "), output)
	return output
}

// Command returns an *exec.Cmd for advanced control (e.g., background processes)
func (e *TestEnv) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(e.CawsBin, args...)
	cmd.Env = e.Env
	return cmd
}

// SetupVault initializes a vault
func (e *TestEnv) SetupVault() {
	e.MustRun("init")
}

// SetupProfile adds a profile to the vault using the add command with test env vars
func (e *TestEnv) SetupProfile(name string) {
	// Use test environment variables to provide credentials non-interactively
	testEnv := append(e.Env,
		"CAWS_TEST_ACCESS_KEY="+e.AccessKey,
		"CAWS_TEST_SECRET_KEY="+e.SecretKey,
	)

	cmd := exec.Command(e.CawsBin, "add", name)
	cmd.Env = testEnv
	output, err := cmd.CombinedOutput()
	require.NoError(e.t, err, "failed to add profile: %s", string(output))
}

// CreateConfig creates an AWS config file with given content
func (e *TestEnv) CreateConfig(content string) {
	configPath := filepath.Join(e.Dir, "config")
	require.NoError(e.t, os.WriteFile(configPath, []byte(content), 0600))
}

// CreateConfigProfile creates a profile in AWS config
func (e *TestEnv) CreateConfigProfile(name, region, mfaSerial string) {
	sectionName := "[profile " + name + "]"
	if name == "default" {
		sectionName = "[default]"
	}

	content := sectionName + "\n"
	if region != "" {
		content += "region = " + region + "\n"
	}
	if mfaSerial != "" {
		content += "mfa_serial = " + mfaSerial + "\n"
	}

	e.CreateConfig(content)
}

// VaultPath returns the path to the vault file
func (e *TestEnv) VaultPath() string {
	return filepath.Join(e.Dir, "vault.enc")
}

// CachePath returns the path to a profile's cache file
func (e *TestEnv) CachePath(profile string) string {
	return filepath.Join(e.Dir, "cache", profile+".json")
}

// LockPath returns the path to the vault lock file
func (e *TestEnv) LockPath() string {
	return filepath.Join(e.Dir, "vault.enc.lock")
}

// VaultExists checks if vault file exists
func (e *TestEnv) VaultExists() bool {
	_, err := os.Stat(e.VaultPath())
	return err == nil
}

// CacheExists checks if a profile's cache file exists
func (e *TestEnv) CacheExists(profile string) bool {
	_, err := os.Stat(e.CachePath(profile))
	return err == nil
}

// LockFileExists checks if lock file exists
func (e *TestEnv) LockFileExists() bool {
	_, err := os.Stat(e.LockPath())
	return err == nil
}

// ReadCache reads and parses a profile's cache file
func (e *TestEnv) ReadCache(profile string) map[string]interface{} {
	data, err := os.ReadFile(e.CachePath(profile))
	require.NoError(e.t, err, "failed to read cache file")

	var cache map[string]interface{}
	require.NoError(e.t, json.Unmarshal(data, &cache), "failed to parse cache JSON")
	return cache
}

// AssertVaultPermissions verifies vault has 0600 permissions
func (e *TestEnv) AssertVaultPermissions() {
	info, err := os.Stat(e.VaultPath())
	require.NoError(e.t, err)
	assert.Equal(e.t, os.FileMode(0600), info.Mode().Perm(), "vault permissions should be 0600")
}
