package model

import (
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
)

type RepoResp struct {
	ID             uint              `json:"id"`
	RepoID         string            `json:"repo_id"`
	CName          string            `json:"c_name"`
	EName          string            `json:"e_name"`
	RepoURL        string            `json:"repo_url"`
	RepoSSHURL     string            `json:"repo_ssh_url"`
	RepoManager    string            `json:"repo_manager"`
	RepoDepartment string            `json:"repo_department"`
	RepoLanguage   string            `json:"repo_language"`
	RepoDesc       string            `json:"repo_desc"`
	RepoDeployType string            `json:"repo_deploy_type"`
	RepoBuildPath  string            `json:"repo_build_path"`
	CreatedAt      timeutil.JSONTime `json:"created_at"`
	UpdatedAt      timeutil.JSONTime `json:"updated_at"`
}
type Repo struct {
	ID             uint              `gorm:"primaryKey" json:"id"`
	RepoID         string            `gorm:"type:text;not null;comment:仓库ID" json:"repo_id"`
	CName          string            `gorm:"size:255;uniqueIndex;not null;comment:中文名称" json:"c_name"`
	EName          string            `gorm:"size:255;uniqueIndex;not null;comment:英文名称" json:"e_name"`
	RepoURL        string            `gorm:"type:text;comment:HTTP地址" json:"repo_url"`
	RepoSSHURL     string            `gorm:"type:text;comment:SSH地址" json:"repo_ssh_url"`
	RepoManager    string            `gorm:"type:text;comment:责任人" json:"repo_manager"`
	RepoDepartment string            `gorm:"type:text;comment:归属部门" json:"repo_department"`
	RepoLanguage   string            `gorm:"type:text;comment:开发语言" json:"repo_language"`
	RepoDesc       string            `gorm:"type:text;comment:描述" json:"repo_desc"`
	RepoDeployType string            `gorm:"type:text;comment:部署类型" json:"repo_deploy_type"`
	RepoBuildPath  string            `gorm:"type:text;comment:构建路径" json:"repo_build_path"`
	CreatedAt      timeutil.JSONTime `gorm:"type:timestamp;comment:创建时间" json:"created_at"`
	UpdatedAt      timeutil.JSONTime `gorm:"type:timestamp;comment:更新时间" json:"updated_at"`

	// 添加与构建模板的多对多关联关系
	Templates []*BuildTemplate `gorm:"many2many:repo_templates;" json:"templates,omitempty"`

	// 添加与部署模板的多对多关联关系
	DeploymentTemplates []*DeploymentTemplate `gorm:"many2many:repo_deployment_templates;" json:"deployment_templates,omitempty"`
}
