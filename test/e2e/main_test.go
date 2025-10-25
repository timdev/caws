package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteWorkflow tests the full lifecycle: init → add → list → exec → remove
func TestCompleteWorkflow(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// 1. Init vault
	output := env.MustRun("init")
	assert.Contains(t, output, "Vault initialized")
	assert.True(t, env.VaultExists(), "vault file should exist")
	env.AssertVaultPermissions()

	// 2. Create AWS config with region (no MFA for easier testing)
	env.CreateConfigProfile("production", "us-east-1", "")

	// 3. Add profile
	// Use test environment variables for non-interactive credential input
	testEnv := append(env.Env,
		"CAWS_TEST_ACCESS_KEY="+env.AccessKey,
		"CAWS_TEST_SECRET_KEY="+env.SecretKey,
	)
	cmd := env.Command("add", "production")
	cmd.Env = testEnv
	outputBytes, err := cmd.CombinedOutput()
	require.NoError(t, err, "add failed: %s", string(outputBytes))
	assert.Contains(t, string(outputBytes), "Successfully added profile 'production'")

	// 4. List profiles - verify profile shown with metadata
	output = env.MustRun("list")
	assert.Contains(t, output, "production")
	assert.Contains(t, output, "us-east-1")

	// 5. Exec command - verify all env vars set
	output = env.MustRun("exec", "production", "--", "env")
	assert.Contains(t, output, "AWS_ACCESS_KEY_ID=")
	assert.Contains(t, output, "AWS_SECRET_ACCESS_KEY=")
	assert.Contains(t, output, "AWS_SESSION_TOKEN=")
	assert.Contains(t, output, "AWS_VAULT=production")
	assert.Contains(t, output, "AWS_REGION=us-east-1")
	assert.Contains(t, output, "AWS_DEFAULT_REGION=us-east-1")
	assert.Contains(t, output, "AWS_CREDENTIAL_EXPIRATION=")

	// Verify AWS_PROFILE is NOT set
	assert.NotContains(t, output, "AWS_PROFILE=")

	// 6. Remove profile
	output = env.MustRunWithStdin("yes\n", "remove", "production")
	assert.Contains(t, output, "Successfully removed profile 'production'")

	// 7. List again - verify profile gone
	output = env.MustRun("list")
	assert.NotContains(t, output, "production")
	assert.Contains(t, output, "No AWS profiles found")
}

// TestCredentialCaching tests that credentials are cached and reused
func TestCredentialCaching(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Setup
	env.SetupVault()
	env.CreateConfigProfile("testprofile", "us-west-2", "")
	env.SetupProfile("testprofile")

	// First exec - should get new credentials
	start1 := time.Now()
	output1 := env.MustRun("exec", "testprofile", "--", "env")
	duration1 := time.Since(start1)

	assert.Contains(t, output1, "Getting temporary credentials")
	assert.True(t, env.CacheExists("testprofile"), "cache file should be created")

	// Verify cache file has correct structure
	cache := env.ReadCache("testprofile")
	assert.Equal(t, "session", cache["Type"], "cache should be session type")
	assert.NotEmpty(t, cache["AccessKeyId"])
	assert.NotEmpty(t, cache["SecretAccessKey"])
	assert.NotEmpty(t, cache["SessionToken"])

	// Second exec - should use cache
	start2 := time.Now()
	output2 := env.MustRun("exec", "testprofile", "--", "env")
	duration2 := time.Since(start2)

	assert.Contains(t, output2, "Using cached credentials")

	// In mock mode, cached should be noticeably faster
	if env.Mock {
		assert.Less(t, duration2, duration1/2, "cached exec should be faster")
	}
}

// TestFileLocking tests that concurrent vault access is prevented
func TestFileLocking(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Setup
	env.SetupVault()
	env.CreateConfigProfile("testprofile", "us-west-2", "")
	env.SetupProfile("testprofile")

	// Start a long-running command that holds the lock
	cmd1 := env.Command("exec", "testprofile", "--", "sleep", "2")
	require.NoError(t, cmd1.Start(), "first command should start")
	defer cmd1.Process.Kill()

	// Give it time to acquire the lock
	time.Sleep(200 * time.Millisecond)

	// Try a second command - should fail with lock error
	output := env.RunExpectError("list")
	assert.Contains(t, output, "locked by another process")

	// Wait for first command to finish
	cmd1.Wait()

	// Now it should succeed
	output = env.MustRun("list")
	assert.Contains(t, output, "testprofile")

	// Verify lock file is cleaned up
	assert.False(t, env.LockFileExists(), "lock file should be cleaned up")
}

// TestCLIFlags tests flag parsing and help/version output
func TestCLIFlags(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	tests := []struct {
		name       string
		args       []string
		wantOutput string
		wantError  bool
	}{
		{
			name:       "--version",
			args:       []string{"--version"},
			wantOutput: "caws version",
			wantError:  false,
		},
		{
			name:       "-v",
			args:       []string{"-v"},
			wantOutput: "caws version",
			wantError:  false,
		},
		{
			name:       "--help",
			args:       []string{"--help"},
			wantOutput: "Usage:",
			wantError:  false,
		},
		{
			name:       "-h",
			args:       []string{"-h"},
			wantOutput: "Usage:",
			wantError:  false,
		},
		{
			name:       "unknown command",
			args:       []string{"foobar"},
			wantOutput: "Unknown command",
			wantError:  true,
		},
		{
			name:       "missing profile arg",
			args:       []string{"add"},
			wantOutput: "Usage: caws add",
			wantError:  true,
		},
		{
			name:       "missing profile arg for exec",
			args:       []string{"exec"},
			wantOutput: "Usage: caws exec",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := env.Run(tt.args...)

			if tt.wantError {
				assert.Error(t, err, "command should fail")
			} else {
				assert.NoError(t, err, "command should succeed")
			}

			assert.Contains(t, output, tt.wantOutput)
		})
	}
}

// TestEnvironmentPropagation tests that all AWS environment variables are set correctly
func TestEnvironmentPropagation(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Setup
	env.SetupVault()
	env.CreateConfigProfile("testprofile", "eu-west-1", "")
	env.SetupProfile("testprofile")

	// Execute env command
	output := env.MustRun("exec", "testprofile", "--", "env")

	// Parse output into a map
	envVars := parseEnvOutput(output)

	// Verify required env vars
	assert.NotEmpty(t, envVars["AWS_ACCESS_KEY_ID"], "AWS_ACCESS_KEY_ID should be set")
	assert.NotEmpty(t, envVars["AWS_SECRET_ACCESS_KEY"], "AWS_SECRET_ACCESS_KEY should be set")
	assert.NotEmpty(t, envVars["AWS_SESSION_TOKEN"], "AWS_SESSION_TOKEN should be set")
	assert.Equal(t, "testprofile", envVars["AWS_VAULT"], "AWS_VAULT should match profile")
	assert.Equal(t, "eu-west-1", envVars["AWS_REGION"], "AWS_REGION should match config")
	assert.Equal(t, "eu-west-1", envVars["AWS_DEFAULT_REGION"], "AWS_DEFAULT_REGION should match config")
	assert.NotEmpty(t, envVars["AWS_CREDENTIAL_EXPIRATION"], "AWS_CREDENTIAL_EXPIRATION should be set")

	// Verify AWS_PROFILE is NOT set
	assert.Empty(t, envVars["AWS_PROFILE"], "AWS_PROFILE should not be set")
}

// TestCredentialTypeIsolation tests that session and federation caches don't interfere
func TestCredentialTypeIsolation(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Setup
	env.SetupVault()
	env.CreateConfigProfile("testprofile", "us-west-2", "")
	env.SetupProfile("testprofile")

	// 1. Run exec - should create session type cache
	output := env.MustRun("exec", "testprofile", "--", "env")
	assert.Contains(t, output, "Getting temporary credentials")

	cache := env.ReadCache("testprofile")
	assert.Equal(t, "session", cache["Type"], "exec should create session type")

	// 2. Run login - should create federation type cache
	output = env.MustRun("login", "testprofile")
	assert.Contains(t, output, "https://signin.aws.amazon.com/federation")

	cache = env.ReadCache("testprofile")
	assert.Equal(t, "federation", cache["Type"], "login should create federation type")

	// 3. Run exec again - should NOT use federation cache, should regenerate
	output = env.MustRun("exec", "testprofile", "--", "env")
	assert.Contains(t, output, "Getting temporary credentials", "should regenerate for wrong type")

	// Verify cache is now session type again
	cache = env.ReadCache("testprofile")
	assert.Equal(t, "session", cache["Type"], "exec should recreate session type")
}

// parseEnvOutput parses env command output into a map
func parseEnvOutput(output string) map[string]string {
	env := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	return env
}
