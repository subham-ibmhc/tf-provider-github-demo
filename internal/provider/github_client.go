package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
	"fmt"
	"net/http"
)

//github repo
type Repository struct {
	ID	int64 `json:"id"`
	Name string `json:"name"`
	FullName string `json:"full_name"`
	Description string `json:"description"`
	Private bool `json:"private"`
	HasIssues bool `json:"has_issues"`
	HasWiki bool `json:"has_wiki"`
	AutoInit bool `json:"auto_init,omitempty"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// create repo request
type CreateRepositoryRequest struct {
	Name string `json:"name"`
	Description string `json:"description"`
	Private bool `json:"private"`
	HasIssues bool `json:"has_issues"`
	HasWiki bool `json:"has_wiki"`
	AutoInit bool `json:"auto_init,omitempty"`
}

// update repo request
type UpdateRepositoryRequest struct {
	Name string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Private bool `json:"private"`
	HasIssues bool `json:"has_issues"`
	HasWiki bool `json:"has_wiki"`
}

// api client
type GitHubClient struct {
	httpClient *http.Client
	token string
	baseUrl string
}

// for creating new Github client
func NewGithubClient(token string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
		baseUrl: "https://api.github.com",
	}
}

func (c *GitHubClient) genericRequest(ctx context.Context, httpMethod, path string, body interface{}) (*http.Response, error) {
	var reqBody bytes.Buffer

	if body != nil {
		err := json.NewEncoder(&reqBody).Encode(body)
		if err!=nil {
			return nil, fmt.Errorf("failed to encode req body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, c.baseUrl+path, &reqBody)

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err!=nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}


// make get, create, update, delete calls

func (c *GitHubClient) GetRepo(ctx context.Context, owner, name string) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)
	resp, err := c.genericRequest(ctx, "GET", path, nil)
	if err!=nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repo: HTTP %d", resp.StatusCode)
	}

	var repo Repository
	e := json.NewDecoder(resp.Body).Decode(&repo)
	if e != nil {
		return nil, fmt.Errorf("failed to decode response: %w", e)
	}

	return &repo, nil
}

func (c *GitHubClient) CreateRepo(ctx context.Context, req *CreateRepositoryRequest) (*Repository, error) {
	resp, err := c.genericRequest(ctx, "POST", "/user/repos", req)
	if err!=nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated{
		return nil, fmt.Errorf("repository not created: HTTP %d", resp.StatusCode)
	}

	var repo Repository
	e := json.NewDecoder(resp.Body).Decode(&repo)
	if e != nil {
		return nil, fmt.Errorf("failed to decode response: %w", e)
	}

	return &repo, nil
}

func (c *GitHubClient) UpdateRepo(ctx context.Context, owner, name string, req *UpdateRepositoryRequest) (*Repository, error) {
	path:= fmt.Sprintf("/repos/%s/%s", owner, name)
	resp, err := c.genericRequest(ctx, "PATCH", c.baseUrl+path, req)

	if err!=nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update repo: HTTP %d", resp.StatusCode)
	}

	var repo Repository
	e := json.NewDecoder(resp.Body).Decode(&repo)
	if e!=nil {
		return nil, fmt.Errorf("failed to decode response: %w", e)
	}

	return &repo, nil
}

func (c *GitHubClient) DeleteRepo(ctx context.Context, owner, name string) error {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)

	resp, err := c.genericRequest(ctx, "DELETE", path, nil)

	if err !=nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete repo: HTTP %d", resp.StatusCode)
	}

	return nil
}

