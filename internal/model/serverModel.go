package model

import (
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
)

type Server struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;uniqueIndex;not null;comment:服务器名称" json:"name"`
	Description string            `gorm:"type:text;comment:服务器描述" json:"description"`
	Host        string            `gorm:"size:255;not null;comment:服务器地址" json:"host"`
	Port        int               `gorm:"default:22;comment:SSH端口" json:"port"`
	Username    string            `gorm:"size:100;not null;comment:用户名" json:"username"`
	AuthType    string            `gorm:"size:20;default:'password';comment:认证类型(password/key)" json:"auth_type"`
	Password    string            `gorm:"type:text;comment:密码" json:"password"`
	PrivateKey  string            `gorm:"type:text;comment:私钥" json:"private_key"`
	IsActive    bool              `gorm:"default:true;comment:是否激活" json:"is_active"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

type DockerContainer struct {
	ID        string            `json:"id"`
	Names     []string          `json:"names"`
	Image     string            `json:"image"`
	Command   string            `json:"command"`
	Status    string            `json:"status"`
	Ports     string            `json:"ports"`
	Labels    string            `json:"labels"`
	CreatedAt timeutil.JSONTime `json:"created_at"`
}
