#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[✓]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
error() { echo -e "${RED}[✗]${NC} $*"; exit 1; }

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Check if a Claude MCP server is already configured by name.
# Usage: mcp_exists "server-name"
mcp_exists() {
  claude mcp list 2>/dev/null | grep -q "^$1:"
}

# ─── 1. Install uv (Python package manager) ──────────────────────────────────
if command -v uv &>/dev/null; then
  info "uv is already installed ($(uv --version))"
else
  warn "uv not found — installing..."
  curl -LsSf https://astral.sh/uv/install.sh | sh
  # Source the env so uv is available in this session
  export PATH="$HOME/.local/bin:$PATH"
  if command -v uv &>/dev/null; then
    info "uv installed successfully ($(uv --version))"
  else
    error "uv installation failed"
  fi
fi

# ─── 2. Install Claude Code CLI ──────────────────────────────────────────────
if command -v claude &>/dev/null; then
  info "Claude Code CLI is already installed ($(claude --version 2>/dev/null || echo 'version unknown'))"
else
  warn "Claude Code CLI not found — installing..."
  curl -fsSL https://claude.ai/install.sh | bash
  # Source the env so claude is available in this session
  export PATH="$HOME/.local/bin:$HOME/.claude/bin:$PATH"
  if command -v claude &>/dev/null; then
    info "Claude Code CLI installed successfully"
  else
    error "Claude Code CLI installation failed"
  fi
fi

# ─── 3. Add MCP Servers ──────────────────────────────────────────────────────

# Serena — semantic code analysis MCP server
if mcp_exists "serena"; then
  info "Serena MCP server is already configured"
else
  warn "Adding Serena MCP server..."
  claude mcp add serena \
    -- uvx --from git+https://github.com/oraios/serena \
    serena start-mcp-server \
    --context ide-assistant \
    --project "$PROJECT_DIR" \
    || error "Failed to add Serena MCP server"
  info "Serena MCP server added"
fi

# Context7 — up-to-date documentation for LLM prompts
if mcp_exists "context7"; then
  info "Context7 MCP server is already configured"
else
  echo ""
  echo -e "${YELLOW}[?]${NC} Context7 MCP provides up-to-date library docs in your prompts."
  echo "    An API key is required (get one at https://context7.com/dashboard)."
  echo ""
  read -rp "    Enter your Context7 API key (or press Enter to skip): " CTX7_API_KEY

  if [[ -n "$CTX7_API_KEY" ]]; then
    warn "Adding Context7 MCP server..."
    claude mcp add context7 -- npx -y @upstash/context7-mcp --api-key "$CTX7_API_KEY" \
      || error "Failed to add Context7 MCP server"
    info "Context7 MCP server added"
  else
    warn "Skipping Context7 MCP server (no API key provided)"
  fi
fi

# ─── Done ─────────────────────────────────────────────────────────────────────
echo ""
info "Environment setup complete!"
echo "  Run 'claude' to start Claude Code."
