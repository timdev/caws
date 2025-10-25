package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func handleAdd(profile string) {
	// Check if profile exists in ~/.aws/config
	exists, err := profileExistsInConfig(profile)
	if err != nil {
		fmt.Printf("Error: Failed to check ~/.aws/config: %v\n", err)
		os.Exit(1)
	}

	if !exists {
		// Profile not found - ask user if they want to create it
		fmt.Printf("Profile '%s' not found in ~/.aws/config\n", profile)

		if readConfirmation("Would you like to create it? (yes/no): ") {
			if err := createConfigProfile(profile); err != nil {
				fmt.Printf("Error creating profile in config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Created [profile %s] in ~/.aws/config\n\n", profile)
		} else {
			fmt.Println("Cancelled. Check your profile name.")
			os.Exit(0)
		}
	}

	// Now proceed with adding credentials to vault
	gp, err := NewVaultClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer gp.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Adding AWS credentials for profile: %s\n\n", profile)

	// Get AWS Access Key ID
	fmt.Print("AWS Access Key ID: ")
	accessKey, _ := reader.ReadString('\n')
	accessKey = strings.TrimSpace(accessKey)

	// Get AWS Secret Access Key (hidden input)
	fmt.Print("AWS Secret Access Key: ")

	// Disable echo
	exec.Command("stty", "-echo").Run()
	secretKey, _ := reader.ReadString('\n')
	exec.Command("stty", "echo").Run()

	secretKey = strings.TrimSpace(secretKey)
	fmt.Println()

	// Create credentials in vault (only access_key + secret_key)
	if err := gp.CreateCredentials(profile, accessKey, secretKey); err != nil {
		fmt.Printf("Error creating profile in vault: %v\n", err)
		os.Exit(1)
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
}

// handleList handles listing AWS profiles
func handleList() {
	gp, err := NewVaultClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer gp.Close()

	profiles, err := gp.ListProfiles()
	if err != nil {
		fmt.Printf("Error listing profiles: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Println("No AWS profiles found. Add one with: caws add <profile-name>")
		return
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
}

// handleExec handles executing a command with AWS credentials
func handleExec(profile string, args []string) {
	// Skip "--" if present
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	// Determine if we're spawning a shell or running a command
	spawnShell := len(args) == 0

	// Check for cached credentials FIRST (before prompting for password)
	stsCreds, err := GetCachedCredentials(profile)
	if err != nil {
		// Cache miss or expired - need to get fresh credentials from vault
		gp, err := NewVaultClient()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer gp.Close()

		// Get credentials from vault
		creds, err := gp.GetCredentials(profile)
		if err != nil {
			fmt.Printf("Error getting profile '%s': %v\n", profile, err)
			fmt.Println("Run 'caws list' to see available profiles")
			os.Exit(1)
		}

		// Get region and MFA from ~/.aws/config
		configSettings, err := getConfigSettings(profile)
		if err != nil {
			fmt.Printf("Warning: Failed to read ~/.aws/config: %v\n", err)
			configSettings = &ConfigSettings{} // Use empty settings
		}

		// Set region (default to us-east-1 if not configured)
		if configSettings.Region != "" {
			creds.Region = configSettings.Region
		} else {
			creds.Region = "us-east-1"
			fmt.Println("⚠️  Warning: No region configured in ~/.aws/config, using us-east-1")
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
			fmt.Printf("Error getting temporary credentials: %v\n", err)
			os.Exit(1)
		}

		// Cache them
		if err := CacheCredentials(profile, stsCreds); err != nil {
			fmt.Printf("Warning: failed to cache credentials: %v\n", err)
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
		fmt.Printf("Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// handleRemove handles removing an AWS profile
func handleRemove(profile string) {
	gp, err := NewVaultClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer gp.Close()

	// Confirm deletion
	if !readConfirmation(fmt.Sprintf("Are you sure you want to remove profile '%s'? (yes/no): ", profile)) {
		fmt.Println("Cancelled")
		return
	}

	if err := gp.RemoveProfile(profile); err != nil {
		fmt.Printf("Error removing profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully removed profile '%s'\n", profile)

	// Remove cached credentials
	cacheDir := getCacheDir()
	cachePath := fmt.Sprintf("%s/%s.json", cacheDir, profile)
	os.Remove(cachePath) // Ignore errors
}
