package types

import "time"

type ContainerExecRequest struct {
	Command string `json:"command"`
}

type ContainerExecResponse struct {
	Output string `json:"output"`
	Error  error  `json:"error,omitempty"`
}

// ContainerInfo represents a single container within a K8s Pod.
type ContainerInfo struct {
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restart_count"`
	Image        string `json:"image"`
	State        string `json:"state"` // running / terminated / waiting
}

type PodInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	NodeName     string            `json:"node_name"`
	Namespace    string            `json:"namespace"`
	PodIP        string            `json:"pod_ip,omitempty"`
	StartTime    *time.Time        `json:"start_time,omitempty"`
	Labels       map[string]string `json:"labels"`
	RestartCount int               `json:"restart_count"`
	Ready        string            `json:"ready"`
	Containers   []ContainerInfo   `json:"containers"`
}

// PodMetric holds CPU/memory usage for a pod from the K8s Metrics API.
type PodMetric struct {
	CPU    string `json:"cpu"`    // e.g. "100m"
	Memory string `json:"memory"` // e.g. "128Mi"
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
