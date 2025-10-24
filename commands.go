package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// handleAdd handles adding a new AWS profile
func handleAdd(profile string) {
	gp, err := NewVaultClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer gp.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Adding AWS profile: %s\n\n", profile)

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

	// Get optional region
	fmt.Print("Default Region (optional, e.g., us-east-1): ")
	region, _ := reader.ReadString('\n')
	region = strings.TrimSpace(region)

	// Get optional MFA serial
	fmt.Print("MFA Serial ARN (optional): ")
	mfaSerial, _ := reader.ReadString('\n')
	mfaSerial = strings.TrimSpace(mfaSerial)

	// Create credentials in vault
	if err := gp.CreateCredentials(profile, accessKey, secretKey, region, mfaSerial); err != nil {
		fmt.Printf("Error creating profile in vault: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully added profile '%s' to vault\n", profile)
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

	if len(args) == 0 {
		fmt.Println("No command specified")
		os.Exit(1)
	}

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
	env := SetEnvVars(stsCreds, stsCreds.Region)

	// Execute command
	cmd := exec.Command(args[0], args[1:]...)
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
	fmt.Printf("Are you sure you want to remove profile '%s'? (yes/no): ", profile)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" && response != "y" {
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
