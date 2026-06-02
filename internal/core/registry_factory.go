package core

import "github.com/ZebraOps/ZebraCICD/internal/model"

// NewRegistryClient creates a RegistryClient based on the repository type.
// Supported types: "v2" (standard Docker Registry V2), "harbor" (Harbor REST API), "acr" (Alibaba Cloud ACR).
func NewRegistryClient(repoType, baseURL, username, password string) RegistryClient {
	switch repoType {
	case "acr":
		return NewACRAdapter(baseURL, username, password)
	case "harbor":
		return NewHarborAdapter(baseURL, username, password)
	default: // "v2" — standard Docker Registry V2 (push auto-creates repo)
		return NewV2RegistryAdapter(baseURL, username, password)
	}
}

// NewRegistryClientFromRepo creates a RegistryClient from an ImageRepository model.
// This is the preferred method as it passes all credentials including ACR-specific AccessKey/SecretKey.
func NewRegistryClientFromRepo(repo *model.ImageRepository) RegistryClient {
	switch repo.Type {
	case "acr":
		return NewACRAdapterWithKeys(repo.URL, repo.Username, repo.Password, repo.AccessKey, repo.SecretKey)
	case "harbor":
		return NewHarborAdapter(repo.URL, repo.Username, repo.Password)
	default: // "v2"
		return NewV2RegistryAdapter(repo.URL, repo.Username, repo.Password)
	}
}