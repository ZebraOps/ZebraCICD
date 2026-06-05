package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// V2RegistryAdapter implements the RegistryClient interface using the
// Docker Registry V2 HTTP API. This works with any registry (Harbor,
// Aliyun ACR, etc.) that supports the V2 API specification.
type V2RegistryAdapter struct {
	baseURL   string // e.g. "https://registry.cn-shanghai.aliyuncs.com"
	username  string
	password  string
	client    *http.Client
	tokenCache struct {
		token    string
		expiry   time.Time
		realm    string
		service  string
	}
}

func NewV2RegistryAdapter(baseURL string, username string, password string) *V2RegistryAdapter {
	// Ensure baseURL has a protocol scheme for HTTP requests.
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	// Strip trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &V2RegistryAdapter{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// tokenResponse represents the JSON response from a Docker registry token endpoint.
type tokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// getToken obtains a Bearer token from the registry's auth endpoint.
// Uses the WWW-Authenticate challenge from a 401 response to discover the token realm.
func (a *V2RegistryAdapter) getToken(scope string) (string, error) {
	// Return cached token if still valid (with 30s buffer)
	if a.tokenCache.token != "" && time.Now().Before(a.tokenCache.expiry.Add(-30*time.Second)) {
		return a.tokenCache.token, nil
	}

	// If we don't know the realm yet, discover it by hitting /v2/ and reading the challenge
	if a.tokenCache.realm == "" {
		if err := a.discoverAuthRealm(); err != nil {
			return "", fmt.Errorf("discover auth realm: %w", err)
		}
	}

	// Build token request URL
	url := a.tokenCache.realm
	if scope != "" {
		url += "?service=" + a.tokenCache.service + "&scope=" + scope
	} else {
		url += "?service=" + a.tokenCache.service
	}

	req, _ := http.NewRequest("GET", url, nil)
	if a.username != "" {
		req.SetBasicAuth(a.username, a.password)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	token := tr.Token
	if token == "" {
		token = tr.AccessToken
	}

	// Cache the token with expiry
	expirySeconds := tr.ExpiresIn
	if expirySeconds <= 0 {
		expirySeconds = 300 // default 5 min
	}
	a.tokenCache.token = token
	a.tokenCache.expiry = time.Now().Add(time.Duration(expirySeconds) * time.Second)

	return token, nil
}

// discoverAuthRealm hits the registry /v2/ endpoint, gets a 401,
// and parses the WWW-Authenticate header to find the Bearer realm and service.
func (a *V2RegistryAdapter) discoverAuthRealm() error {
	req, _ := http.NewRequest("GET", a.baseURL+"/v2/", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("hit /v2/ endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// No auth needed — token realm empty
		a.tokenCache.realm = ""
		a.tokenCache.service = ""
		return nil
	}

	if resp.StatusCode != 401 {
		return fmt.Errorf("/v2/ returned %d, expected 401 or 200", resp.StatusCode)
	}

	// Parse WWW-Authenticate: Bearer realm="...",service="..."
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("401 with no WWW-Authenticate header")
	}

	realm, service := parseBearerChallenge(authHeader)
	a.tokenCache.realm = realm
	a.tokenCache.service = service
	return nil
}

// parseBearerChallenge extracts realm and service from a Bearer challenge header.
// Example: Bearer realm="https://dockerauth.cn-hangzhou.aliyuncs.com/auth",service="registry.aliyuncs.com:cn-shanghai:26842"
func parseBearerChallenge(header string) (realm string, service string) {
	// Remove "Bearer " prefix
	s := strings.TrimPrefix(header, "Bearer ")
	s = strings.TrimSpace(s)

	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		switch key {
		case "realm":
			realm = val
		case "service":
			service = val
		}
	}
	return realm, service
}

// VerifyImageExists checks if a specific image tag exists in the registry
// using the Docker Registry V2 API (manifest query).
// This works with any registry (Harbor, Aliyun ACR, etc.) that supports the V2 API.
func (a *V2RegistryAdapter) VerifyImageExists(project, imageName, tag string) bool {
	// Docker V2 API uses <project>/<imageName> as the repository name
	repoName := project + "/" + imageName
	scope := "repository:" + repoName + ":pull"

	token, err := a.getToken(scope)
	if err != nil {
		fmt.Printf("Error getting auth token: %v\n", err)
		return false
	}

	url := fmt.Sprintf("%s/v2/%s/manifests/%s", a.baseURL, repoName, tag)
	req, _ := http.NewRequest("HEAD", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Accept both v2 and v1 manifest schemas
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Printf("Error verifying image %s/%s:%s: %v\n", project, imageName, tag, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true
	}

	// If HEAD returns 404, try GET — some registries don't support HEAD on manifests
	if resp.StatusCode == 404 {
		req.Method = "GET"
		resp2, err := a.client.Do(req)
		if err != nil {
			return false
		}
		defer resp2.Body.Close()
		return resp2.StatusCode == 200
	}

	fmt.Printf("Image %s/%s:%s not found (status %d)\n", project, imageName, tag, resp.StatusCode)
	return false
}

// ListTags returns the list of tags for a given repository using the Docker Registry V2 API.
// Endpoint: GET /v2/{project}/{imageName}/tags/list
func (a *V2RegistryAdapter) ListTags(project, imageName string) ([]string, error) {
	repoName := project + "/" + imageName
	scope := "repository:" + repoName + ":pull"

	token, err := a.getToken(scope)
	if err != nil {
		return nil, fmt.Errorf("error getting auth token for tag listing: %w", err)
	}

	url := fmt.Sprintf("%s/v2/%s/tags/list", a.baseURL, repoName)
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error listing tags for %s: %w", repoName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tag list request returned %d: %s", resp.StatusCode, string(body))
	}

	var tagList struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagList); err != nil {
		return nil, fmt.Errorf("error decoding tag list response: %w", err)
	}

	return tagList.Tags, nil
}

// EnsureProjectExists is a no-op for standard V2 registries.
// Projects/repositories are created implicitly on first docker push.
func (a *V2RegistryAdapter) EnsureProjectExists(project string) error {
	return nil
}

// Ping tests the connection to the registry by hitting the /v2/ endpoint.
func (a *V2RegistryAdapter) Ping() error {
	req, _ := http.NewRequest("GET", a.baseURL+"/v2/", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	// 200 means no auth needed, 401 means auth is required but endpoint exists
	if resp.StatusCode == 200 || resp.StatusCode == 401 {
		return nil
	}
	return fmt.Errorf("连接失败: HTTP %d", resp.StatusCode)
}