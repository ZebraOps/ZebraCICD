package model

import "time"

// StageHistory records each pipeline stage's execution details for a deploy task.
// Stages: PENDING, BUILDING, PUSHING, DEPLOYING
// Status per stage: running, success, failed
type StageHistory struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	TaskID       uint       `gorm:"index;not null;comment:关联部署任务ID" json:"task_id"`
	Stage        string     `gorm:"type:varchar(20);not null;comment:阶段名称" json:"stage"`
	Status       string     `gorm:"type:varchar(20);not null;comment:阶段状态" json:"status"`
	RetryCount   int        `gorm:"default:0;comment:重试轮次(0=首次,1=第一次重试)" json:"retry_count"`
	StartedAt    *time.Time `gorm:"comment:阶段开始时间" json:"started_at,omitempty"`
	FinishedAt   *time.Time `gorm:"comment:阶段结束时间" json:"finished_at,omitempty"`
	ErrorMessage string     `gorm:"type:text;comment:错误信息" json:"error_message,omitempty"`
	LogSummary   string     `gorm:"type:text;comment:日志摘要" json:"log_summary,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (StageHistory) TableName() string {
	return "stage_history"
}