package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HarborClient struct {
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

func NewHarborClient(baseURL string, username string, password string) *HarborClient {
	// Ensure baseURL has a protocol scheme for HTTP requests.
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	// Strip trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &HarborClient{
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
func (h *HarborClient) getToken(scope string) (string, error) {
	// Return cached token if still valid (with 30s buffer)
	if h.tokenCache.token != "" && time.Now().Before(h.tokenCache.expiry.Add(-30*time.Second)) {
		return h.tokenCache.token, nil
	}

	// If we don't know the realm yet, discover it by hitting /v2/ and reading the challenge
	if h.tokenCache.realm == "" {
		if err := h.discoverAuthRealm(); err != nil {
			return "", fmt.Errorf("discover auth realm: %w", err)
		}
	}

	// Build token request URL
	url := h.tokenCache.realm
	if scope != "" {
		url += "?service=" + h.tokenCache.service + "&scope=" + scope
	} else {
		url += "?service=" + h.tokenCache.service
	}

	req, _ := http.NewRequest("GET", url, nil)
	if h.username != "" {
		req.SetBasicAuth(h.username, h.password)
	}

	resp, err := h.client.Do(req)
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
	h.tokenCache.token = token
	h.tokenCache.expiry = time.Now().Add(time.Duration(expirySeconds) * time.Second)

	return token, nil
}

// discoverAuthRealm hits the registry /v2/ endpoint, gets a 401,
// and parses the WWW-Authenticate header to find the Bearer realm and service.
func (h *HarborClient) discoverAuthRealm() error {
	req, _ := http.NewRequest("GET", h.baseURL+"/v2/", nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("hit /v2/ endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// No auth needed — token realm empty
		h.tokenCache.realm = ""
		h.tokenCache.service = ""
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
	h.tokenCache.realm = realm
	h.tokenCache.service = service
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
func (h *HarborClient) VerifyImageExists(project, imageName, tag string) bool {
	// Docker V2 API uses <project>/<imageName> as the repository name
	repoName := project + "/" + imageName
	scope := "repository:" + repoName + ":pull"

	token, err := h.getToken(scope)
	if err != nil {
		fmt.Printf("Error getting auth token: %v\n", err)
		return false
	}

	url := fmt.Sprintf("%s/v2/%s/manifests/%s", h.baseURL, repoName, tag)
	req, _ := http.NewRequest("HEAD", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Accept both v2 and v1 manifest schemas
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := h.client.Do(req)
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
		resp2, err := h.client.Do(req)
		if err != nil {
			return false
		}
		defer resp2.Body.Close()
		return resp2.StatusCode == 200
	}

	fmt.Printf("Image %s/%s:%s not found (status %d)\n", project, imageName, tag, resp.StatusCode)
	return false
}

// GetImageTags is kept for backward compatibility but now uses VerifyImageExists internally.
// It's no longer used by deployService directly.
func (h *HarborClient) GetImageTags(project, repository string) ([]HarborTag, error) {
	return nil, fmt.Errorf("GetImageTags is deprecated, use VerifyImageExists instead")
}

type HarborTag struct {
	Name string `json:"name"`
}