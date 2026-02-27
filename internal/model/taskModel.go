package model

import "time"

type DeployTask struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProjectID  uint      `gorm:"comment:项目ID" json:"project_id"`
	EnvID      uint      `gorm:"comment:环境ID" json:"env_id"`
	GitRef     string    `gorm:"size:255;comment:Git引用（分支/标签）" json:"git_ref"`
	ImageTag   string    `gorm:"size:255;comment:镜像标签" json:"image_tag"`
	Status     string    `gorm:"size:50;index;comment:部署状态" json:"status"` // PENDING, BUILDING, PUSHING, DEPLOYING, SUCCESS, FAILED
	LogPath    string    `gorm:"type:text;comment:日志路径" json:"log_path"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 新增字段
	K8sClusterID   uint   `gorm:"comment:K8s集群ID" json:"k8s_cluster_id"`
	K8sNamespace   string `gorm:"size:100;comment:K8s命名空间" json:"k8s_namespace"`
	JenkinsJobName string `gorm:"size:255;comment:Jenkins任务名称" json:"jenkins_job_name"`
	HarborProject  string `gorm:"size:255;comment:Harbor项目" json:"harbor_project"`
	ImageName      string `gorm:"size:255;comment:镜像名称" json:"image_name"`
	DeploymentName string `gorm:"size:255;comment:部署名称" json:"deployment_name"`
}
