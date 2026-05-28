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
}

type ApplicationDeploymentRequest struct {
	ApplicationID        uint   `gorm:"not null;comment:应用服务ID" json:"application_id"`
	EnvironmentID        uint   `gorm:"not null;comment:环境ID" json:"environment_id"`
	PlatformCredentialID *uint  `gorm:"comment:平台凭据ID" json:"platform_credential_id"`
	BuildSource          string `gorm:"size:50;default:'tag';comment:构建源(tag/branch)" json:"build_source"`
	Description          string `gorm:"type:text;comment:描述" json:"description"`
	BuildTemplateID      *uint  `gorm:"comment:构建模板ID" json:"build_template_id"`
	DeploymentTemplateID *uint  `gorm:"comment:部署模板ID" json:"deployment_template_id"`
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
}
