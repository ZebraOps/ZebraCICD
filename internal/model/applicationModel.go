package model

import (
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
)

type ApplicationRequest struct {
	RepoID          uint   `gorm:"not null;comment:关联仓库ID" json:"repo_id"`
	CName           string `gorm:"size:255;not null;comment:中文名称" json:"c_name"`
	EName           string `gorm:"size:255;not null;comment:英文名称" json:"e_name"`
	ListenPort      int    `gorm:"comment:监听端口" json:"listen_port"`
	HealthCheckType string `gorm:"size:50;comment:健康检查类型(http/tcp/custom)" json:"health_check_type"`
	HealthCheckURL  string `gorm:"size:255;comment:健康检查URL" json:"health_check_url"`
	Description     string `gorm:"type:text;comment:描述" json:"description"`
}

type ApplicationResponse struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	RepoID          uint              `gorm:"not null;comment:关联仓库ID" json:"repo_id"`
	CName           string            `gorm:"size:255;not null;comment:中文名称" json:"c_name"`
	EName           string            `gorm:"size:255;not null;comment:英文名称" json:"e_name"`
	ListenPort      int               `gorm:"comment:监听端口" json:"listen_port"`
	HealthCheckType string            `gorm:"size:50;comment:健康检查类型(http/tcp/custom)" json:"health_check_type"`
	HealthCheckURL  string            `gorm:"size:255;comment:健康检查URL" json:"health_check_url"`
	Description     string            `gorm:"type:text;comment:描述" json:"description"`
	Department      string            `json:"department"`   // 归属部门（来自关联仓库）
	Language        string            `json:"language"`     // 开发语言（来自关联仓库）
	CreatedAt       timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt       timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
	DeploymentCount int64             `json:"deployment_count"` // 部署配置数量
}

// Application 应用服务表
type Application struct {
	ID uint `gorm:"primaryKey" json:"id"`
	ApplicationRequest
	CreatedAt timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`

	// 修正外键关系定义
	Repo        *Repo                   `gorm:"foreignKey:RepoID;references:ID" json:"repo,omitempty"`
	Deployments []ApplicationDeployment `gorm:"foreignKey:ApplicationID" json:"deployments,omitempty"`

	// 关联构建模板的多对多关系
	BuildTemplates      []BuildTemplate      `gorm:"many2many:build_template_applications;" json:"build_templates,omitempty"`
	// 关联部署模板的多对多关系
	DeploymentTemplates []DeploymentTemplate `gorm:"many2many:deployment_template_applications;" json:"deployment_templates,omitempty"`
}

type ApplicationDeploymentRequest struct {
	ApplicationID        uint   `gorm:"not null;comment:应用服务ID" json:"application_id"`
	EnvironmentID        uint   `gorm:"not null;comment:环境ID" json:"environment_id"`
	DeployTarget         string `gorm:"size:50;not null;default:'k8s';comment:部署目标(k8s/docker/linux)" json:"deploy_target"`
	BuildSource          string `gorm:"size:50;default:'tag';comment:构建源(tag/branch)" json:"build_source"`
	Description          string `gorm:"type:text;comment:描述" json:"description"`
	BuildTemplateID      *uint  `gorm:"comment:构建模板ID" json:"build_template_id"`
	DeploymentTemplateID *uint  `gorm:"comment:部署模板ID" json:"deployment_template_id"`
	K8sClusterID         *uint  `gorm:"comment:K8s集群ID" json:"k8s_cluster_id"`
	K8sNamespace         string `gorm:"size:100;default:'default';comment:K8s命名空间" json:"k8s_namespace"`
	ServerID             *uint  `gorm:"comment:目标服务器ID(docker/linux)" json:"server_id"`
	DeployPath           string `gorm:"size:500;comment:部署路径(linux/Nginx代理目录)" json:"deploy_path"`
	// 新增：平台关联字段（nullable，兼容旧数据）
	JenkinsPlatformID    *uint  `gorm:"index;comment:Jenkins平台ID" json:"jenkins_platform_id"`
	GitPlatformID        *uint  `gorm:"index;comment:Git平台ID" json:"git_platform_id"`
	ImageRepoID          *uint  `gorm:"index;comment:镜像仓库ID" json:"image_repo_id"`
}

type ApplicationDeploymentResponse struct {
	ID uint `gorm:"primaryKey" json:"id"`
	ApplicationDeploymentRequest
	CreatedAt timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
	//Application        *Application        `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	//Environment        *Environment        `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
	//K8sCluster         *K8SCluster         `gorm:"foreignKey:K8sClusterID" json:"k8s_cluster,omitempty"`
	//BuildTemplate      *BuildTemplate      `gorm:"foreignKey:BuildTemplateID" json:"build_template,omitempty"`
	//DeploymentTemplate *DeploymentTemplate `gorm:"foreignKey:DeploymentTemplateID" json:"deployment_template,omitempty"`
}

// ApplicationDeployment 应用部署配置表
type ApplicationDeployment struct {
	ID uint `gorm:"primaryKey" json:"id"`
	ApplicationDeploymentRequest
	CreatedAt timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`

	// 关联关系
	Application        *Application        `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	Environment        *Environment        `gorm:"foreignKey:EnvironmentID" json:"environment,omitempty"`
	BuildTemplate      *BuildTemplate      `gorm:"foreignKey:BuildTemplateID" json:"build_template,omitempty"`
	DeploymentTemplate *DeploymentTemplate `gorm:"foreignKey:DeploymentTemplateID" json:"deployment_template,omitempty"`
	K8sCluster         *K8SCluster         `gorm:"foreignKey:K8sClusterID" json:"k8s_cluster,omitempty"`
	Server             *Server             `gorm:"foreignKey:ServerID" json:"server,omitempty"`
	// 新增：平台关联
	JenkinsPlatform    *JenkinsPlatform    `gorm:"foreignKey:JenkinsPlatformID" json:"jenkins_platform,omitempty"`
	GitPlatform        *GitPlatform        `gorm:"foreignKey:GitPlatformID" json:"git_platform,omitempty"`
	ImageRepository    *ImageRepository    `gorm:"foreignKey:ImageRepoID" json:"image_repository,omitempty"`
}
