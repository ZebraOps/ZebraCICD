package model

import "github.com/ZebraOps/ZebraCICD/pkg/timeutil"

// JenkinsCredential 存储从 Jenkins 同步过来的凭据元数据（不存密文）
type JenkinsCredential struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	JenkinsPlatformID uint   `gorm:"not null;index;comment:所属Jenkins平台ID" json:"jenkins_platform_id"`
	CredentialID      string `gorm:"size:255;not null;comment:Jenkins中的凭据ID" json:"credential_id"`
	DisplayName       string `gorm:"size:255;comment:显示名称" json:"display_name"`
	Description       string `gorm:"size:500;comment:描述" json:"description"`
	CredentialType    string `gorm:"size:100;comment:凭据类型(UsernamePassword/SecretText/SSHKey/Certificate等)" json:"credential_type"`
	Username          string `gorm:"size:255;comment:用户名(仅UsernamePassword类型)" json:"username"`
	Scope             string `gorm:"size:50;default:'GLOBAL';comment:作用域(GLOBAL/SYSTEM)" json:"scope"`
	// active: 正常; synced_deleted: Jenkins上已不存在，本地保留引用
	Status    string            `gorm:"size:50;default:'active';comment:状态(active/synced_deleted)" json:"status"`
	SyncedAt  timeutil.JSONTime `gorm:"comment:最后同步时间" json:"synced_at"`
	CreatedAt timeutil.JSONTime `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt timeutil.JSONTime `gorm:"comment:更新时间" json:"updated_at"`
}

func (JenkinsCredential) TableName() string {
	return "jenkins_credentials"
}
