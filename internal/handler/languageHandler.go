package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type LanguageRepository struct {
	db *gorm.DB
}

func NewLanguageRepository(db *gorm.DB) *LanguageRepository {
	return &LanguageRepository{db: db}
}

// Create 创建开发语言
func (r *LanguageRepository) Create(lang *model.Language) error {
	return r.db.Create(lang).Error
}

// GetByID 根据ID获取开发语言
func (r *LanguageRepository) GetByID(id uint) (*model.Language, error) {
	var lang model.Language
	if err := r.db.First(&lang, id).Error; err != nil {
		return nil, err
	}
	return &lang, nil
}

// ListWithConditions 根据条件分页获取开发语言列表
func (r *LanguageRepository) ListWithConditions(conditions types.LanguageQueryConditions, page, size int) ([]model.Language, int64, error) {
	var langs []model.Language
	var total int64

	offset := (page - 1) * size

	db := r.db.Model(&model.Language{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(size).Order("sort_order ASC, id DESC").Find(&langs).Error; err != nil {
		return nil, 0, err
	}

	return langs, total, nil
}

// Update 更新开发语言
func (r *LanguageRepository) Update(lang *model.Language) error {
	return r.db.Save(lang).Error
}

// Delete 删除开发语言
func (r *LanguageRepository) Delete(id uint) error {
	return r.db.Delete(&model.Language{}, id).Error
}
