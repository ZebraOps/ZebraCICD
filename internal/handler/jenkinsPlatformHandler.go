package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type JenkinsPlatformRepository struct {
	db *gorm.DB
}

func NewJenkinsPlatformRepository(db *gorm.DB) *JenkinsPlatformRepository {
	return &JenkinsPlatformRepository{db: db}
}

func (r *JenkinsPlatformRepository) Create(platform *model.JenkinsPlatform) error {
	return r.db.Create(platform).Error
}

func (r *JenkinsPlatformRepository) GetByID(id uint) (*model.JenkinsPlatform, error) {
	var platform model.JenkinsPlatform
	if err := r.db.First(&platform, id).Error; err != nil {
		return nil, err
	}
	return &platform, nil
}

func (r *JenkinsPlatformRepository) ListWithConditions(conditions types.JenkinsPlatformQueryConditions, page, size int) ([]model.JenkinsPlatform, int64, error) {
	var platforms []model.JenkinsPlatform
	var total int64

	offset := (page - 1) * size
	db := r.db.Model(&model.JenkinsPlatform{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}
	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&platforms).Error; err != nil {
		return nil, 0, err
	}
	return platforms, total, nil
}

func (r *JenkinsPlatformRepository) Update(platform *model.JenkinsPlatform) error {
	return r.db.Save(platform).Error
}

func (r *JenkinsPlatformRepository) Delete(id uint) error {
	return r.db.Delete(&model.JenkinsPlatform{}, id).Error
}