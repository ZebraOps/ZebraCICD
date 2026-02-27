package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type GitLabClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewGitLabClient(baseURL, token string) *GitLabClient {
	return &GitLabClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (g *GitLabClient) GetBranches(projectPath string) ([]types.Branch, error) {
	// projectPath should be URL-encoded full path like "group%2Frepo" or numeric ID
	u := fmt.Sprintf("%s/api/v4/projects/%s/repository/branches", g.baseURL, url.PathEscape(projectPath))
	req, _ := http.NewRequest("GET", u, nil)
	if g.token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 读取错误响应体用于调试
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab API error: status=%d, url=%s, body=%s", resp.StatusCode, u, string(body))
	}

	var branches []types.Branch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return branches, nil
}

// GetProject 获取项目信息
func (g *GitLabClient) GetProject(repoID string) (*types.Project, error) {
	u := fmt.Sprintf("%s/api/v4/projects/%s", g.baseURL, url.PathEscape(repoID))

	req, _ := http.NewRequest("GET", u, nil)
	if g.token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response Body: %s\n", string(body))
		return nil, fmt.Errorf("gitlab API error: status=%d, url=%s, body=%s", resp.StatusCode, u, string(body))
	}

	var project types.Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode project response: %v", err)
	}
	return &project, nil
}

func (g *GitLabClient) GetTags(projectPath string) ([]types.Tag, error) {
	u := fmt.Sprintf("%s/api/v4/projects/%s/repository/tags", g.baseURL, url.PathEscape(projectPath))
	req, _ := http.NewRequest("GET", u, nil)
	if g.token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tags []types.Tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	return tags, nil
}
