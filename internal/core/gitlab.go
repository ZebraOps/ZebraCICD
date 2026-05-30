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

// --- GitLab Client (existing) ---
// (kept in this file for reference, actual code unchanged)

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab API error: status=%d, url=%s, body=%s", resp.StatusCode, u, string(body))
	}

	var branches []types.Branch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return branches, nil
}

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

// --- Git Platform Connectivity Test ---
// Generic HTTP-based connectivity test for any Git platform

// TestGitPlatformConnectivity 测试Git平台连通性
// 根据 platform_type 从平台地址推算 API 地址，验证连通性和认证配置
func TestGitPlatformConnectivity(platformURL, platformType, authType, authConfig string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	testURL := resolveTestURL(platformURL, platformType)

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("构造请求失败: %v", err)
	}

	// 根据认证方式设置请求头
	if err := setAuthHeaders(req, platformType, authType, authConfig); err != nil {
		return fmt.Errorf("解析认证配置失败: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体（用于调试）
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("认证失败 (HTTP %d): 请检查认证配置是否正确", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("请求失败 (HTTP %d): %s", resp.StatusCode, truncateString(string(body), 200))
	}

	// 200-299 视为连通成功
	return nil
}

// resolveTestURL 根据平台类型从平台地址推算测试 API 地址
func resolveTestURL(platformURL, platformType string) string {
	baseURL := normalizeURL(platformURL)
	switch platformType {
	case "gitlab":
		return baseURL + "/api/v4/user"
	case "github":
		// GitHub 公开 API 固定地址；企业版使用平台地址 + /api/v3
		if baseURL == "https://github.com" || baseURL == "http://github.com" {
			return "https://api.github.com/user"
		}
		return baseURL + "/api/v3/user"
	case "gitea":
		return baseURL + "/api/v1/user"
	default:
		return baseURL + "/user"
	}
}

// setAuthHeaders 设置认证请求头（仅支持 Token）
func setAuthHeaders(req *http.Request, platformType, authType, authConfig string) error {
	if authType == "" || authConfig == "" {
		return nil
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(authConfig), &config); err != nil {
		return fmt.Errorf("auth_config JSON解析失败: %v", err)
	}

	token := config["token"]
	if token == "" {
		return fmt.Errorf("token 为空")
	}

	switch platformType {
	case "github":
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		// GitLab / Gitea 使用 PRIVATE-TOKEN
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	return nil
}

// normalizeURL 确保 URL 格式正确
func normalizeURL(rawURL string) string {
	u := rawURL
	// 去掉末尾斜杠
	u = stringsTrimSuffix(u, "/")
	return u
}

func stringsTrimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}