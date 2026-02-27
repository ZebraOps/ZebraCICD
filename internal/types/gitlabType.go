package types

type Project struct {
	Path          string `json:"path_with_namespace"`
	Name          string `json:"name"`
	SSHURLToRepo  string `json:"ssh_url_to_repo"`
	HTTPURLToRepo string `json:"http_url_to_repo"`
	Desc          string `json:"description"`
}

type Branch struct {
	Name string `json:"name"`
}

type Tag struct {
	Name string `json:"name"`
}
