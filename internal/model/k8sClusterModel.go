package model

import (
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
)

type K8SCluster struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;uniqueIndex;not null;comment:集群名称" json:"name"`
	Description string            `gorm:"type:text;comment:集群描述" json:"description"`
	ApiServer   string            `gorm:"type:text;not null;comment:API服务器地址" json:"api_server"`
	CaCert      string            `gorm:"type:text;comment:CA证书" json:"ca_cert"`
	ClientCert  string            `gorm:"type:text;comment:客户端证书" json:"client_cert"`
	ClientKey   string            `gorm:"type:text;comment:客户端私钥" json:"client_key"`
	Token       string            `gorm:"type:text;comment:认证Token" json:"token"`
	SkipVerify  bool              `gorm:"default:false;comment:是否跳过证书验证" json:"skip_verify"`
	Namespace   string            `gorm:"size:100;default:'default';comment:默认命名空间" json:"namespace"`
	IsActive    bool              `gorm:"default:true;comment:是否激活" json:"is_active"`
	Vendor      string            `gorm:"size:100;comment:云厂商" json:"vendor"`
	Environment string            `gorm:"size:50;comment:所属环境" json:"environment"`
	Enabled     bool              `gorm:"default:true;comment:是否启用" json:"enabled"`
	CreatedAt   timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (K8SCluster) TableName() string {
	return "k8s_clusters"
}
