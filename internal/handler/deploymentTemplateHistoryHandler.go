package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type DeploymentTemplateHistoryRepository struct {
	db *gorm.DB
}

func NewDeploymentTemplateHistoryRepository(db *gorm.DB) *DeploymentTemplateHistoryRepository {
	return &DeploymentTemplateHistoryRepository{db: db}
}

// GetByID 根据ID获取历史记录
func (r *DeploymentTemplateHistoryRepository) GetByID(id uint) (*model.DeploymentTemplateHistory, error) {
	var history model.DeploymentTemplateHistory
	if err := r.db.First(&history, id).Error; err != nil {
		return nil, err
	}
	return &history, nil
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

// GetHistoryByTemplateIDPaginated 根据部署模板ID获取历史记录（分页）
func (r *DeploymentTemplateHistoryRepository) GetHistoryByTemplateIDPaginated(templateID uint, page, size int) ([]types.DeploymentTemplateHistoryResponse, int64, error) {
	var histories []model.DeploymentTemplateHistory
	var count int64

	// 查询总数
	if err := r.db.Model(&model.DeploymentTemplateHistory{}).Where("deployment_template_id = ?", templateID).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := r.db.Where("deployment_template_id = ?", templateID).Order("created_at DESC").Offset(offset).Limit(size).Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	// 转换为响应类型
	responses := make([]types.DeploymentTemplateHistoryResponse, len(histories))
	for i, history := range histories {
		responses[i] = types.DeploymentTemplateHistoryResponse{
			ID:                   history.ID,
			DeploymentTemplateID: history.DeploymentTemplateID,
			Modifier:             history.Modifier,
			Name:                 history.Name,
			Description:          history.Description,
			TemplateType:         history.TemplateType,
			Content:              history.Content,
			Variables:            history.Variables,
			Version:              history.Version,
			ChangeReason:         history.ChangeReason,
			CreatedAt:            history.CreatedAt,
		}
	}

	return responses, count, nil
}
