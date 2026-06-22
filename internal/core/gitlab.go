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

// NewGitLabClientWithTimeout 创建带自定义超时配置的GitLab客户端
func NewGitLabClientWithTimeout(baseURL, token string, timeout time.Duration) *GitLabClient {
	return &GitLabClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: timeout,
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

	// Gitee 认证通过 URL 查询参数传递
	if platformType == "gitee" {
		if token := extractToken(authConfig); token != "" {
			testURL = testURL + "?access_token=" + token
		}
	}

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
	case "gitee":
		return "https://gitee.com/api/v5/user"
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
	case "gitee":
		// Gitee uses query parameter ?access_token=, not HTTP headers
		return nil
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

// extractToken 从 AuthConfig JSON 中提取 token
func extractToken(authConfig string) string {
	var config map[string]string
	if err := json.Unmarshal([]byte(authConfig), &config); err != nil {
		return ""
	}
	return config["token"]
}

// FetchPlatformProjects 从Git平台获取项目列表，支持搜索
func FetchPlatformProjects(platformURL, platformType, authType, authConfig, search string, page, size int) ([]types.Project, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	var apiURL string
	baseURL := normalizeURL(platformURL)

	switch platformType {
	case "gitlab":
		apiURL = fmt.Sprintf("%s/api/v4/projects?membership=true&per_page=%d&page=%d", baseURL, size, page)
		if search != "" {
			apiURL += "&search=" + url.QueryEscape(search)
		}
	case "github":
		if baseURL == "https://github.com" || baseURL == "http://github.com" {
			apiURL = fmt.Sprintf("https://api.github.com/user/repos?per_page=%d&page=%d", size, page)
		} else {
			apiURL = fmt.Sprintf("%s/api/v3/user/repos?per_page=%d&page=%d", baseURL, size, page)
		}
		if search != "" {
			// GitHub doesn't support search in user/repos; use search/repos endpoint
			if baseURL == "https://github.com" || baseURL == "http://github.com" {
				apiURL = fmt.Sprintf("https://api.github.com/search/repositories?q=%s+user:@me&per_page=%d&page=%d", url.QueryEscape(search), size, page)
			} else {
				apiURL = fmt.Sprintf("%s/api/v3/search/repositories?q=%s&per_page=%d&page=%d", baseURL, url.QueryEscape(search), size, page)
			}
		}
	case "gitea":
		apiURL = fmt.Sprintf("%s/api/v1/repos/search?limit=%d&page=%d", baseURL, size, page)
		if search != "" {
			apiURL += "&q=" + url.QueryEscape(search)
		}
	case "gitee":
		// Gitee OpenAPI v5
		apiBase := "https://gitee.com/api/v5"
		if baseURL != "https://gitee.com" && baseURL != "http://gitee.com" {
			apiBase = baseURL + "/api/v5"
		}
		if search != "" {
			apiURL = fmt.Sprintf("%s/search/repositories?q=%s&page=%d&per_page=%d", apiBase, url.QueryEscape(search), page, size)
		} else {
			apiURL = fmt.Sprintf("%s/user/repos?page=%d&per_page=%d&type=all", apiBase, page, size)
		}
		// Gitee 认证通过 URL 查询参数
		if token := extractToken(authConfig); token != "" {
			apiURL = apiURL + "&access_token=" + token
		}
	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %v", err)
	}

	if err := setAuthHeaders(req, platformType, authType, authConfig); err != nil {
		return nil, fmt.Errorf("解析认证配置失败: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败 (HTTP %d): %s", resp.StatusCode, truncateString(string(body), 200))
	}

	// 根据平台类型解析不同的响应格式
	switch platformType {
	case "gitlab":
		var projects []types.Project
		if err := json.Unmarshal(body, &projects); err != nil {
			return nil, fmt.Errorf("解析GitLab项目列表失败: %v", err)
		}
		return projects, nil

	case "github":
		if search != "" && (baseURL == "https://github.com" || baseURL == "http://github.com") {
			// GitHub search/repositories returns { total_count, items: [...] }
			var result struct {
				Items []struct {
					FullName    string `json:"full_name"`
					Name        string `json:"name"`
					HTMLURL     string `json:"html_url"`
					SSHURL      string `json:"ssh_url"`
					Description string `json:"description"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("解析GitHub搜索结果失败: %v", err)
			}
			projects := make([]types.Project, len(result.Items))
			for i, item := range result.Items {
				projects[i] = types.Project{
					Path:          item.FullName,
					Name:          item.Name,
					HTTPURLToRepo: item.HTMLURL,
					SSHURLToRepo:  item.SSHURL,
					Desc:          item.Description,
				}
			}
			return projects, nil
		}
		// GitHub user/repos returns array directly
		var repos []struct {
			FullName    string `json:"full_name"`
			Name        string `json:"name"`
			HTMLURL     string `json:"html_url"`
			SSHURL      string `json:"ssh_url"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, fmt.Errorf("解析GitHub仓库列表失败: %v", err)
		}
		projects := make([]types.Project, len(repos))
		for i, repo := range repos {
			projects[i] = types.Project{
				Path:          repo.FullName,
				Name:          repo.Name,
				HTTPURLToRepo: repo.HTMLURL,
				SSHURLToRepo:  repo.SSHURL,
				Desc:          repo.Description,
			}
		}
		return projects, nil

	case "gitea":
		// Gitea search returns { ok, data: [...] }
		var result struct {
			Data []struct {
				FullName    string `json:"full_name"`
				Name        string `json:"name"`
				HTMLURL     string `json:"html_url"`
				SSHURL      string `json:"ssh_url"`
				Description string `json:"description"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("解析Gitea项目列表失败: %v", err)
		}
		projects := make([]types.Project, len(result.Data))
		for i, item := range result.Data {
			projects[i] = types.Project{
				Path:          item.FullName,
				Name:          item.Name,
				HTTPURLToRepo: item.HTMLURL,
				SSHURLToRepo:  item.SSHURL,
				Desc:          item.Description,
			}
		}
		return projects, nil

	case "gitee":
		if search != "" {
			// Gitee search returns { items: [...] } same as GitHub
			var result struct {
				Items []struct {
					FullName    string `json:"full_name"`
					Name        string `json:"name"`
					HTMLURL     string `json:"html_url"`
					SSHURL      string `json:"ssh_url"`
					Description string `json:"description"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("解析Gitee搜索结果失败: %v", err)
			}
			projects := make([]types.Project, len(result.Items))
			for i, item := range result.Items {
				projects[i] = types.Project{
					Path:          item.FullName,
					Name:          item.Name,
					HTTPURLToRepo: item.HTMLURL,
					SSHURLToRepo:  item.SSHURL,
					Desc:          item.Description,
				}
			}
			return projects, nil
		}
		// Gitee user/repos returns flat array same as GitLab
		var repos []struct {
			FullName    string `json:"full_name"`
			Name        string `json:"name"`
			HTMLURL     string `json:"html_url"`
			SSHURL      string `json:"ssh_url"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, fmt.Errorf("解析Gitee仓库列表失败: %v", err)
		}
		projects := make([]types.Project, len(repos))
		for i, repo := range repos {
			projects[i] = types.Project{
				Path:          repo.FullName,
				Name:          repo.Name,
				HTTPURLToRepo: repo.HTMLURL,
				SSHURLToRepo:  repo.SSHURL,
				Desc:          repo.Description,
			}
		}
		return projects, nil
	}

	return nil, nil
}

// ─── 多平台分支/标签查询 ──────────────────────────────────────────

// FetchBranches 根据平台类型获取仓库分支列表
// repoPath: GitLab 为项目 ID/路径, GitHub/Gitea 为 owner/repo
func FetchBranches(platformURL, platformType, authType, authConfig, repoPath string) ([]types.Branch, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	baseURL := normalizeURL(platformURL)

	var apiURL string
	switch platformType {
	case "gitlab":
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/branches?per_page=100", baseURL, url.PathEscape(repoPath))
	case "github":
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/branches?per_page=100", repoPath)
		if baseURL != "https://github.com" && baseURL != "http://github.com" {
			apiURL = fmt.Sprintf("%s/api/v3/repos/%s/branches?per_page=100", baseURL, repoPath)
		}
	case "gitea":
		apiURL = fmt.Sprintf("%s/api/v1/repos/%s/branches?limit=100", baseURL, repoPath)
	case "gitee":
		// Gitee OpenAPI v5
		apiBase := "https://gitee.com/api/v5"
		if baseURL != "https://gitee.com" && baseURL != "http://gitee.com" {
			apiBase = baseURL + "/api/v5"
		}
		apiURL = fmt.Sprintf("%s/repos/%s/branches", apiBase, repoPath)
		if token := extractToken(authConfig); token != "" {
			apiURL = apiURL + "?access_token=" + token
		}
	default:
		// 回退到 GitLab 兼容路径
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/branches?per_page=100", baseURL, url.PathEscape(repoPath))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %v", err)
	}
	if err := setAuthHeaders(req, platformType, authType, authConfig); err != nil {
		return nil, fmt.Errorf("解析认证配置失败: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取分支失败 (HTTP %d): %s", resp.StatusCode, truncateString(string(body), 200))
	}

	var branches []types.Branch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("解析分支列表失败: %v", err)
	}
	return branches, nil
}

// FetchTags 根据平台类型获取仓库标签列表
func FetchTags(platformURL, platformType, authType, authConfig, repoPath string) ([]types.Tag, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	baseURL := normalizeURL(platformURL)

	var apiURL string
	switch platformType {
	case "gitlab":
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/tags?per_page=100", baseURL, url.PathEscape(repoPath))
	case "github":
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100", repoPath)
		if baseURL != "https://github.com" && baseURL != "http://github.com" {
			apiURL = fmt.Sprintf("%s/api/v3/repos/%s/tags?per_page=100", baseURL, repoPath)
		}
	case "gitea":
		apiURL = fmt.Sprintf("%s/api/v1/repos/%s/tags?limit=100", baseURL, repoPath)
	case "gitee":
		// Gitee OpenAPI v5
		apiBase := "https://gitee.com/api/v5"
		if baseURL != "https://gitee.com" && baseURL != "http://gitee.com" {
			apiBase = baseURL + "/api/v5"
		}
		apiURL = fmt.Sprintf("%s/repos/%s/tags", apiBase, repoPath)
		if token := extractToken(authConfig); token != "" {
			apiURL = apiURL + "?access_token=" + token
		}
	default:
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/tags?per_page=100", baseURL, url.PathEscape(repoPath))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %v", err)
	}
	if err := setAuthHeaders(req, platformType, authType, authConfig); err != nil {
		return nil, fmt.Errorf("解析认证配置失败: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取标签失败 (HTTP %d): %s", resp.StatusCode, truncateString(string(body), 200))
	}

	var tags []types.Tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("解析标签列表失败: %v", err)
	}
	return tags, nil
}