package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		if err := InitVault(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: caws add <profile-name>")
			os.Exit(1)
		}
		handleAdd(os.Args[2])
	case "list", "ls":
		handleList()
	case "exec":
		if len(os.Args) < 3 {
			fmt.Println("Usage: caws exec <profile-name> [-- <command>]")
			os.Exit(1)
		}
		handleExec(os.Args[2], os.Args[3:])
	case "login":
		if len(os.Args) < 3 {
			fmt.Println("Usage: caws login <profile-name>")
			os.Exit(1)
		}
		handleLogin(os.Args[2])
	case "remove", "rm":
		if len(os.Args) < 3 {
			fmt.Println("Usage: caws remove <profile-name>")
			os.Exit(1)
		}
		handleRemove(os.Args[2])
	case "version", "--version", "-v":
		fmt.Printf("caws version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`caws - Fast, local-first AWS credential manager

Usage:
  caws init                            Initialize a new encrypted vault
  caws add <profile>                   Add AWS credentials for a profile
  caws list                            List available AWS profiles
  caws exec <profile>                  Spawn subshell with AWS credentials
  caws exec <profile> -- <command>     Execute command with AWS credentials
  caws login <profile>                 Generate AWS Console login URL
  caws remove <profile>                Remove a profile from vault
  caws version                         Show version

Examples:
  caws init
  caws add production
  caws exec production                 # Spawns shell with credentials
  caws exec production -- aws s3 ls    # Run single command
  caws login production | pbcopy       # Copy console URL to clipboard

Credentials stored in:
  ~/.caws/vault.enc (encrypted access keys)
  ~/.aws/config (profile settings: region, MFA)

Environment variables set:
  AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN
  AWS_VAULT=<profile> (for shell prompts)
  AWS_REGION, AWS_DEFAULT_REGION
  AWS_CREDENTIAL_EXPIRATION`)
}
