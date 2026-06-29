package model

import "time"

// ContainerOperation records container management actions (restart, delete, terminal, etc.)
type ContainerOperation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OperationType string    `gorm:"type:varchar(30);not null;comment:操作类型" json:"operation_type"`
	TargetType    string    `gorm:"type:varchar(10);not null;comment:目标类型(k8s/docker)" json:"target_type"`
	TargetDetail  string    `gorm:"type:varchar(500);not null;comment:操作目标详情" json:"target_detail"`
	Operator      string    `gorm:"type:varchar(100);not null;comment:操作人" json:"operator"`
	Result        string    `gorm:"type:varchar(20);not null;comment:操作结果" json:"result"`
	Details       string    `gorm:"type:text;comment:附加详情" json:"details,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ContainerOperation) TableName() string {
	return "container_operations"
}
