package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

// ImageRepository 表示通用的镜像仓库信息
type ImageRepository struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;uniqueIndex;not null;comment:仓库名称" json:"name"`
	URL         string            `gorm:"size:255;not null;comment:仓库地址" json:"url"`
	Username    string            `gorm:"size:100;comment:用户名" json:"username"`
	Password    string            `gorm:"type:text;comment:密码" json:"password"`
	Description string            `gorm:"type:text;comment:描述" json:"description"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (ImageRepository) TableName() string {
	return "image_repositories"
}
