package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

type Environment struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;uniqueIndex;not null;comment:环境名称" json:"name"`
	Description string            `gorm:"type:text;comment:环境描述" json:"description"`
	Type        string            `gorm:"size:50;comment:环境类型(dev/test/prod)" json:"type"`
	Status      string            `gorm:"size:50;default:'active';comment:环境状态(active/inactive)" json:"status"`
	Config      string            `gorm:"type:text;comment:环境配置(JSON格式)" json:"config"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (Environment) TableName() string {
	return "environments"
}
