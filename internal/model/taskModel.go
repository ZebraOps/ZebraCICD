package model

import "time"

type DeployTask struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	ProjectID  uint       `gorm:"comment:项目ID" json:"project_id"`
	EnvID      uint       `gorm:"comment:环境ID" json:"env_id"`
	GitRef     string     `gorm:"size:255;comment:Git引用（分支/标签）" json:"git_ref"`
	ImageTag   string     `gorm:"size:255;comment:镜像标签" json:"image_tag"`
	Status     string     `gorm:"size:50;index;comment:部署状态" json:"status"` // PENDING, BUILDING, PUSHING, DEPLOYING, SUCCESS, FAILED
	LogPath    string     `gorm:"type:text;comment:日志路径" json:"log_path"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 新增字段
	K8sClusterID    uint   `gorm:"comment:K8s集群ID" json:"k8s_cluster_id"`
	K8sNamespace    string `gorm:"size:100;comment:K8s命名空间" json:"k8s_namespace"`
	JenkinsJobName  string `gorm:"size:255;comment:Jenkins任务名称" json:"jenkins_job_name"`
	RegistryProject string `gorm:"size:255;comment:仓库项目" json:"registry_project"`
	ImageName       string `gorm:"size:255;comment:镜像名称" json:"image_name"`
	DeploymentName  string `gorm:"size:255;comment:部署名称" json:"deployment_name"`

	// 模板关联
	BuildTemplateID      *uint `gorm:"comment:构建模板ID" json:"build_template_id,omitempty"`
	DeploymentTemplateID *uint `gorm:"comment:部署模板ID" json:"deployment_template_id,omitempty"`

	// 部署类型与目标
	DeployType        string `gorm:"size:50;default:'k8s';comment:部署类型(k8s/docker)" json:"deploy_type"`
	DeployTarget      string `gorm:"size:50;default:'k8s';comment:部署目标(k8s/docker/linux)" json:"deploy_target"`
	ServerID          uint   `gorm:"comment:目标Linux服务器ID(Docker/Linux部署)" json:"server_id"`
	DockerComposePath string `gorm:"size:500;comment:docker-compose文件路径" json:"docker_compose_path,omitempty"`
	DeployPath        string `gorm:"size:500;comment:部署路径(linux/Nginx代理目录)" json:"deploy_path,omitempty"`

	// Jenkins 构建信息
	JenkinsBuildNumber int    `gorm:"comment:Jenkins构建编号" json:"jenkins_build_number,omitempty"`
	ErrorMessage       string `gorm:"type:text;comment:错误信息" json:"error_message,omitempty"`
	RetryCount         int    `gorm:"default:0;comment:重试次数" json:"retry_count"`

	// 回滚相关字段
	IsRollback   bool `gorm:"default:false;comment:是否为回滚任务" json:"is_rollback"`
	RollbackFrom uint `gorm:"comment:回滚源任务ID" json:"rollback_from,omitempty"`
}
