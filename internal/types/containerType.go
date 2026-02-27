package types

import "time"

type ContainerExecRequest struct {
	Command string `json:"command"`
}

type ContainerExecResponse struct {
	Output string `json:"output"`
	Error  error  `json:"error,omitempty"`
}

type PodInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	NodeName  string            `json:"node_name"`
	Namespace string            `json:"namespace"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	Labels    map[string]string `json:"labels"`
}
