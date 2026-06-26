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
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	NodeName     string            `json:"node_name"`
	Namespace    string            `json:"namespace"`
	StartTime    *time.Time        `json:"start_time,omitempty"`
	Labels       map[string]string `json:"labels"`
	RestartCount int               `json:"restart_count"`
	Ready        string            `json:"ready"`
	Containers   []string          `json:"containers"`
}

// PodLogResponse K8s Pod 日志响应
type PodLogResponse struct {
	Output    string `json:"output"`
	PodName   string `json:"pod_name"`
	Namespace string `json:"namespace"`
	Container string `json:"container,omitempty"`
}

// ContainerLogResponse Docker 容器日志响应
type ContainerLogResponse struct {
	Output      string `json:"output"`
	ContainerID string `json:"container_id"`
}
