package model

import (
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
)

type RepoResp struct {
	ID             uint              `gorm:"column:id" json:"id"`
	RepoNumber     string            `gorm:"column:repo_number" json:"repo_number"`
	CName          string            `gorm:"column:c_name" json:"c_name"`
	EName          string            `gorm:"column:e_name" json:"e_name"`
	RepoURL        string            `gorm:"column:repo_url" json:"repo_url"`
	RepoSSHURL     string            `gorm:"column:repo_ssh_url" json:"repo_ssh_url"`
	RepoManager    string            `gorm:"column:repo_manager" json:"repo_manager"`
	RepoDepartment string            `gorm:"column:repo_department" json:"repo_department"`
	RepoLanguage   string            `gorm:"column:repo_language" json:"repo_language"`
	RepoDesc       string            `gorm:"column:repo_desc" json:"repo_desc"`
	RepoDeployType string            `gorm:"column:repo_deploy_type" json:"repo_deploy_type"`
	RepoBuildPath  string            `gorm:"column:repo_build_path" json:"repo_build_path"`
	CreatedAt      timeutil.JSONTime `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      timeutil.JSONTime `gorm:"column:updated_at" json:"updated_at"`
}
type Repo struct {
	ID             uint              `gorm:"primaryKey" json:"id"`
	RepoNumber     string            `gorm:"type:text;comment:仓库编号" json:"repo_number"`
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

	Applications []*Application `gorm:"foreignKey:RepoID" json:"applications,omitempty"`
}
