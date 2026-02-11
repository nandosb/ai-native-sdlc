#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# Colors & Styles
# ─────────────────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m' # No Color

# ─────────────────────────────────────────────────────────────────────────────
# Logging Functions
# ─────────────────────────────────────────────────────────────────────────────
info()     { echo -e "${GREEN}[✓]${NC} $*"; }
warn()     { echo -e "${YELLOW}[!]${NC} $*"; }
error()    { echo -e "${RED}[✗]${NC} $*"; exit 1; }
step()     { echo -e "\n${CYAN}${BOLD}▶ $*${NC}"; }
substep()  { echo -e "${DIM}  ➜ $*${NC}"; }
success()  { echo -e "${GREEN}${BOLD}✓ $*${NC}"; }

# ─────────────────────────────────────────────────────────────────────────────
# Helper Functions
# ─────────────────────────────────────────────────────────────────────────────
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Print banner
print_banner() {
  echo -e "${CYAN}"
  echo "╔══════════════════════════════════════════════════════════════════════╗"
  echo "║                                                                      ║"
  echo "║           🚀 MX Hackathon Development Environment Setup 🚀          ║"
  echo "║                                                                      ║"
  echo "╚══════════════════════════════════════════════════════════════════════╝"
  echo -e "${NC}"
  echo -e "${DIM}This script will set up your development environment with:${NC}"
  echo -e "${DIM}  • Python package manager (uv)${NC}"
  echo -e "${DIM}  • Claude Code CLI${NC}"
  echo -e "${DIM}  • MCP Servers (Serena, Context7)${NC}"
  echo ""
}

# Print usage
print_usage() {
  echo "Usage: ./setup.sh [options]"
  echo ""
  echo "Options:"
  echo "  -h, --help     Show this help message"
  echo "  -y, --yes      Skip confirmation prompts (auto-yes)"
  echo ""
  exit 0
}

# Confirm action
confirm() {
  if [[ "${AUTO_YES:-false}" == "true" ]]; then
    return 0
  fi
  
  local prompt="$1"
  local response
  echo -e "${YELLOW}[?]${NC} ${prompt} ${DIM}(y/n)${NC}"
  read -rp "    " response
  case "$response" in
    [yY][eE][sS]|[yY]) return 0 ;;
    *) return 1 ;;
  esac
}

# Track installed components
declare -a INSTALLED_COMPONENTS=()
declare -a SKIPPED_COMPONENTS=()

# Track installed components
declare -a INSTALLED_COMPONENTS=()
declare -a SKIPPED_COMPONENTS=()

# Check if a Claude MCP server is already configured by name.
# Usage: mcp_exists "server-name"
mcp_exists() {
  claude mcp list 2>/dev/null | grep -q "^$1:"
}

# Print final summary
print_summary() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║${NC}  ${BOLD}Installation Summary${NC}                                              ${CYAN}║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
  
  if [[ ${#INSTALLED_COMPONENTS[@]} -gt 0 ]]; then
    echo -e "${GREEN}${BOLD}✓ Installed/Configured:${NC}"
    for component in "${INSTALLED_COMPONENTS[@]}"; do
      echo -e "  ${GREEN}•${NC} $component"
    done
    echo ""
  fi
  
  if [[ ${#SKIPPED_COMPONENTS[@]} -gt 0 ]]; then
    echo -e "${YELLOW}${BOLD}⊘ Skipped:${NC}"
    for component in "${SKIPPED_COMPONENTS[@]}"; do
      echo -e "  ${YELLOW}•${NC} $component"
    done
    echo ""
  fi
  
  success "Environment setup complete!"
  echo ""
  echo -e "${CYAN}${BOLD}Next Steps:${NC}"
  echo -e "  ${DIM}1.${NC} Start building amazing features! 🎉"
  echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Main Setup Flow
# ─────────────────────────────────────────────────────────────────────────────

# Parse command line arguments
AUTO_YES=false
for arg in "$@"; do
  case $arg in
    -h|--help)
      print_usage
      ;;
    -y|--yes)
      AUTO_YES=true
      shift
      ;;
    *)
      echo "Unknown option: $arg"
      print_usage
      ;;
  esac
done

# Show banner
print_banner

# Check if .claude directory already exists (initial setup already run)
if [[ -d "$PROJECT_DIR/.claude" ]]; then
  echo ""
  info "Setup has already been completed!"
  echo -e "${DIM}The .claude directory exists, indicating initial setup was already run.${NC}"
  echo ""
  echo -e "${CYAN}${BOLD}Available actions:${NC}"
  echo -e "  ${DIM}•${NC} Run individual setup steps if needed"
  echo -e "  ${DIM}•${NC} Delete .claude directory to run fresh setup"
  echo ""
  exit 0
fi

# Confirm to proceed
if ! confirm "Ready to set up your development environment?"; then
  echo -e "${YELLOW}Setup cancelled by user.${NC}"
  exit 0
fi

echo ""


echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Step 1: Install uv (Python package manager)
# ─────────────────────────────────────────────────────────────────────────────
step "Step 1/4: Installing uv (Python package manager)"

if command -v uv &>/dev/null; then
  substep "uv is already installed ($(uv --version))"
  info "Skipping uv installation"
  INSTALLED_COMPONENTS+=("uv (already installed)")
else
  substep "Installing uv package manager..."
  if curl -LsSf https://astral.sh/uv/install.sh | sh; then
    # Source the env so uv is available in this session
    export PATH="$HOME/.local/bin:$PATH"
    if command -v uv &>/dev/null; then
      success "uv installed successfully ($(uv --version))"
      INSTALLED_COMPONENTS+=("uv package manager")
    else
      error "uv installation failed - command not found after install"
    fi
  else
    error "uv installation script failed"
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 2: Install Claude Code CLI
# ─────────────────────────────────────────────────────────────────────────────
step "Step 2/4: Installing Claude Code CLI"

if command -v claude &>/dev/null; then
  substep "Claude Code CLI is already installed"
  info "Skipping Claude Code CLI installation"
  INSTALLED_COMPONENTS+=("Claude Code CLI (already installed)")
else
  substep "Installing Claude Code CLI..."
  if curl -fsSL https://claude.ai/install.sh | bash; then
    # Source the env so claude is available in this session
    export PATH="$HOME/.local/bin:$HOME/.claude/bin:$PATH"
    if command -v claude &>/dev/null; then
      success "Claude Code CLI installed successfully"
      INSTALLED_COMPONENTS+=("Claude Code CLI")
    else
      error "Claude Code CLI installation failed - command not found after install"
    fi
  else
    error "Claude Code CLI installation script failed"
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 3: Configure MCP Servers
# ─────────────────────────────────────────────────────────────────────────────
step "Step 3/4: Configuring MCP Servers"

echo ""
echo -e "${DIM}MCP (Model Context Protocol) servers enhance Claude with additional capabilities.${NC}"
echo ""

# Serena — semantic code analysis MCP server
substep "Configuring Serena MCP server (semantic code analysis)..."
if mcp_exists "serena"; then
  info "Serena MCP server is already configured"
  INSTALLED_COMPONENTS+=("Serena MCP (already configured)")
else
  if confirm "Would you like to add the Serena MCP server for semantic code analysis?"; then
    substep "Adding Serena MCP server..."
    if claude mcp add serena \
      -- uvx --from git+https://github.com/oraios/serena \
      serena start-mcp-server \
      --context ide-assistant \
      --project "$PROJECT_DIR"; then
      success "Serena MCP server added successfully"
      INSTALLED_COMPONENTS+=("Serena MCP server")
    else
      warn "Failed to add Serena MCP server"
      SKIPPED_COMPONENTS+=("Serena MCP server (installation failed)")
    fi
  else
    warn "Skipping Serena MCP server"
    SKIPPED_COMPONENTS+=("Serena MCP server (user declined)")
  fi
fi

# Context7 — up-to-date documentation for LLM prompts
echo ""
substep "Configuring Context7 MCP server (library documentation)..."
if mcp_exists "context7"; then
  info "Context7 MCP server is already configured"
  INSTALLED_COMPONENTS+=("Context7 MCP (already configured)")
else
  echo ""
  echo -e "${CYAN}  ℹ  Context7 provides up-to-date library documentation in your prompts.${NC}"
  echo -e "${DIM}     Get your API key at: ${NC}${BLUE}https://context7.com/dashboard${NC}"
  echo ""
  
  if confirm "Would you like to add the Context7 MCP server?"; then
    read -rp "$(echo -e "${YELLOW}[?]${NC} Enter your Context7 API key: ")" CTX7_API_KEY
    
    if [[ -n "$CTX7_API_KEY" ]]; then
      substep "Adding Context7 MCP server..."
      if claude mcp add context7 -- npx -y @upstash/context7-mcp --api-key "$CTX7_API_KEY"; then
        success "Context7 MCP server added successfully"
        INSTALLED_COMPONENTS+=("Context7 MCP server")
      else
        warn "Failed to add Context7 MCP server"
        SKIPPED_COMPONENTS+=("Context7 MCP server (installation failed)")
      fi
    else
      warn "No API key provided, skipping Context7"
      SKIPPED_COMPONENTS+=("Context7 MCP server (no API key)")
    fi
  else
    warn "Skipping Context7 MCP server"
    SKIPPED_COMPONENTS+=("Context7 MCP server (user declined)")
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Step 4: Initialize Claude Project
# ─────────────────────────────────────────────────────────────────────────────
step "Step 4/4: Initializing Claude Project"

echo ""
substep "Checking for existing claude.md file..."

if [[ -f "$PROJECT_DIR/claude.md" ]]; then
  info "Project already initialized (claude.md exists)"
  INSTALLED_COMPONENTS+=("Claude project (already initialized)")
else
  substep "Initializing Claude project..."
  echo ""
  echo -e "${DIM}Claude will now initialize this project and create a claude.md file.${NC}"
  echo -e "${DIM}This file will contain project context and instructions for Claude.${NC}"
  echo ""
  
  # Change to project directory and initialize
  cd "$PROJECT_DIR" || error "Failed to change to project directory"
  
  # Create .claude directory if it doesn't exist and download settings
  substep "Configuring Claude permissions..."
  mkdir -p "$PROJECT_DIR/.claude"
  
  if [[ ! -f "$PROJECT_DIR/.claude/settings.json" ]]; then
    substep "Downloading settings template from ai-native-sdlc repository..."
    if curl -fsSL https://raw.githubusercontent.com/nandosb/ai-native-sdlc/main/claude-templates/settings.json -o "$PROJECT_DIR/.claude/settings.json"; then
      success "Applied pre-configured permissions from remote template"
    else
      warn "Failed to download settings template from remote repository"
    fi
  else
    substep "Settings file already exists, skipping template download"
  fi
  
  # Download Claude commands from remote repository
  substep "Syncing Claude commands from repository..."
  mkdir -p "$PROJECT_DIR/.claude/commands"
  
  # Create a temporary directory for cloning
  TEMP_REPO=$(mktemp -d)
  
  substep "Fetching latest commands from ai-native-sdlc repository..."
  if git clone --depth 1 --filter=blob:none --sparse https://github.com/nandosb/ai-native-sdlc.git "$TEMP_REPO" 2>/dev/null; then
    cd "$TEMP_REPO" || error "Failed to change to temp directory"
    git sparse-checkout set claude-commands 2>/dev/null
    
    # Copy command files that don't exist locally
    if [[ -d "$TEMP_REPO/claude-commands" ]]; then
      files_synced=0
      files_skipped=0
      
      for cmd_file in "$TEMP_REPO/claude-commands"/*.md; do
        if [[ -f "$cmd_file" ]]; then
          filename=$(basename "$cmd_file")
          if [[ ! -f "$PROJECT_DIR/.claude/commands/$filename" ]]; then
            cp "$cmd_file" "$PROJECT_DIR/.claude/commands/$filename"
            ((files_synced++))
          else
            ((files_skipped++))
          fi
        fi
      done
      
      cd "$PROJECT_DIR" || error "Failed to return to project directory"
      rm -rf "$TEMP_REPO"
      
      if [[ $files_synced -gt 0 ]]; then
        success "Synced $files_synced command file(s)"
      fi
      if [[ $files_skipped -gt 0 ]]; then
        substep "Skipped $files_skipped existing file(s)"
      fi
    else
      warn "claude-commands directory not found in repository"
      rm -rf "$TEMP_REPO"
    fi
  else
    warn "Failed to clone ai-native-sdlc repository"
    rm -rf "$TEMP_REPO" 2>/dev/null || true
  fi
  
  # Run claude /init only if CLAUDE.md doesn't exist
  if [[ ! -f "$PROJECT_DIR/.claude/CLAUDE.md" ]]; then
    substep "Running claude initialization..."
    echo "" | claude init 2>/dev/null || true
    echo "" | claude /init 2>/dev/null || true
    
    # Move CLAUDE.md from root to .claude if it was created in root
    if [[ -f "$PROJECT_DIR/CLAUDE.md" ]]; then
      substep "Moving CLAUDE.md to .claude directory..."
      mv "$PROJECT_DIR/CLAUDE.md" "$PROJECT_DIR/.claude/CLAUDE.md"
    fi
  else
    substep "CLAUDE.md already exists, skipping initialization"
  fi
  
  # Check if initialization created project files
  if [[ -d "$PROJECT_DIR/.claude" ]] || [[ -d "$PROJECT_DIR/.serena" ]]; then
    success "Claude project initialized successfully"
    
    INSTALLED_COMPONENTS+=("Claude project initialization")
    
    if [[ -f "$PROJECT_DIR/claude.md" ]]; then
      substep "Created claude.md at: $PROJECT_DIR/claude.md"
    fi
    if [[ -d "$PROJECT_DIR/.claude" ]]; then
      substep "Created .claude directory"
    fi
    if [[ -d "$PROJECT_DIR/.serena" ]]; then
      substep "Created .serena directory with project memories"
    fi
  else
    warn "Failed to initialize Claude project automatically"
    warn "You can initialize it manually by running: claude init"
    SKIPPED_COMPONENTS+=("Claude project initialization (failed)")
  fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# Completion
# ─────────────────────────────────────────────────────────────────────────────
print_summary