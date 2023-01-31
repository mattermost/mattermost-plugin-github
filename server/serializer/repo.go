package serializer

// Only send down fields to the client that are needed
type RepositoryResponse struct {
	Name        string          `json:"name,omitempty"`
	FullName    string          `json:"full_name,omitempty"`
	Permissions map[string]bool `json:"permissions,omitempty"`
}
