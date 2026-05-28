package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type RepoRepository struct {
	db *gorm.DB
}

func NewRepoRepository(db *gorm.DB) *RepoRepository {
	return &RepoRepository{db: db}
}

func (r *RepoRepository) Create(repo *model.Repo) error {
	return r.db.Create(repo).Error
}

func (r *RepoRepository) GetByID(id uint) (*model.Repo, error) {
	var repo model.Repo
	if err := r.db.Preload("Templates").Preload("DeploymentTemplates").First(&repo, id).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *RepoRepository) List() ([]model.Repo, error) {
	var repos []model.Repo
	if err := r.db.Preload("Templates").Preload("DeploymentTemplates").Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *RepoRepository) Update(repo *model.Repo) error {
	return r.db.Save(repo).Error
}

func (r *RepoRepository) Delete(id uint) error {
	return r.db.Delete(&model.Repo{}, id).Error
}

func (r *RepoRepository) GetByEName(eName string) (*model.Repo, error) {
	var repo model.Repo
	if err := r.db.Preload("Templates").Preload("DeploymentTemplates").Where("e_name = ?", eName).First(&repo).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListWithConditions 根据条件分页获取仓库列表
func (r *RepoRepository) ListWithConditions(conditions types.RepoQueryConditions, page, size int) ([]model.RepoResp, int64, error) {
	var repos []model.RepoResp
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.Repo{})

	if conditions.CName != "" {
		db = db.Where("c_name LIKE ?", "%"+conditions.CName+"%")
	}

	if conditions.EName != "" {
		db = db.Where("e_name LIKE ?", "%"+conditions.EName+"%")
	}

	if conditions.Department != "" {
		db = db.Where("repo_department LIKE ?", "%"+conditions.Department+"%")
	}

	if conditions.Language != "" {
		db = db.Where("repo_language LIKE ?", "%"+conditions.Language+"%")
	}

	if conditions.Manager != "" {
		db = db.Where("repo_manager LIKE ?", "%"+conditions.Manager+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&repos).Error; err != nil {
		return nil, 0, err
	}
	// 获取分页数据，并预加载关联模板
	//if err := db.Offset(offset).Limit(size).Preload("Templates").Preload("DeploymentTemplates").Order("id DESC").Find(&repos).Error; err != nil {
	//	return nil, 0, err
	//}

	return repos, total, nil
}
