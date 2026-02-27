package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type ServerRepository struct {
	db *gorm.DB
}

func NewServerRepository(db *gorm.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Create(server *model.Server) error {
	return r.db.Create(server).Error
}

func (r *ServerRepository) GetByID(id uint) (*model.Server, error) {
	var server model.Server
	if err := r.db.First(&server, id).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

func (r *ServerRepository) List() ([]model.Server, error) {
	var servers []model.Server
	if err := r.db.Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

func (r *ServerRepository) Update(server *model.Server) error {
	return r.db.Save(server).Error
}

func (r *ServerRepository) Delete(id uint) error {
	return r.db.Delete(&model.Server{}, id).Error
}

// ListWithConditions 根据条件分页获取服务器列表
func (r *ServerRepository) ListWithConditions(conditions types.ServerQueryConditions, page, size int) ([]model.Server, int64, error) {
	var servers []model.Server
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.Server{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.Host != "" {
		db = db.Where("host LIKE ?", "%"+conditions.Host+"%")
	}

	if conditions.IsActive != nil {
		db = db.Where("is_active = ?", *conditions.IsActive)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&servers).Error; err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}
