package handler

import (
	"gorm.io/gorm"

	"github.com/ZebraOps/ZebraCICD/internal/model"
)

type BuildTemplateRepository struct {
	db *gorm.DB
}

func NewBuildTemplateRepository(db *gorm.DB) *BuildTemplateRepository {
	return &BuildTemplateRepository{db: db}
}

// Create 创建模板
func (r *BuildTemplateRepository) Create(template *model.BuildTemplate) error {
	return r.db.Create(template).Error
}

// GetByID 根据ID获取模板
func (r *BuildTemplateRepository) GetByID(id uint) (*model.BuildTemplate, error) {
	var template model.BuildTemplate
	if err := r.db.Preload("Repos").First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// List 获取模板列表并返回总数，支持过滤和分页
func (r *BuildTemplateRepository) List(name, language, creator, updater string, page, size int) ([]model.BuildTemplateResponse, int64, error) {
	var templates []model.BuildTemplateResponse
	var count int64

	// 构建查询条件
	query := r.db.Model(&model.BuildTemplate{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if language != "" {
		query = query.Where("language LIKE ?", "%"+language+"%")
	}
	if creator != "" {
		query = query.Where("creator LIKE ?", "%"+creator+"%")
	}

	if updater != "" {
		query = query.Where("updater LIKE ?", "%"+updater+"%")
	}

	// 获取总数
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Order("id DESC").Offset(offset).Limit(size).Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, count, nil
}

// Update 更新模板
func (r *BuildTemplateRepository) Update(template *model.BuildTemplate) error {
	return r.db.Save(template).Error
}

// Delete 删除模板
func (r *BuildTemplateRepository) Delete(id uint) error {
	// 先删除关联的历史记录
	if err := r.db.Where("template_id = ?", id).Delete(&model.TemplateHistory{}).Error; err != nil {
		return err
	}

	// 再删除模板记录
	return r.db.Delete(&model.BuildTemplate{}, id).Error
}

// AddRepoToTemplate 添加模板到仓库关联
func (r *BuildTemplateRepository) AddRepoToTemplate(templateID, repoID uint) error {
	template := &model.BuildTemplate{ID: templateID}
	repo := &model.Repo{ID: repoID}
	return r.db.Model(template).Association("Repos").Append(repo)
}

// RemoveRepoFromTemplate 移除仓库与模板的关联
func (r *BuildTemplateRepository) RemoveRepoFromTemplate(templateID, repoID uint) error {
	template := &model.BuildTemplate{ID: templateID}
	repo := &model.Repo{ID: repoID}
	return r.db.Model(template).Association("Repos").Delete(repo)
}

// GetTemplatesByRepoID 根据仓库ID获取关联的模板
func (r *BuildTemplateRepository) GetTemplatesByRepoID(repoID uint) ([]model.BuildTemplate, error) {
	var repo model.Repo
	if err := r.db.Preload("Templates").First(&repo, repoID).Error; err != nil {
		return nil, err
	}

	// 将 []*model.BuildTemplate 转换为 []model.BuildTemplate
	templates := make([]model.BuildTemplate, len(repo.Templates))
	for i, template := range repo.Templates {
		templates[i] = *template
	}

	return templates, nil
}
