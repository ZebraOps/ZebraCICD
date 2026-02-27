package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type EnvRepository struct {
	db *gorm.DB
}

func NewEnvRepository(db *gorm.DB) *EnvRepository {
	return &EnvRepository{db: db}
}

// Create 创建环境
func (r *EnvRepository) Create(env *model.Environment) error {
	return r.db.Create(env).Error
}

// GetByID 根据ID获取环境
func (r *EnvRepository) GetByID(id uint) (*model.Environment, error) {
	var env model.Environment
	if err := r.db.First(&env, id).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

// ListWithConditions 根据条件分页获取环境列表
func (r *EnvRepository) ListWithConditions(conditions types.EnvQueryConditions, page, size int) ([]model.Environment, int64, error) {
	var envs []model.Environment
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.Environment{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.Type != "" {
		db = db.Where("type = ?", conditions.Type)
	}

	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&envs).Error; err != nil {
		return nil, 0, err
	}

	return envs, total, nil
}

// Update 更新环境
func (r *EnvRepository) Update(env *model.Environment) error {
	return r.db.Save(env).Error
}

// Delete 删除环境
func (r *EnvRepository) Delete(id uint) error {
	return r.db.Model(&model.Environment{}).Where("id = ?", id).Delete(&model.Environment{}).Error
}

// HardDelete 硬删除环境
func (r *EnvRepository) HardDelete(id uint) error {
	return r.db.Delete(&model.Environment{}, id).Error
}
