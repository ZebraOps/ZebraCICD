package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

type Language struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:100;uniqueIndex;not null;comment:语言名称" json:"name"`
	DisplayName string            `gorm:"size:100;comment:显示名称" json:"display_name"`
	Icon        string            `gorm:"size:255;comment:图标" json:"icon"`
	Status      string            `gorm:"size:50;default:'active';comment:状态(active/inactive)" json:"status"`
	SortOrder   int               `gorm:"default:0;comment:排序" json:"sort_order"`
	Description string            `gorm:"type:text;comment:描述" json:"description"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (Language) TableName() string {
	return "languages"
}
