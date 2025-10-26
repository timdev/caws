FROM golang:1.24-bookworm

# Install system dependencies including Node.js and GitHub CLI
RUN apt-get update && apt-get install -y \
    git \
    vim \
    curl \
    build-essential \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI from official repository
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI via npm
RUN npm install -g @anthropic-ai/claude-code

# Create non-root user
RUN useradd -m -s /bin/bash -u 1000 caws && \
    mkdir -p /workspace && \
    chown caws:caws /workspace

# Set working directory
WORKDIR /workspace

# Copy go.mod and go.sum first for layer caching
COPY --chown=caws:caws go.mod go.sum ./

# Download dependencies as root first, then switch to caws user
RUN go mod download

# Switch to non-root user
USER caws

# Create Claude Code agents directory
RUN mkdir -p /home/caws/.claude/agents

# Default command: bash shell
CMD ["/bin/bash"]
