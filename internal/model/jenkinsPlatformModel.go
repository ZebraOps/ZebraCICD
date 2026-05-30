package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

type JenkinsPlatform struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:100;uniqueIndex;not null;comment:平台名称" json:"name"`
	DisplayName string            `gorm:"size:100;comment:显示名称" json:"display_name"`
	URL         string            `gorm:"size:500;not null;comment:平台地址" json:"url"`
	Username    string            `gorm:"size:100;comment:用户名" json:"username"`
	Password    string            `gorm:"size:255;comment:密码/Token" json:"password"`
	Description string            `gorm:"size:500;comment:描述" json:"description"`
	Status      string            `gorm:"size:50;default:'active';comment:状态(active/inactive)" json:"status"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (JenkinsPlatform) TableName() string {
	return "jenkins_platforms"
}