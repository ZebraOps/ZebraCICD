package core

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ACRAdapter implements the RegistryClient interface for Alibaba Cloud Container Registry (ACR).
// It uses the ACR OpenAPI for project/namespace management and the V2 API for image operations.
type ACRAdapter struct {
	baseURL   string // V2 registry URL (e.g. "registry.cn-shanghai.aliyuncs.com")
	username  string // Docker login username
	password  string // Docker login password
	accessKey string // Alibaba Cloud AccessKey ID for OpenAPI
	secretKey string // Alibaba Cloud AccessKey Secret for OpenAPI
	region    string // ACR region (e.g. "cn-shanghai")
	v2        *V2RegistryAdapter
}

func NewACRAdapter(baseURL string, username string, password string) *ACRAdapter {
	// Extract region from URL: registry.cn-shanghai.aliyuncs.com → cn-shanghai
	region := extractACRRegion(baseURL)
	return &ACRAdapter{
		baseURL:  baseURL,
		username: username,
		password: password,
		region:   region,
		v2:       NewV2RegistryAdapter(baseURL, username, password),
	}
}

func NewACRAdapterWithKeys(baseURL, username, password, accessKey, secretKey string) *ACRAdapter {
	region := extractACRRegion(baseURL)
	return &ACRAdapter{
		baseURL:   baseURL,
		username:  username,
		password:  password,
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		v2:        NewV2RegistryAdapter(baseURL, username, password),
	}
}

// extractACRRegion parses the region from an ACR registry URL.
// e.g. "registry.cn-shanghai.aliyuncs.com" → "cn-shanghai"
func extractACRRegion(rawURL string) string {
	u := strings.TrimPrefix(rawURL, "https://")
	u = strings.TrimPrefix(u, "http://")
	parts := strings.Split(u, ".")
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "registry") {
		return parts[1]
	}
	return "cn-shanghai" // default
}

// VerifyImageExists uses the V2 API to verify image existence (ACR supports V2).
func (a *ACRAdapter) VerifyImageExists(project, imageName, tag string) bool {
	return a.v2.VerifyImageExists(project, imageName, tag)
}

// ListTags uses the V2 API to list tags.
func (a *ACRAdapter) ListTags(project, imageName string) ([]string, error) {
	return a.v2.ListTags(project, imageName)
}

// EnsureProjectExists checks if the ACR namespace exists and creates it if needed.
// ACR requires a namespace to be created before pushing images.
func (a *ACRAdapter) EnsureProjectExists(project string) error {
	if a.accessKey == "" || a.secretKey == "" {
		// No AccessKey/SecretKey configured — cannot manage namespaces via OpenAPI.
		// Fallback: assume the namespace already exists (common for pre-configured environments).
		fmt.Printf("ACR: AccessKey/SecretKey not configured, skipping namespace creation for %s\n", project)
		return nil
	}

	// Check if namespace already exists
	exists, err := a.namespaceExists(project)
	if err != nil {
		fmt.Printf("ACR: error checking namespace %s: %v, assuming it exists\n", project, err)
		return nil // Don't block deployment if we can't check
	}
	if exists {
		fmt.Printf("ACR: namespace %s already exists\n", project)
		return nil
	}

	// Create the namespace
	return a.createNamespace(project)
}

// namespaceExists checks if a namespace exists in ACR via OpenAPI.
func (a *ACRAdapter) namespaceExists(namespace string) (bool, error) {
	params := map[string]string{
		"Action":   "GetNamespace",
		"Namespace": namespace,
		"RegionId": a.region,
	}

	body, statusCode, err := a.callOpenAPI("GET", params)
	if err != nil {
		return false, err
	}

	if statusCode == 200 {
		var result struct {
			Code string `json:"Code"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			// ACR returns Code="" for success, "NamespaceNotExist" if not found
			if result.Code == "" || result.Code == "SUCCESS" {
				return true, nil
			}
		}
		return true, nil // If we got 200, the namespace exists
	}

	if statusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status %d checking namespace: %s", statusCode, string(body))
}

// createNamespace creates a namespace in ACR via OpenAPI.
func (a *ACRAdapter) createNamespace(namespace string) error {
	params := map[string]string{
		"Action":           "CreateNamespace",
		"Namespace":        namespace,
		"RegionId":         a.region,
		"AutoCreateRepo":   "true",   // Automatically create repo on first push
		"DefaultRepoType":  "PUBLIC", // Default visibility (can be overridden)
	}

	_, statusCode, err := a.callOpenAPI("POST", params)
	if err != nil {
		return fmt.Errorf("ACR create namespace API error: %w", err)
	}

	if statusCode == 200 || statusCode == 201 {
		fmt.Printf("ACR: namespace %s created successfully\n", namespace)
		return nil
	}

	// Namespace might already exist (concurrent creation)
	if statusCode == 409 {
		fmt.Printf("ACR: namespace %s already exists (concurrent creation)\n", namespace)
		return nil
	}

	return fmt.Errorf("ACR create namespace returned status %d", statusCode)
}

// callOpenAPI makes a signed call to the Alibaba Cloud ACR OpenAPI.
// API endpoint: https://cr.{region}.aliyuncs.com
func (a *ACRAdapter) callOpenAPI(method string, params map[string]string) ([]byte, int, error) {
	apiURL := fmt.Sprintf("https://cr.%s.aliyuncs.com", a.region)

	// Add common parameters
	params["AccessKeyId"] = a.accessKey
	params["Format"] = "JSON"
	params["Version"] = "2016-06-07"
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())

	// Sort and encode parameters for signature
	sortedKeys := make([]string, 0, len(params))
	for k := range params {
		sortedKeys = append(sortedKeys, k)
	}
	sortStrings(sortedKeys)

	queryParts := make([]string, 0, len(params))
	for _, k := range sortedKeys {
		queryParts = append(queryParts, specialEncode(k)+"="+specialEncode(params[k]))
	}
	canonicalQuery := strings.Join(queryParts, "&")

	// Build string to sign
	stringToSign := method + "&" + specialEncode("/") + "&" + specialEncode(canonicalQuery)

	// Compute signature: HMAC-SHA1 with AccessKey Secret + "&"
	signature := computeHMACSHA1(stringToSign, a.secretKey+"&")

	// Add signature to query
	finalQuery := canonicalQuery + "&Signature=" + specialEncode(signature)

	reqURL := apiURL + "?" + finalQuery
	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// specialEncode implements Alibaba Cloud's URL encoding spec
// (different from standard URL encoding — uses uppercase hex, encodes more chars)
func specialEncode(s string) string {
	encoded := url.QueryEscape(s)
	// Alibaba Cloud spec requires uppercase hex
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func computeHMACSHA1(data, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// ListRepositories lists repositories in an ACR namespace (Phase 2 implementation).
func (a *ACRAdapter) ListRepositories(namespace string) ([]string, error) {
	if a.accessKey == "" || a.secretKey == "" {
		return nil, fmt.Errorf("ACR AccessKey/SecretKey required for repository listing")
	}
	params := map[string]string{
		"Action":    "ListRepository",
		"Namespace": namespace,
		"RegionId":  a.region,
	}
	body, statusCode, err := a.callOpenAPI("GET", params)
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("ACR ListRepository returned status %d: %s", statusCode, string(body))
	}

	var result struct {
		Data struct {
			Repositories []struct {
				Name string `json:"RepoName"`
			} `json:"Repositories"`
		} `json:"Data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(result.Data.Repositories))
	for _, r := range result.Data.Repositories {
		names = append(names, r.Name)
	}
	return names, nil
}