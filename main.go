package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Define global flags
	versionFlag := flag.Bool("version", false, "show version")
	flag.BoolVar(versionFlag, "v", false, "show version (shorthand)")
	helpFlag := flag.Bool("help", false, "show help")
	flag.BoolVar(helpFlag, "h", false, "show help (shorthand)")

	flag.Usage = printUsage
	flag.Parse()

	// Handle version and help flags
	if *versionFlag {
		fmt.Printf("caws %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

	if *helpFlag {
		printUsage()
		return
	}

	// Get subcommand
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	// Execute subcommand
	var err error
	switch command {
	case "init":
		err = InitVault()
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: caws add <profile-name>")
			os.Exit(1)
		}
		err = handleAdd(args[1])
	case "list", "ls":
		err = handleList()
	case "exec":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: caws exec <profile-name> [-- <command>]")
			os.Exit(1)
		}
		err = handleExec(args[1], args[2:])
	case "login":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: caws login <profile-name>")
			os.Exit(1)
		}
		err = handleLogin(args[1])
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: caws remove <profile-name>")
			os.Exit(1)
		}
		err = handleRemove(args[1])
	case "version":
		fmt.Printf("caws %s (commit: %s, built: %s)\n", version, commit, date)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
  $XDG_DATA_HOME/caws/vault.enc (encrypted access keys, defaults to ~/.local/share/caws/vault.enc)
  $XDG_CACHE_HOME/caws/ (temporary credentials cache, defaults to ~/.cache/caws/)
  ~/.aws/config (profile settings: region, MFA)

Environment variables set:
  AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN
  AWS_VAULT=<profile> (for shell prompts)
  AWS_REGION, AWS_DEFAULT_REGION
  AWS_CREDENTIAL_EXPIRATION`)
}
