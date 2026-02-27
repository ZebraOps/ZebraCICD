package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

// DeploymentTemplateHistory 部署模板历史记录
type DeploymentTemplateHistory struct {
	ID                   uint              `gorm:"primaryKey" json:"id"`
	DeploymentTemplateID uint              `gorm:"index;comment:部署模板ID" json:"deployment_template_id"`
	Modifier             string            `gorm:"size:100;comment:修改人" json:"modifier"`
	Name                 string            `gorm:"size:255;comment:模板名称" json:"name"`
	DisplayName          string            `gorm:"size:255;comment:显示名称" json:"display_name"`
	Description          string            `gorm:"type:text;comment:模板描述" json:"description"`
	TemplateType         string            `gorm:"size:50;comment:模板类型" json:"template_type"`
	Content              string            `gorm:"type:text;comment:模板内容" json:"content"`
	Variables            string            `gorm:"type:text;comment:模板变量" json:"variables"`
	Parameters           string            `gorm:"type:text;comment:模板参数" json:"parameters"`
	Version              string            `gorm:"size:50;comment:模板版本" json:"version"`
	ChangeReason         string            `gorm:"type:text;comment:修改原因" json:"change_reason"`
	CreatedAt            timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
}

// DeploymentTemplate 部署模板模型
type DeploymentTemplate struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	Name         string            `gorm:"size:255;uniqueIndex;not null;comment:部署模板名称" json:"name"`
	DisplayName  string            `gorm:"size:255;comment:显示名称" json:"display_name"`
	Description  string            `gorm:"type:text;comment:部署模板描述" json:"description"`
	TemplateType string            `gorm:"size:50;comment:模板类型(k8s/helm/docker)" json:"template_type"`
	Content      string            `gorm:"type:text;comment:模板内容(YAML/JSON格式)" json:"content"`
	Variables    string            `gorm:"type:text;comment:模板变量(JSON格式)" json:"variables"`
	Parameters   string            `gorm:"type:text;comment:模板参数(JSON格式)" json:"parameters"`
	Version      string            `gorm:"size:50;default:'1.0';comment:模板版本" json:"version"`
	Status       string            `gorm:"size:50;default:'active';comment:状态(active/inactive)" json:"status"`
	Creator      string            `gorm:"size:100;comment:创建人" json:"creator"`
	Updater      string            `gorm:"size:100;comment:更新人" json:"updater"`
	CreatedAt    timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`

	// 关联仓库表的多对多关系
	Repos []Repo `gorm:"many2many:repo_deployment_templates;" json:"repos,omitempty"`

	// 正确的关联历史记录关系
	Histories []DeploymentTemplateHistory `gorm:"foreignKey:DeploymentTemplateID" json:"histories,omitempty"`
}
