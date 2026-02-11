#!/bin/bash
# Hook: SubagentStop
# Parses the sub-agent transcript to extract token usage and appends to metrics.jsonl.
#
# Claude Code sends a JSON object via stdin with these fields:
#   agent_id              — unique ID of the agent invocation
#   agent_type            — agent name (e.g., "coder", "quality-reviewer")
#   agent_transcript_path — path to the JSONL transcript file
#
# Note: The transcript JSONL format is NOT a stable API. This hook may need
# updates when Claude Code changes its internal format.

set -euo pipefail

METRICS_FILE="metrics.jsonl"

# Check jq is available
if ! command -v jq &>/dev/null; then
  exit 0
fi

# Read input JSON from stdin
INPUT=$(cat)

# Extract fields from stdin JSON
AGENT_ID=$(echo "$INPUT" | jq -r '.agent_id // "unknown"')
AGENT_TYPE=$(echo "$INPUT" | jq -r '.agent_type // "unknown"')
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.agent_transcript_path // ""')

# Validate transcript path
if [[ -z "$TRANSCRIPT_PATH" || ! -f "$TRANSCRIPT_PATH" ]]; then
  exit 0
fi

# Parse token usage from transcript JSONL
TOKENS=$(jq -s '[.[] | .usage // empty] | {
  input: (map(.input_tokens // 0) | add // 0),
  output: (map(.output_tokens // 0) | add // 0)
}' "$TRANSCRIPT_PATH" 2>/dev/null || echo '{"input": 0, "output": 0}')

# Append metrics entry
echo "{\"agent_id\": \"$AGENT_ID\", \"agent_type\": \"$AGENT_TYPE\", \"tokens\": $TOKENS, \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" \
  >> "$METRICS_FILE"
