package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type GitPlatformRepository struct {
	db *gorm.DB
}

func NewGitPlatformRepository(db *gorm.DB) *GitPlatformRepository {
	return &GitPlatformRepository{db: db}
}

// Create 创建Git平台配置
func (r *GitPlatformRepository) Create(platform *model.GitPlatform) error {
	return r.db.Create(platform).Error
}

// GetByID 根据ID获取Git平台配置
func (r *GitPlatformRepository) GetByID(id uint) (*model.GitPlatform, error) {
	var platform model.GitPlatform
	if err := r.db.First(&platform, id).Error; err != nil {
		return nil, err
	}
	return &platform, nil
}

// ListWithConditions 根据条件分页获取Git平台配置列表
func (r *GitPlatformRepository) ListWithConditions(conditions types.GitPlatformQueryConditions, page, size int) ([]model.GitPlatform, int64, error) {
	var platforms []model.GitPlatform
	var total int64

	offset := (page - 1) * size

	db := r.db.Model(&model.GitPlatform{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.PlatformType != "" {
		db = db.Where("platform_type = ?", conditions.PlatformType)
	}

	if conditions.AuthType != "" {
		db = db.Where("auth_type = ?", conditions.AuthType)
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

// Update 更新Git平台配置
func (r *GitPlatformRepository) Update(platform *model.GitPlatform) error {
	return r.db.Save(platform).Error
}

// Delete 删除Git平台配置
func (r *GitPlatformRepository) Delete(id uint) error {
	return r.db.Delete(&model.GitPlatform{}, id).Error
}