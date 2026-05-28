package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

// ImageRepositoryRepository 提供镜像仓库的数据访问接口
type ImageRepositoryRepository struct {
	db *gorm.DB
}

// NewImageRepositoryRepository 创建一个新的 ImageRepositoryRepository 实例
func NewImageRepositoryRepository(db *gorm.DB) *ImageRepositoryRepository {
	return &ImageRepositoryRepository{db: db}
}

// Create 创建镜像仓库
func (r *ImageRepositoryRepository) Create(repository *model.ImageRepository) error {
	return r.db.Create(repository).Error
}

// GetByID 根据ID获取镜像仓库
func (r *ImageRepositoryRepository) GetByID(id uint) (*model.ImageRepository, error) {
	var repository model.ImageRepository
	if err := r.db.First(&repository, id).Error; err != nil {
		return nil, err
	}
	return &repository, nil
}

// List 获取镜像仓库列表
func (r *ImageRepositoryRepository) List() ([]model.ImageRepository, error) {
	var repositories []model.ImageRepository
	if err := r.db.Order("id DESC").Find(&repositories).Error; err != nil {
		return nil, err
	}
	return repositories, nil
}

// ListWithConditions 根据条件分页获取镜像仓库列表
func (r *ImageRepositoryRepository) ListWithConditions(conditions types.ImageRepositoryQueryConditions) ([]model.ImageRepository, int64, error) {
	var repositories []model.ImageRepository
	var total int64

	offset := (conditions.Page - 1) * conditions.Size

	// 构建查询条件
	db := r.db.Model(&model.ImageRepository{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.URL != "" {
		db = db.Where("url LIKE ?", "%"+conditions.URL+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(conditions.Size).Order("id DESC").Find(&repositories).Error; err != nil {
		return nil, 0, err
	}

	return repositories, total, nil
}

// Update 更新镜像仓库
func (r *ImageRepositoryRepository) Update(repository *model.ImageRepository) error {
	return r.db.Save(repository).Error
}

// Delete 删除镜像仓库
func (r *ImageRepositoryRepository) Delete(id uint) error {
	return r.db.Model(&model.ImageRepository{}).Where("id = ?", id).Delete(model.ImageRepository{}).Error
}
