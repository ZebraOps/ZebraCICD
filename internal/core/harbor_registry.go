package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HarborAdapter implements the RegistryClient interface for Harbor registries.
// It uses the Harbor REST API for project management and the V2 API for image operations.
//
// Harbor requires projects to be explicitly created before images can be pushed.
// This adapter handles project creation via the Harbor REST API (/api/v2.0/projects).
type HarborAdapter struct {
	baseURL  string // Harbor URL (e.g. "https://harbor.example.com")
	username string
	password string
	client   *http.Client
	v2       *V2RegistryAdapter // V2 API operations delegated here
}

func NewHarborAdapter(baseURL string, username string, password string) *HarborAdapter {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &HarborAdapter{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 15 * time.Second},
		v2:       NewV2RegistryAdapter(baseURL, username, password),
	}
}

// VerifyImageExists delegates to V2 API.
func (h *HarborAdapter) VerifyImageExists(project, imageName, tag string) bool {
	return h.v2.VerifyImageExists(project, imageName, tag)
}

// ListTags delegates to V2 API.
func (h *HarborAdapter) ListTags(project, imageName string) ([]string, error) {
	return h.v2.ListTags(project, imageName)
}

// EnsureProjectExists checks if a project exists in Harbor and creates it if needed.
// Harbor requires projects to exist before images can be pushed to them.
func (h *HarborAdapter) EnsureProjectExists(project string) error {
	// 1. Check if project already exists
	exists, err := h.projectExists(project)
	if err != nil {
		fmt.Printf("Harbor: error checking project %s: %v, attempting creation anyway\n", project, err)
		// Don't block deployment — try to create it, it might already exist
	}
	if exists {
		fmt.Printf("Harbor: project %s already exists\n", project)
		return nil
	}

	// 2. Create the project
	return h.createProject(project)
}

// projectExists checks if a project exists in Harbor.
func (h *HarborAdapter) projectExists(project string) (bool, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects?name=%s&page_size=1", h.baseURL, project)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(h.username, h.password)

	resp, err := h.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Harbor returns an array of matching projects
		body, _ := io.ReadAll(resp.Body)
		var projects []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &projects); err == nil {
			for _, p := range projects {
				if p.Name == project {
					return true, nil
				}
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("harbor project check returned status %d", resp.StatusCode)
}

// createProject creates a project in Harbor via REST API.
func (h *HarborAdapter) createProject(project string) error {
	url := fmt.Sprintf("%s/api/v2.0/projects", h.baseURL)
	payload := map[string]interface{}{
		"project_name": project,
		"metadata": map[string]string{
			"public": "false", // Private project by default
		},
		"storage_limit": -1, // No storage limit
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.SetBasicAuth(h.username, h.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("harbor create project request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 || resp.StatusCode == 200 {
		fmt.Printf("Harbor: project %s created successfully\n", project)
		return nil
	}

	// Project might already exist (concurrent creation)
	if resp.StatusCode == 409 {
		fmt.Printf("Harbor: project %s already exists (concurrent creation)\n", project)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("harbor create project returned status %d: %s", resp.StatusCode, string(respBody))
}