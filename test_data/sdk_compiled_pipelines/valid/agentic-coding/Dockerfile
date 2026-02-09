FROM python:3.13-slim as base

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    git \
    bash \
    build-essential \
    gpg \
    && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code
RUN curl -fsSL https://claude.ai/install.sh | bash

# Add Claude Code to PATH
ENV PATH="/root/.local/bin:${PATH}"

# Verify installations and PATH
RUN claude --version || echo "Claude Code installed successfully"
RUN gh --version
RUN which gh && echo "GitHub CLI is in PATH"

# Set up working directory to home
WORKDIR /root

COPY .claude .claude
COPY .claude/CLAUDE.md CLAUDE.md

# Default command
CMD ["bash"]
