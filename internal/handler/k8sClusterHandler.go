package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type K8SClusterRepository struct {
	db *gorm.DB
}

func NewK8SClusterRepository(db *gorm.DB) *K8SClusterRepository {
	return &K8SClusterRepository{db: db}
}

func (r *K8SClusterRepository) Create(cluster *model.K8SCluster) error {
	return r.db.Create(cluster).Error
}

func (r *K8SClusterRepository) GetByID(id uint) (*model.K8SCluster, error) {
	var cluster model.K8SCluster
	if err := r.db.First(&cluster, id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *K8SClusterRepository) List() ([]model.K8SCluster, error) {
	var clusters []model.K8SCluster
	if err := r.db.Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func (r *K8SClusterRepository) Update(cluster *model.K8SCluster) error {
	return r.db.Save(cluster).Error
}

func (r *K8SClusterRepository) Delete(id uint) error {
	return r.db.Delete(&model.K8SCluster{}, id).Error
}

// ListWithConditions 根据条件分页获取集群列表
func (r *K8SClusterRepository) ListWithConditions(conditions types.ClusterQueryConditions, page, size int) ([]model.K8SCluster, int64, error) {
	var clusters []model.K8SCluster
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.K8SCluster{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.Enabled != nil {
		db = db.Where("enabled = ?", *conditions.Enabled)
	}

	if conditions.Vendor != "" {
		db = db.Where("vendor = ?", conditions.Vendor)
	}

	if conditions.Environment != "" {
		db = db.Where("environment = ?", conditions.Environment)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&clusters).Error; err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
}
