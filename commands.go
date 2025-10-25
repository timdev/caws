package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// readConfirmation prompts for yes/no or auto-confirms in test mode
func readConfirmation(prompt string) bool {
	// Check for test mode
	if os.Getenv("CAWS_AUTO_CONFIRM") != "" {
		fmt.Printf("%s[auto-confirmed]\n", prompt)
		return true
	}

	// Normal interactive prompt
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "yes" || response == "y"
}

// handleAdd handles adding a new AWS profile
func handleAdd(profile string) error {
	// Validate profile name
	if err := validateProfileName(profile); err != nil {
		return err
	}

	// Check if profile exists in ~/.aws/config
	exists, err := profileExistsInConfig(profile)
	if err != nil {
		return fmt.Errorf("failed to check ~/.aws/config: %w", err)
	}

	if !exists {
		// Profile not found - ask user if they want to create it
		fmt.Printf("Profile '%s' not found in ~/.aws/config\n", profile)

		if readConfirmation("Would you like to create it? (yes/no): ") {
			if err := createConfigProfile(profile); err != nil {
				return fmt.Errorf("failed to create profile in config: %w", err)
			}
			fmt.Printf("✓ Created [profile %s] in ~/.aws/config\n\n", profile)
		} else {
			fmt.Println("Cancelled. Check your profile name.")
			return nil
		}
	}

	// Now proceed with adding credentials to vault
	client, err := NewVaultClient()
	if err != nil {
		return err
	}
	defer client.Close()

	fmt.Printf("Adding AWS credentials for profile: %s\n\n", profile)

	// Get AWS Access Key ID
	var accessKey string
	if testAccessKey := os.Getenv("CAWS_TEST_ACCESS_KEY"); testAccessKey != "" {
		fmt.Println("AWS Access Key ID: [test mode]")
		accessKey = testAccessKey
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("AWS Access Key ID: ")
		accessKey, _ = reader.ReadString('\n')
		accessKey = strings.TrimSpace(accessKey)
	}

	// Validate access key format
	if err := validateAccessKey(accessKey); err != nil {
		return err
	}

	// Get AWS Secret Access Key (hidden input)
	var secretKey string
	var secretKeyBytes []byte

	// Check for test mode (allows non-interactive input)
	if testSecretKey := os.Getenv("CAWS_TEST_SECRET_KEY"); testSecretKey != "" {
		fmt.Println("AWS Secret Access Key: [test mode]")
		secretKey = testSecretKey
	} else {
		fmt.Print("AWS Secret Access Key: ")
		secretKeyBytes, err = term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read secret key: %w", err)
		}

		secretKey = string(secretKeyBytes)
		// Clear sensitive data from memory
		defer func() {
			for i := range secretKeyBytes {
				secretKeyBytes[i] = 0
			}
		}()
	}

	// Create credentials in vault (only access_key + secret_key)
	if err := client.CreateCredentials(profile, accessKey, secretKey); err != nil {
		return fmt.Errorf("failed to create profile in vault: %w", err)
	}

	fmt.Printf("✓ Successfully added profile '%s' to vault\n\n", profile)

	// Check if config has region/MFA configured
	configSettings, err := getConfigSettings(profile)
	if err == nil {
		if configSettings.Region != "" {
			fmt.Printf("Using region '%s' from ~/.aws/config\n", configSettings.Region)
		} else {
			fmt.Println("Tip: Add region to ~/.aws/config:")
			fmt.Printf("  [profile %s]\n", profile)
			fmt.Println("  region = us-east-1")
		}

		if configSettings.MFASerial != "" {
			fmt.Println("MFA configured in ~/.aws/config")
		} else {
			fmt.Println("\nTip: To enable MFA, add to ~/.aws/config:")
			fmt.Printf("  [profile %s]\n", profile)
			fmt.Println("  mfa_serial = arn:aws:iam::123456789012:mfa/your-username")
		}
	}

	return nil
}

// handleList handles listing AWS profiles
func handleList() error {
	client, err := NewVaultClient()
	if err != nil {
		return err
	}
	defer client.Close()

	profiles, err := client.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No AWS profiles found. Add one with: caws add <profile-name>")
		return nil
	}

	fmt.Println("Available AWS profiles:")
	for _, profile := range profiles {
		fmt.Printf("  • %s", profile.Name)
		if profile.Region != "" {
			fmt.Printf(" (region: %s)", profile.Region)
		}
		if profile.MFASerial != "" {
			fmt.Printf(" [MFA enabled]")
		}
		fmt.Println()
	}

	return nil
}

// handleExec handles executing a command with AWS credentials
func handleExec(profile string, args []string) error {
	// Validate profile name
	if err := validateProfileName(profile); err != nil {
		return err
	}

	// Skip "--" if present
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	// Determine if we're spawning a shell or running a command
	spawnShell := len(args) == 0

	// Check for cached credentials FIRST (before prompting for password)
	stsCreds, err := GetCachedCredentials(profile)
	if err != nil || stsCreds.Type != "session" {
		// Cache miss, expired, or wrong type - need to get fresh credentials from vault
		client, err := NewVaultClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Get credentials from vault
		creds, err := client.GetCredentials(profile)
		if err != nil {
			return fmt.Errorf("failed to get profile '%s': %w\nRun 'caws list' to see available profiles", profile, err)
		}

		// Get region and MFA from ~/.aws/config
		configSettings, err := getConfigSettings(profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read ~/.aws/config: %v\n", err)
			configSettings = &ConfigSettings{} // Use empty settings
		}

		// Set region (default to us-east-1 if not configured)
		if configSettings.Region != "" {
			creds.Region = configSettings.Region
		} else {
			creds.Region = "us-east-1"
			fmt.Fprintln(os.Stderr, "⚠️  Warning: No region configured in ~/.aws/config, using us-east-1")
		}

		// Set MFA serial if configured
		creds.MFASerial = configSettings.MFASerial

		fmt.Println("Getting temporary credentials...")

		// Get MFA code if needed
		var mfaCode string
		if creds.MFASerial != "" {
			fmt.Print("Enter MFA code: ")
			reader := bufio.NewReader(os.Stdin)
			mfaCode, _ = reader.ReadString('\n')
			mfaCode = strings.TrimSpace(mfaCode)
		}

		// Get temporary credentials
		stsCreds, err = AssumeRole(creds, 3600, mfaCode)
		if err != nil {
			return fmt.Errorf("failed to get temporary credentials: %w", err)
		}

		// Cache them
		if err := CacheCredentials(profile, stsCreds); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache credentials: %v\n", err)
		} else {
			fmt.Printf("✓ Credentials cached (valid until %s)\n", stsCreds.Expiration.Format("15:04:05"))
		}
	} else {
		fmt.Printf("Using cached credentials (valid until %s)\n", stsCreds.Expiration.Format("15:04:05"))
	}

	// Set up environment
	env := SetEnvVars(profile, stsCreds, stsCreds.Region)

	var cmd *exec.Cmd

	if spawnShell {
		// No command specified - spawn a subshell
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}

		cmd = exec.Command(shell)

		fmt.Printf("Spawning subshell with AWS credentials for profile '%s'\n", profile)
		fmt.Printf("Credentials valid until %s\n", stsCreds.Expiration.Format("15:04:05"))
		fmt.Println("Type 'exit' to return to your normal shell")
		fmt.Println()
	} else {
		// Execute the specified command
		cmd = exec.Command(args[0], args[1:]...)
	}

	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

// handleRemove handles removing an AWS profile
func handleRemove(profile string) error {
	// Validate profile name
	if err := validateProfileName(profile); err != nil {
		return err
	}

	client, err := NewVaultClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Confirm deletion
	if !readConfirmation(fmt.Sprintf("Are you sure you want to remove profile '%s'? (yes/no): ", profile)) {
		fmt.Println("Cancelled")
		return nil
	}

	if err := client.RemoveProfile(profile); err != nil {
		return fmt.Errorf("failed to remove profile: %w", err)
	}

	fmt.Printf("✓ Successfully removed profile '%s'\n", profile)

	// Remove cached credentials
	cacheDir := getCacheDir()
	cachePath := fmt.Sprintf("%s/%s.json", cacheDir, profile)
	os.Remove(cachePath) // Ignore errors

	return nil
}

// handleLogin handles generating an AWS Console login URL
func handleLogin(profile string) error {
	// Validate profile name
	if err := validateProfileName(profile); err != nil {
		return err
	}

	// Check for cached credentials FIRST (before prompting for password)
	stsCreds, err := GetCachedCredentials(profile)
	if err != nil || stsCreds.Type != "federation" {
		// Cache miss, expired, or wrong type - need to get fresh credentials from vault
		client, err := NewVaultClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Get credentials from vault
		creds, err := client.GetCredentials(profile)
		if err != nil {
			return fmt.Errorf("failed to get profile '%s': %w\nRun 'caws list' to see available profiles", profile, err)
		}

		// Get region and MFA from ~/.aws/config
		configSettings, err := getConfigSettings(profile)
		if err != nil {
			return fmt.Errorf("failed to read ~/.aws/config: %w", err)
		}

		// Set region (default to us-east-1 if not configured)
		if configSettings.Region != "" {
			creds.Region = configSettings.Region
		} else {
			creds.Region = "us-east-1"
		}

		// Get federation token for console login (12 hour duration)
		// Note: GetFederationToken doesn't support MFA parameter, but the base
		// credentials are still protected by MFA if configured
		stsCreds, err = GetFederationToken(creds, 43200, profile)
		if err != nil {
			return fmt.Errorf("failed to get federation token: %w", err)
		}

		// Cache them
		if err := CacheCredentials(profile, stsCreds); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache credentials: %v\n", err)
		}
	}

	// Generate console URL
	consoleURL, err := GetConsoleURL(stsCreds, stsCreds.Region)
	if err != nil {
		return fmt.Errorf("failed to generate console URL: %w", err)
	}

	// Print ONLY the URL to stdout (for piping to pbcopy, etc.)
	fmt.Println(consoleURL)

	return nil
}
