package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const linearAPIBase = "https://api.linear.app/graphql"

// LinearClient interacts with the Linear API.
type LinearClient struct {
	apiKey     string
	httpClient *http.Client
}

// LinearIssue represents an issue to create.
type LinearIssue struct {
	Title       string
	Description string
	Team        string
	Estimate    string
	Labels      []string
}

// NewLinearClient creates a Linear client from env vars.
func NewLinearClient() *LinearClient {
	return &LinearClient{
		apiKey:     os.Getenv("LINEAR_API_KEY"),
		httpClient: &http.Client{},
	}
}

// IsConfigured returns whether the client has valid credentials.
func (lc *LinearClient) IsConfigured() bool {
	return lc.apiKey != ""
}

// CreateIssue creates a new issue and returns its identifier.
func (lc *LinearClient) CreateIssue(issue LinearIssue) (string, error) {
	if !lc.IsConfigured() {
		return "", fmt.Errorf("LINEAR_API_KEY not set")
	}

	// First, resolve team ID
	teamID, err := lc.resolveTeamID(issue.Team)
	if err != nil {
		return "", fmt.Errorf("resolve team: %w", err)
	}

	query := `mutation CreateIssue($input: IssueCreateInput!) {
		issueCreate(input: $input) {
			success
			issue {
				id
				identifier
			}
		}
	}`

	input := map[string]interface{}{
		"title":       issue.Title,
		"description": issue.Description,
		"teamId":      teamID,
	}

	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool `json:"success"`
				Issue   struct {
					ID         string `json:"id"`
					Identifier string `json:"identifier"`
				} `json:"issue"`
			} `json:"issueCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := lc.graphql(query, map[string]interface{}{"input": input}, &resp); err != nil {
		return "", err
	}
	if len(resp.Errors) > 0 {
		return "", fmt.Errorf("linear error: %s", resp.Errors[0].Message)
	}

	return resp.Data.IssueCreate.Issue.Identifier, nil
}

// LinearIssueDetails holds the fetched details of a Linear issue.
type LinearIssueDetails struct {
	Identifier  string
	Title       string
	Description string
}

// GetIssueByIdentifier fetches an issue's title and description by its
// identifier (e.g. "IGN-63"). The identifier is split into team key and
// issue number, then queried via GraphQL.
func (lc *LinearClient) GetIssueByIdentifier(identifier string) (*LinearIssueDetails, error) {
	if !lc.IsConfigured() {
		return nil, fmt.Errorf("LINEAR_API_KEY not set")
	}

	// Split "TEAM-123" into team key and number
	parts := strings.SplitN(identifier, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid identifier format: %s", identifier)
	}
	teamKey := parts[0]
	number, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number in identifier %s: %w", identifier, err)
	}

	query := `query GetIssue($filter: IssueFilter) {
		issues(filter: $filter, first: 1) {
			nodes {
				identifier
				title
				description
			}
		}
	}`

	filter := map[string]interface{}{
		"number": map[string]interface{}{
			"eq": number,
		},
		"team": map[string]interface{}{
			"key": map[string]interface{}{
				"eq": teamKey,
			},
		},
	}

	var resp struct {
		Data struct {
			Issues struct {
				Nodes []struct {
					Identifier  string `json:"identifier"`
					Title       string `json:"title"`
					Description string `json:"description"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := lc.graphql(query, map[string]interface{}{"filter": filter}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("linear error: %s", resp.Errors[0].Message)
	}
	if len(resp.Data.Issues.Nodes) == 0 {
		return nil, fmt.Errorf("issue not found: %s", identifier)
	}

	node := resp.Data.Issues.Nodes[0]
	return &LinearIssueDetails{
		Identifier:  node.Identifier,
		Title:       node.Title,
		Description: node.Description,
	}, nil
}

// SearchIssueByTitle searches for an existing issue by title.
func (lc *LinearClient) SearchIssueByTitle(teamName, title string) (string, bool, error) {
	if !lc.IsConfigured() {
		return "", false, nil
	}

	query := `query SearchIssues($filter: IssueFilter) {
		issues(filter: $filter, first: 1) {
			nodes {
				id
				identifier
				title
			}
		}
	}`

	filter := map[string]interface{}{
		"title": map[string]interface{}{
			"eq": title,
		},
	}

	var resp struct {
		Data struct {
			Issues struct {
				Nodes []struct {
					ID         string `json:"id"`
					Identifier string `json:"identifier"`
					Title      string `json:"title"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
	}

	if err := lc.graphql(query, map[string]interface{}{"filter": filter}, &resp); err != nil {
		return "", false, err
	}

	if len(resp.Data.Issues.Nodes) > 0 {
		return resp.Data.Issues.Nodes[0].Identifier, true, nil
	}
	return "", false, nil
}

// CreateRelation creates a blocking relationship between two issues.
func (lc *LinearClient) CreateRelation(blockingID, blockedID, relationType string) error {
	if !lc.IsConfigured() {
		return nil
	}

	query := `mutation CreateRelation($input: IssueRelationCreateInput!) {
		issueRelationCreate(input: $input) {
			success
		}
	}`

	input := map[string]interface{}{
		"issueId":    blockedID,
		"relatedIssueId": blockingID,
		"type":       "blocks",
	}

	var resp struct {
		Data struct {
			IssueRelationCreate struct {
				Success bool `json:"success"`
			} `json:"issueRelationCreate"`
		} `json:"data"`
	}

	return lc.graphql(query, map[string]interface{}{"input": input}, &resp)
}

// UpdateIssueStatus updates an issue's state.
func (lc *LinearClient) UpdateIssueStatus(issueID, stateName string) error {
	if !lc.IsConfigured() {
		return nil
	}

	query := `mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
		issueUpdate(id: $id, input: $input) {
			success
		}
	}`

	// This is simplified — in production, resolve state ID first
	input := map[string]interface{}{
		"stateId": stateName,
	}

	var resp struct{}
	return lc.graphql(query, map[string]interface{}{"id": issueID, "input": input}, &resp)
}

func (lc *LinearClient) resolveTeamID(teamName string) (string, error) {
	query := `query Teams {
		teams {
			nodes {
				id
				name
				key
			}
		}
	}`

	var resp struct {
		Data struct {
			Teams struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Key  string `json:"key"`
				} `json:"nodes"`
			} `json:"teams"`
		} `json:"data"`
	}

	if err := lc.graphql(query, nil, &resp); err != nil {
		return "", err
	}

	for _, t := range resp.Data.Teams.Nodes {
		if t.Name == teamName || t.Key == teamName {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("team not found: %s", teamName)
}

func (lc *LinearClient) graphql(query string, variables map[string]interface{}, result interface{}) error {
	body := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		body["variables"] = variables
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", linearAPIBase, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", lc.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("linear request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("linear HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return json.Unmarshal(respBody, result)
}
