package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

type CloudProvider struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;uniqueIndex;not null;comment:云厂商名称" json:"name"`
	DisplayName string            `gorm:"size:255;comment:显示名称" json:"display_name"`
	Description string            `gorm:"type:text;comment:云厂商描述" json:"description"`
	Provider    string            `gorm:"size:50;uniqueIndex;not null;comment:提供商标识(aliyun/aws/azure/gcp)" json:"provider"`
	Region      string            `gorm:"size:100;comment:默认区域" json:"region"`
	AccessKey   string            `gorm:"type:text;comment:访问密钥" json:"access_key"`
	SecretKey   string            `gorm:"type:text;comment:密钥" json:"secret_key"`
	Endpoint    string            `gorm:"type:text;comment:API端点" json:"endpoint"`
	Config      string            `gorm:"type:text;comment:配置信息(JSON格式)" json:"config"`
	Status      string            `gorm:"size:50;default:'active';comment:状态(active/inactive)" json:"status"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (CloudProvider) TableName() string {
	return "cloud_providers"
}
