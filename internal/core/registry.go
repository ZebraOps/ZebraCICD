package core

// RegistryClient defines the interface for interacting with a Docker/OCI image registry.
// Implementations include V2RegistryAdapter (standard Docker Registry V2 API),
// ACRAdapter (Alibaba Cloud ACR), and HarborAdapter (Harbor REST API).
type RegistryClient interface {
	// VerifyImageExists checks if a specific image tag exists in the registry.
	VerifyImageExists(project, imageName, tag string) bool

	// ListTags returns the list of tags for a given repository in the registry.
	ListTags(project, imageName string) ([]string, error)

	// EnsureProjectExists checks if the project/namespace exists in the registry,
	// and creates it if it doesn't. For standard V2 registries, this is a no-op
	// since projects are created implicitly on first push.
	// For Harbor, this calls the Harbor REST API to create a project.
	// For ACR, this calls the ACR OpenAPI to create a namespace.
	EnsureProjectExists(project string) error

	// Ping tests the connection to the registry.
	Ping() error
}