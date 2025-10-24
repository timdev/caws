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
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bw-aws add <profile-name>")
			os.Exit(1)
		}
		handleAdd(os.Args[2])
	case "list", "ls":
		handleList()
	case "exec":
		if len(os.Args) < 4 {
			fmt.Println("Usage: bw-aws exec <profile-name> -- <command>")
			os.Exit(1)
		}
		handleExec(os.Args[2], os.Args[3:])
	case "login":
		handleLogin()
	case "remove", "rm":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bw-aws remove <profile-name>")
			os.Exit(1)
		}
		handleRemove(os.Args[2])
	case "version", "--version", "-v":
		fmt.Printf("bw-aws version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`bw-aws - AWS credential manager using Bitwarden

Usage:
  bw-aws login                           Authenticate with Bitwarden
  bw-aws add <profile>                   Add AWS credentials for a profile
  bw-aws list                            List available AWS profiles
  bw-aws exec <profile> -- <command>     Execute command with AWS credentials
  bw-aws remove <profile>                Remove a profile from Bitwarden
  bw-aws version                         Show version

Examples:
  bw-aws add production
  bw-aws exec production -- aws s3 ls
  bw-aws exec dev -- env | grep AWS

Credentials are stored in Bitwarden as Secure Notes with the name:
  "bw-aws:<profile-name>"`)
}
