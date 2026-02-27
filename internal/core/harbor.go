package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HarborClient struct {
	baseURL string
	client  *http.Client
}

func NewHarborClient(baseURL string) *HarborClient {
	return &HarborClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type HarborTag struct {
	Name string `json:"name"`
}

func (h *HarborClient) GetImageTags(project, repository string) ([]HarborTag, error) {
	// Example: /api/v2.0/projects/{project}/repositories/{repository}/artifacts
	u := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts", h.baseURL, project, repository)
	fmt.Println(u)
	req, _ := http.NewRequest("GET", u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []struct {
		Tags []HarborTag `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	// flatten tags
	var out []HarborTag
	for _, d := range data {
		out = append(out, d.Tags...)
	}
	return out, nil
}
