package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/yalochat/agentic-sdlc/internal/engine"
)

// RunConfig configures a claude CLI invocation.
type RunConfig struct {
	Prompt       string
	CWD          string
	Model        string
	AllowedTools []string
	MaxTurns     int
	SessionID    string // if set → --session-id <uuid> (first turn)
	Resume       bool   // if true → --resume <uuid> (follow-up, uses SessionID)
}

// Result is the aggregated result of a claude CLI run.
type Result struct {
	Output   string
	TokensIn int64
	TokensOut int64
	ExitCode int
}

// Run invokes the claude CLI and streams output to the event bus.
func Run(ctx context.Context, cfg RunConfig, bus *engine.EventBus, issueID string) (*Result, error) {
	args := []string{
		"-p", cfg.Prompt,
		"--output-format", "stream-json",
		"--verbose",
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if len(cfg.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(cfg.AllowedTools, ","))
	}
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns))
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if cfg.CWD != "" {
		cmd.Dir = cfg.CWD
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	result := &Result{}
	var output strings.Builder

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(line, &obj); err != nil {
			// Non-JSON line, treat as raw output
			output.Write(line)
			output.WriteString("\n")
			continue
		}

		// Publish streaming output to event bus
		if bus != nil {
			bus.Publish(engine.Event{
				Type: engine.EventAgentOutput,
				Data: map[string]interface{}{
					"issue_id": issueID,
					"raw":      string(line),
				},
			})
		}

		evtType, _ := obj["type"].(string)

		switch evtType {
		case "assistant":
			// Extract text from message.content[].text
			if msg, ok := obj["message"].(map[string]interface{}); ok {
				if contents, ok := msg["content"].([]interface{}); ok {
					for _, block := range contents {
						cb, ok := block.(map[string]interface{})
						if !ok {
							continue
						}
						if cbType, _ := cb["type"].(string); cbType == "text" {
							if text, ok := cb["text"].(string); ok && text != "" {
								output.WriteString(text)
							}
						}
					}
				}
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if v, ok := usage["input_tokens"].(float64); ok {
						result.TokensIn += int64(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok {
						result.TokensOut += int64(v)
					}
				}
			}

		case "result":
			// Final result — use as authoritative output
			if resultText, ok := obj["result"].(string); ok && resultText != "" {
				output.Reset()
				output.WriteString(resultText)
			}
			if usage, ok := obj["usage"].(map[string]interface{}); ok {
				if v, ok := usage["input_tokens"].(float64); ok {
					result.TokensIn = int64(v)
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					result.TokensOut = int64(v)
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("wait claude: %w", err)
		}
	}

	result.Output = output.String()
	return result, nil
}

// RunSession invokes the claude CLI with session support, streams text
// incrementally, and announces tool usage as progress messages.
func RunSession(
	ctx context.Context,
	cfg RunConfig,
	bus *engine.EventBus,
	execMgr *engine.ExecutionManager,
	execID string,
) (*Result, error) {
	args := []string{
		"-p", cfg.Prompt,
		"--output-format", "stream-json",
		"--verbose",
	}

	// Session management
	if cfg.Resume && cfg.SessionID != "" {
		args = append(args, "--resume", cfg.SessionID)
	} else if cfg.SessionID != "" {
		args = append(args, "--session-id", cfg.SessionID)
	}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if len(cfg.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(cfg.AllowedTools, ","))
	}
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns))
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if cfg.CWD != "" {
		cmd.Dir = cfg.CWD
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Capture stderr for error diagnostics
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude session: %w", err)
	}

	result := &Result{}
	var textAccum strings.Builder // accumulated assistant text for streaming
	lastTextUpdate := time.Now()

	// flushText updates the single assistant message with all accumulated text so far.
	flushText := func() {
		if textAccum.Len() == 0 || execMgr == nil {
			return
		}
		execMgr.UpdateLastAssistant(execID, textAccum.String())
	}

	// progress appends a system message for tool activity announcements.
	progress := func(msg string) {
		if execMgr == nil {
			return
		}
		// Flush any pending text first so the progress message appears after it
		flushText()
		execMgr.AppendMessage(execID, engine.Message{Role: "system", Content: msg})
		if bus != nil {
			bus.Publish(engine.Event{
				Type: engine.EventExecOutput,
				Data: map[string]interface{}{
					"execution_id": execID,
					"progress":     msg,
				},
			})
		}
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse into generic map to inspect all fields
		var obj map[string]interface{}
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		evtType, _ := obj["type"].(string)

		switch evtType {
		case "assistant":
			// Assistant turn — extract text content and tool_use from message.content[]
			if msg, ok := obj["message"].(map[string]interface{}); ok {
				if contents, ok := msg["content"].([]interface{}); ok {
					for _, block := range contents {
						cb, ok := block.(map[string]interface{})
						if !ok {
							continue
						}
						cbType, _ := cb["type"].(string)
						if cbType == "text" {
							if text, ok := cb["text"].(string); ok && text != "" {
								textAccum.WriteString(text)
							}
						}
						if cbType == "tool_use" {
							toolName, _ := cb["name"].(string)
							input, _ := cb["input"].(map[string]interface{})
							progress(describeToolUse(toolName, input))
						}
					}
				}
				// Collect usage
				if usage, ok := msg["usage"].(map[string]interface{}); ok {
					if v, ok := usage["input_tokens"].(float64); ok {
						result.TokensIn += int64(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok {
						result.TokensOut += int64(v)
					}
				}
			}

		case "result":
			// Final result — use the result text as authoritative output
			if resultText, ok := obj["result"].(string); ok && resultText != "" {
				textAccum.Reset()
				textAccum.WriteString(resultText)
			}
			if usage, ok := obj["usage"].(map[string]interface{}); ok {
				if v, ok := usage["input_tokens"].(float64); ok {
					result.TokensIn = int64(v)
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					result.TokensOut = int64(v)
				}
			}
		}

		// Periodically update the streaming assistant message (every 500ms)
		if textAccum.Len() > 0 && time.Since(lastTextUpdate) > 500*time.Millisecond {
			flushText()
			lastTextUpdate = time.Now()
		}
	}

	// Final flush
	flushText()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// Detect signal-based kills (SIGKILL = -1 exit code on darwin, 137 on linux)
			if result.ExitCode == -1 || result.ExitCode == 137 {
				msg := fmt.Sprintf("Claude process was killed (exit code %d). Likely OOM — consider reducing task scope or using --max-turns.", result.ExitCode)
				if execMgr != nil {
					execMgr.AppendMessage(execID, engine.Message{
						Role: "system", Content: msg,
					})
				}
			}
		} else {
			return nil, fmt.Errorf("wait claude session: %w", err)
		}
	}

	// Report stderr for diagnostics (truncate if very large)
	stderrStr := stderrBuf.String()
	if stderrStr != "" && execMgr != nil {
		if len(stderrStr) > 2000 {
			stderrStr = stderrStr[:2000] + "... (truncated)"
		}
		if textAccum.Len() == 0 || result.ExitCode != 0 {
			execMgr.AppendMessage(execID, engine.Message{
				Role:    "system",
				Content: "stderr: " + stderrStr,
			})
		}
	}

	if execMgr != nil {
		execMgr.UpdateTokens(execID, result.TokensIn, result.TokensOut)
	}

	result.Output = textAccum.String()
	return result, nil
}

// describeToolUse returns a human-readable progress string for a tool invocation.
func describeToolUse(tool string, input map[string]interface{}) string {
	filePath, _ := input["file_path"].(string)
	command, _ := input["command"].(string)
	pattern, _ := input["pattern"].(string)

	switch tool {
	case "Write":
		if filePath != "" {
			return "Writing " + shortPath(filePath)
		}
		return "Writing file..."
	case "Edit":
		if filePath != "" {
			return "Editing " + shortPath(filePath)
		}
		return "Editing file..."
	case "Read":
		if filePath != "" {
			return "Reading " + shortPath(filePath)
		}
		return "Reading file..."
	case "Glob":
		if pattern != "" {
			return "Searching files: " + pattern
		}
		return "Searching files..."
	case "Grep":
		if pattern != "" {
			return "Searching for: " + pattern
		}
		return "Searching code..."
	case "Bash":
		if command != "" {
			if len(command) > 60 {
				command = command[:60] + "..."
			}
			return "Running: " + command
		}
		return "Running command..."
	default:
		if tool != "" {
			return "Using " + tool
		}
		return "Working..."
	}
}

// shortPath returns the last 2 path components for display.
func shortPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}
