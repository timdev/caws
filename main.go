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
		if len(os.Args) < 4 {
			fmt.Println("Usage: caws exec <profile-name> -- <command>")
			os.Exit(1)
		}
		handleExec(os.Args[2], os.Args[3:])
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
  caws exec <profile> -- <command>     Execute command with AWS credentials
  caws remove <profile>                Remove a profile from vault
  caws version                         Show version

Examples:
  caws init
  caws add production
  caws exec production -- aws s3 ls
  caws exec dev -- env | grep AWS

Credentials are stored encrypted in:
  ~/.caws/vault.enc

Prerequisites:
  - AWS CLI must be installed (for STS operations)`)
}
