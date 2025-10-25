FROM golang:1.24-bookworm

# Install system dependencies including Node.js
RUN apt-get update && apt-get install -y \
    git \
    vim \
    curl \
    build-essential \
    nodejs \
    npm \
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
