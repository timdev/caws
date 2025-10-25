# Docker Sandbox for Claude Code Development

This Docker setup provides an isolated development environment for using Claude Code with `--dangerously-skip-permissions` safely.

## Why Use a Docker Sandbox?

When using Claude Code with `--dangerously-skip-permissions`, the AI can execute commands without asking permission. While convenient for development, this requires isolation:

- **Filesystem protection**: Changes are contained to the project directory
- **Resource limits**: Prevents CPU/memory exhaustion
- **Clean environment**: Fresh state for each session
- **No secrets at risk**: Container has no sensitive credentials

## Quick Start

```bash
# Start Claude Code in sandbox (one command!)
cd sandbox
just claude

# When done, exit Claude and stop container
just down
```

## Available Commands

All commands use `just` (command runner). Run from the `/sandbox` directory:

```bash
cd sandbox

just claude          # Start Claude Code in sandbox (primary command)
just up              # Start container in background
just down            # Stop and remove container
just shell           # Enter container bash shell (for manual work)
just build           # Build caws inside container
just test            # Run E2E tests inside container
just rebuild         # Rebuild Docker image (after Dockerfile changes)
just logs            # Show container logs
just clean           # Remove everything (container + image)
just status          # Show container/compose status
```

Or from the project root with `-f`:

```bash
just -f sandbox/Justfile claude
```

## What's Inside the Container

**Base image**: `golang:1.24-bookworm`

**Installed tools**:
- Go 1.24 toolchain
- Claude Code CLI
- git, vim, curl, build-essential
- All Go dependencies pre-downloaded

**User**: Non-root user `caws` (UID 1000)

**Working directory**: `/workspace` (mounted from host)

## How It Works

### Directory Mounting

The project directory is mounted at `/workspace` in the container:

```
Host: /path/to/caws (your project directory)
Container: /workspace
```

Changes made inside the container are reflected on the host and vice versa.

### Environment Variables

Test credentials are pre-configured (safe to use, no real permissions):

- `CAWS_TEST_AWS_ACCESS_KEY` - Test IAM user access key
- `CAWS_TEST_AWS_SECRET_KEY` - Test IAM user secret key
- `CAWS_TEST_DIR=/tmp/caws-test` - Isolated test directory
- `CAWS_PASSWORD=testpass` - Auto-confirm password prompts
- `CAWS_AUTO_CONFIRM=yes` - Auto-confirm yes/no prompts

### Resource Limits

Container is limited to prevent abuse:

- **CPU**: Max 2 cores, reserved 0.5 cores
- **Memory**: Max 2GB, reserved 512MB

### Security Boundaries

**What's protected** (host is safe from):
- Filesystem changes outside `/workspace`
- System-wide process interference
- Kernel modifications
- Docker socket access
- Unlimited resource consumption

**What's not protected** (acceptable):
- Network access (no secrets to leak)
- Package installation (user controls container)
- Resource usage within limits

## Typical Workflow

```bash
# Start Claude Code in sandbox
cd sandbox
just claude

# Claude can now freely:
#    - Build: go build -o caws
#    - Test: ./test/e2e/run_all.sh
#    - Commit: git add/commit/push
#    - Install packages: go get, apt install
#    - Edit files: all changes persist on host

# Exit Claude when done
# (Press Ctrl+D or type 'exit')

# Stop container
just down
```

## Advanced: Manual Container Access

If you need to work in the container without Claude Code:

```bash
# Start container and get a shell
cd sandbox
just shell

# Do manual work...
go build -o caws
./test/e2e/run_all.sh
git status

# Exit
exit
just down
```

## Rebuilding the Image

After changing the Dockerfile or Go dependencies:

```bash
cd sandbox
just rebuild
just up
```

## Troubleshooting

### Container won't start

```bash
# Check logs
cd sandbox
just logs

# Clean up and rebuild
just clean
just up
```

### Permission errors

The container runs as user `caws` (UID 1000). If your host user has a different UID, you may see permission issues. Fix by changing the UID in the Dockerfile:

```dockerfile
RUN useradd -m -s /bin/bash -u YOUR_UID caws
```

### Out of disk space

```bash
# Clean up Docker resources
docker system prune -a
```

## AWS Testing

The container includes test credentials for AWS integration testing. These credentials have no real permissions and are safe to use:

```bash
# Inside container
./test/e2e/run_all.sh
```

Tests will use real AWS STS API calls with the pre-configured test credentials.

## Networking

The container has **unrestricted internet access**. This is safe because:

1. No real secrets are in the container
2. Test AWS credentials have no permissions
3. Vault password is just "testpass" for testing
4. All real work happens on the host via mounted directory

If you need network isolation for other reasons, modify `docker-compose.yml`:

```yaml
networks:
  isolated:
    driver: bridge
    internal: true
```

## Cleaning Up

```bash
# Stop and remove container (keeps image)
cd sandbox
just down

# Remove everything including image
just clean
```
