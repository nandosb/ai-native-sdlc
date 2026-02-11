package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const notionAPIBase = "https://api.notion.com/v1"

// NotionClient interacts with the Notion API.
type NotionClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewNotionClient creates a Notion client from env vars.
func NewNotionClient() *NotionClient {
	return &NotionClient{
		apiKey:     os.Getenv("NOTION_API_KEY"),
		httpClient: &http.Client{},
	}
}

// IsConfigured returns whether the client has valid credentials.
func (nc *NotionClient) IsConfigured() bool {
	return nc.apiKey != ""
}

// ReadPage retrieves page content as markdown-like text.
func (nc *NotionClient) ReadPage(pageID string) (string, error) {
	if !nc.IsConfigured() {
		return "", fmt.Errorf("NOTION_API_KEY not set")
	}

	// Extract page ID from URL if needed
	pageID = extractNotionPageID(pageID)

	// Get page blocks
	url := fmt.Sprintf("%s/blocks/%s/children?page_size=100", notionAPIBase, pageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	nc.setHeaders(req)

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("notion request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("notion HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			Type      string `json:"type"`
			Paragraph *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"paragraph,omitempty"`
			Heading1 *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"heading_1,omitempty"`
			Heading2 *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"heading_2,omitempty"`
			Heading3 *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"heading_3,omitempty"`
			BulletedListItem *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"bulleted_list_item,omitempty"`
			NumberedListItem *struct {
				RichText []notionRichText `json:"rich_text"`
			} `json:"numbered_list_item,omitempty"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse notion response: %w", err)
	}

	var md strings.Builder
	for _, block := range result.Results {
		switch block.Type {
		case "heading_1":
			if block.Heading1 != nil {
				md.WriteString("# " + richTextToString(block.Heading1.RichText) + "\n\n")
			}
		case "heading_2":
			if block.Heading2 != nil {
				md.WriteString("## " + richTextToString(block.Heading2.RichText) + "\n\n")
			}
		case "heading_3":
			if block.Heading3 != nil {
				md.WriteString("### " + richTextToString(block.Heading3.RichText) + "\n\n")
			}
		case "paragraph":
			if block.Paragraph != nil {
				md.WriteString(richTextToString(block.Paragraph.RichText) + "\n\n")
			}
		case "bulleted_list_item":
			if block.BulletedListItem != nil {
				md.WriteString("- " + richTextToString(block.BulletedListItem.RichText) + "\n")
			}
		case "numbered_list_item":
			if block.NumberedListItem != nil {
				md.WriteString("1. " + richTextToString(block.NumberedListItem.RichText) + "\n")
			}
		}
	}

	return md.String(), nil
}

// CreatePage creates a new Notion page with markdown content.
func (nc *NotionClient) CreatePage(parentPageID, title, content string) (string, error) {
	if !nc.IsConfigured() {
		return "", fmt.Errorf("NOTION_API_KEY not set")
	}

	parentPageID = extractNotionPageID(parentPageID)

	payload := map[string]interface{}{
		"parent": map[string]string{
			"page_id": parentPageID,
		},
		"properties": map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"text": map[string]string{
						"content": title,
					},
				},
			},
		},
		"children": markdownToBlocks(content),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", notionAPIBase+"/pages", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	nc.setHeaders(req)

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("notion request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("notion HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.URL, nil
}

func (nc *NotionClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+nc.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
}

type notionRichText struct {
	PlainText string `json:"plain_text"`
}

func richTextToString(texts []notionRichText) string {
	var parts []string
	for _, t := range texts {
		parts = append(parts, t.PlainText)
	}
	return strings.Join(parts, "")
}

func extractNotionPageID(urlOrID string) string {
	// Handle full URLs like https://notion.so/org/Page-Title-abc123def456
	if strings.Contains(urlOrID, "notion.so") {
		parts := strings.Split(urlOrID, "-")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			// Remove query params
			if idx := strings.Index(last, "?"); idx != -1 {
				last = last[:idx]
			}
			if len(last) == 32 {
				return last
			}
		}
		// Try getting the last path segment
		parts = strings.Split(urlOrID, "/")
		last := parts[len(parts)-1]
		if idx := strings.LastIndex(last, "-"); idx != -1 {
			id := last[idx+1:]
			if qIdx := strings.Index(id, "?"); qIdx != -1 {
				id = id[:qIdx]
			}
			return id
		}
	}
	return urlOrID
}

func markdownToBlocks(content string) []map[string]interface{} {
	var blocks []map[string]interface{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			blocks = append(blocks, textBlock("heading_1", strings.TrimPrefix(line, "# ")))
		} else if strings.HasPrefix(line, "## ") {
			blocks = append(blocks, textBlock("heading_2", strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "### ") {
			blocks = append(blocks, textBlock("heading_3", strings.TrimPrefix(line, "### ")))
		} else if strings.HasPrefix(line, "- ") {
			blocks = append(blocks, textBlock("bulleted_list_item", strings.TrimPrefix(line, "- ")))
		} else {
			blocks = append(blocks, textBlock("paragraph", line))
		}
	}

	return blocks
}

func textBlock(blockType, text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   blockType,
		blockType: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]string{
						"content": text,
					},
				},
			},
		},
	}
}
