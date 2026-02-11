package claude

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ExtractJSON extracts the first JSON object or array from claude output.
func ExtractJSON(output string) (json.RawMessage, error) {
	// Try to find JSON block in markdown code fence
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\n(.*?)```")
	if matches := re.FindStringSubmatch(output); len(matches) > 1 {
		return json.RawMessage(strings.TrimSpace(matches[1])), nil
	}

	// Try to find raw JSON object
	start := strings.IndexAny(output, "{[")
	if start == -1 {
		return nil, nil
	}

	// Find matching closing bracket
	opener := output[start]
	closer := byte('}')
	if opener == '[' {
		closer = ']'
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(output); i++ {
		if escaped {
			escaped = false
			continue
		}
		c := output[i]
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == opener {
			depth++
		} else if c == closer {
			depth--
			if depth == 0 {
				raw := output[start : i+1]
				return json.RawMessage(raw), nil
			}
		}
	}

	return nil, nil
}

// ExtractMarkdownSection extracts content under a specific heading.
func ExtractMarkdownSection(output, heading string) string {
	lines := strings.Split(output, "\n")
	var result strings.Builder
	capturing := false
	headingLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the target heading
		if !capturing {
			if strings.Contains(strings.ToLower(trimmed), strings.ToLower(heading)) && strings.HasPrefix(trimmed, "#") {
				capturing = true
				headingLevel = countPrefix(trimmed, '#')
				continue
			}
		} else {
			// Stop at same or higher level heading
			if strings.HasPrefix(trimmed, "#") {
				level := countPrefix(trimmed, '#')
				if level <= headingLevel {
					break
				}
			}
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String())
}

func countPrefix(s string, char byte) int {
	count := 0
	for i := 0; i < len(s) && s[i] == char; i++ {
		count++
	}
	return count
}
