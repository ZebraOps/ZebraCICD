package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type CloudProviderRepository struct {
	db *gorm.DB
}

func NewCloudProviderRepository(db *gorm.DB) *CloudProviderRepository {
	return &CloudProviderRepository{db: db}
}

// Create 创建云厂商
func (r *CloudProviderRepository) Create(provider *model.CloudProvider) error {
	return r.db.Create(provider).Error
}

// GetByID 根据ID获取云厂商
func (r *CloudProviderRepository) GetByID(id uint) (*model.CloudProvider, error) {
	var provider model.CloudProvider
	if err := r.db.First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

// GetByProvider 根据提供商标识获取云厂商
func (r *CloudProviderRepository) GetByProvider(provider string) (*model.CloudProvider, error) {
	var cloudProvider model.CloudProvider
	if err := r.db.Where("provider = ? AND is_deleted = ?", provider, false).First(&cloudProvider).Error; err != nil {
		return nil, err
	}
	return &cloudProvider, nil
}

// ListWithConditions 根据条件分页获取云厂商列表
func (r *CloudProviderRepository) ListWithConditions(conditions types.CloudProviderQueryConditions, page, size int) ([]model.CloudProvider, int64, error) {
	var providers []model.CloudProvider
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.CloudProvider{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.Provider != "" {
		db = db.Where("provider = ?", conditions.Provider)
	}

	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&providers).Error; err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// Update 更新云厂商
func (r *CloudProviderRepository) Update(provider *model.CloudProvider) error {
	return r.db.Save(provider).Error
}

// Delete 删除云厂商
func (r *CloudProviderRepository) Delete(id uint) error {
	return r.db.Model(&model.CloudProvider{}).Where("id = ?", id).Delete(&model.CloudProvider{}).Error
}
