package handler

import (
	"gorm.io/gorm"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type TemplateHistoryRepository struct {
	db *gorm.DB
}

func NewTemplateHistoryRepository(db *gorm.DB) *TemplateHistoryRepository {
	return &TemplateHistoryRepository{db: db}
}

// Create 创建模板历史记录
func (r *TemplateHistoryRepository) Create(history *model.TemplateHistory) error {
	return r.db.Create(history).Error
}

// GetHistoryByTemplateIDPaginated 根据模板ID获取历史记录（分页）
func (r *TemplateHistoryRepository) GetHistoryByTemplateIDPaginated(templateID uint, page, size int) ([]types.TemplateHistoryResponse, int64, error) {
	var histories []model.TemplateHistory
	var count int64

	// 查询总数
	if err := r.db.Model(&model.TemplateHistory{}).Where("template_id = ?", templateID).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := r.db.Where("template_id = ?", templateID).Order("created_at DESC").Offset(offset).Limit(size).Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	// 转换为响应类型
	responses := make([]types.TemplateHistoryResponse, len(histories))
	for i, history := range histories {
		responses[i] = types.TemplateHistoryResponse{
			ID:         history.ID,
			TemplateID: history.TemplateID,
			Modifier:   history.Modifier,
			Dockerfile: history.Dockerfile,
			Pipeline:   history.Pipeline,
			CreatedAt:  history.CreatedAt,
		}
	}

	return responses, count, nil
}
