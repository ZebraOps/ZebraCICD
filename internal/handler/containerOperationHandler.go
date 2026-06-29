package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"gorm.io/gorm"
)

type ContainerOperationRepository struct {
	db *gorm.DB
}

func NewContainerOperationRepository(db *gorm.DB) *ContainerOperationRepository {
	return &ContainerOperationRepository{db: db}
}

// Create saves a new container operation record.
func (r *ContainerOperationRepository) Create(op *model.ContainerOperation) error {
	return r.db.Create(op).Error
}

// List returns paginated container operation records, ordered by newest first.
func (r *ContainerOperationRepository) List(page, size int, filters map[string]interface{}) ([]model.ContainerOperation, int64, error) {
	var ops []model.ContainerOperation
	var total int64

	query := r.db.Model(&model.ContainerOperation{})

	if v, ok := filters["operation_type"]; ok && v != "" {
		query = query.Where("operation_type = ?", v)
	}
	if v, ok := filters["target_type"]; ok && v != "" {
		query = query.Where("target_type = ?", v)
	}
	if v, ok := filters["result"]; ok && v != "" {
		query = query.Where("result = ?", v)
	}
	if v, ok := filters["operator"]; ok && v != "" {
		query = query.Where("operator LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filters["start_time"]; ok && v != "" {
		query = query.Where("created_at >= ?", v)
	}
	if v, ok := filters["end_time"]; ok && v != "" {
		query = query.Where("created_at <= ?", v)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&ops).Error
	return ops, total, err
}
