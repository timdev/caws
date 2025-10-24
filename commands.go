package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const itemPrefix = "bw-aws:"

// handleLogin handles the login command
func handleLogin() {
	bw := NewBitwardenClient()
	if err := bw.Login(); err != nil {
		fmt.Printf("Error logging in: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully authenticated with Bitwarden")
}

// handleAdd handles adding a new AWS profile
func handleAdd(profile string) {
	bw := NewBitwardenClient()
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

	// Create fields map
	fields := map[string]string{
		"aws_access_key_id":     accessKey,
		"aws_secret_access_key": secretKey,
	}

	if region != "" {
		fields["region"] = region
	}

	if mfaSerial != "" {
		fields["mfa_serial"] = mfaSerial
	}

	// Create secure note in Bitwarden
	itemName := itemPrefix + profile
	notes := fmt.Sprintf("AWS credentials for profile: %s\nManaged by bw-aws", profile)

	if err := bw.CreateSecureNote(itemName, fields, notes); err != nil {
		fmt.Printf("Error creating item in Bitwarden: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully added profile '%s' to Bitwarden\n", profile)
}

// handleList handles listing AWS profiles
func handleList() {
	bw := NewBitwardenClient()
	items, err := bw.ListItems(itemPrefix)
	if err != nil {
		fmt.Printf("Error listing profiles: %v\n", err)
		os.Exit(1)
	}

	if len(items) == 0 {
		fmt.Println("No AWS profiles found. Add one with: bw-aws add <profile-name>")
		return
	}

	fmt.Println("Available AWS profiles:")
	for _, item := range items {
		if strings.HasPrefix(item.Name, itemPrefix) {
			profile := strings.TrimPrefix(item.Name, itemPrefix)
			region := GetFieldValue(&item, "region")
			mfaSerial := GetFieldValue(&item, "mfa_serial")

			fmt.Printf("  • %s", profile)
			if region != "" {
				fmt.Printf(" (region: %s)", region)
			}
			if mfaSerial != "" {
				fmt.Printf(" [MFA enabled]")
			}
			fmt.Println()
		}
	}
}

// handleExec handles executing a command with AWS credentials
func handleExec(profile string, args []string) {
	bw := NewBitwardenClient()

	// Skip "--" if present
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("No command specified")
		os.Exit(1)
	}

	// Get credentials from Bitwarden
	itemName := itemPrefix + profile
	item, err := bw.GetItem(itemName)
	if err != nil {
		fmt.Printf("Error getting profile '%s': %v\n", profile, err)
		fmt.Println("Run 'bw-aws list' to see available profiles")
		os.Exit(1)
	}

	// Extract credentials
	creds := &AWSCredentials{
		AccessKeyID:     GetFieldValue(item, "aws_access_key_id"),
		SecretAccessKey: GetFieldValue(item, "aws_secret_access_key"),
		Region:          GetFieldValue(item, "region"),
		MFASerial:       GetFieldValue(item, "mfa_serial"),
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		fmt.Println("Invalid credentials in Bitwarden item")
		os.Exit(1)
	}

	// Check for cached credentials first
	stsCreds, err := GetCachedCredentials(profile)
	if err != nil {
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
	env := SetEnvVars(stsCreds, creds.Region)

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
	bw := NewBitwardenClient()
	itemName := itemPrefix + profile
	item, err := bw.GetItem(itemName)
	if err != nil {
		fmt.Printf("Error: profile '%s' not found\n", profile)
		os.Exit(1)
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to remove profile '%s'? (yes/no): ", profile)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" && response != "y" {
		fmt.Println("Cancelled")
		return
	}

	if err := bw.DeleteItem(item.ID); err != nil {
		fmt.Printf("Error removing profile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully removed profile '%s'\n", profile)

	// Remove cached credentials
	cacheDir := getCacheDir()
	cachePath := fmt.Sprintf("%s/%s.json", cacheDir, profile)
	os.Remove(cachePath) // Ignore errors
}
