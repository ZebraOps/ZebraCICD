package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

type GitPlatform struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:100;uniqueIndex;not null;comment:平台名称" json:"name"`
	DisplayName string            `gorm:"size:100;comment:显示名称" json:"display_name"`
	PlatformType string           `gorm:"size:50;not null;default:'gitlab';comment:平台类型(gitlab/github/gitea/custom)" json:"platform_type"`
	URL         string            `gorm:"size:500;not null;comment:平台地址" json:"url"`
	APIUrl      string            `gorm:"size:500;comment:API地址" json:"api_url"`
	AuthType    string            `gorm:"size:50;default:'token';comment:认证方式(token/oauth/password)" json:"auth_type"`
	AuthConfig  string            `gorm:"type:text;comment:认证配置JSON" json:"auth_config"`
	Description string            `gorm:"size:500;comment:描述" json:"description"`
	Status      string            `gorm:"size:50;default:'active';comment:状态(active/inactive)" json:"status"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (GitPlatform) TableName() string {
	return "git_platforms"
}