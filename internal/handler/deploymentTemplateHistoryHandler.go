package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"gorm.io/gorm"
)

type DeploymentTemplateHistoryRepository struct {
	db *gorm.DB
}

func NewDeploymentTemplateHistoryRepository(db *gorm.DB) *DeploymentTemplateHistoryRepository {
	return &DeploymentTemplateHistoryRepository{db: db}
}

// Create 创建部署模板历史记录
func (r *DeploymentTemplateHistoryRepository) Create(history *model.DeploymentTemplateHistory) error {
	return r.db.Create(history).Error
}

// GetHistoryByTemplateID 根据部署模板ID获取历史记录
func (r *DeploymentTemplateHistoryRepository) GetHistoryByTemplateID(templateID uint) ([]model.DeploymentTemplateHistory, error) {
	var histories []model.DeploymentTemplateHistory
	if err := r.db.Where("deployment_template_id = ?", templateID).Order("created_at DESC").Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// GetLatestHistory 获取最新的历史记录
func (r *DeploymentTemplateHistoryRepository) GetLatestHistory(templateID uint) (*model.DeploymentTemplateHistory, error) {
	var history model.DeploymentTemplateHistory
	if err := r.db.Where("deployment_template_id = ?", templateID).Order("created_at DESC").First(&history).Error; err != nil {
		return nil, err
	}
	return &history, nil
}
