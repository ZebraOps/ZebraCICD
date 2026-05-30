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
	if err := r.db.First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// List 获取模板列表并返回总数，支持过滤和分页
func (r *BuildTemplateRepository) List(name, language, department, creator, updater string, page, size int) ([]model.BuildTemplateResponse, int64, error) {
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
	if department != "" {
		query = query.Where("department LIKE ?", "%"+department+"%")
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

	// 删除多对多关联 build_template_applications
	if err := r.db.Exec("DELETE FROM build_template_applications WHERE build_template_id = ?", id).Error; err != nil {
		return err
	}

	// 再删除模板记录
	return r.db.Delete(&model.BuildTemplate{}, id).Error
}

// AddApplicationToTemplate 添加模板到应用关联
func (r *BuildTemplateRepository) AddApplicationToTemplate(templateID, applicationID uint) error {
	template := &model.BuildTemplate{ID: templateID}
	app := &model.Application{ID: applicationID}
	return r.db.Model(template).Association("Applications").Append(app)
}

// RemoveApplicationFromTemplate 移除应用与模板的关联
func (r *BuildTemplateRepository) RemoveApplicationFromTemplate(templateID, applicationID uint) error {
	template := &model.BuildTemplate{ID: templateID}
	app := &model.Application{ID: applicationID}
	return r.db.Model(template).Association("Applications").Delete(app)
}

// GetApplicationsByTemplateID 根据构建模板ID获取关联的应用列表
func (r *BuildTemplateRepository) GetApplicationsByTemplateID(templateID uint) ([]model.ApplicationResponse, error) {
	var template model.BuildTemplate
	if err := r.db.Preload("Applications").First(&template, templateID).Error; err != nil {
		return nil, err
	}

	if len(template.Applications) == 0 {
		return []model.ApplicationResponse{}, nil
	}

	var responses []model.ApplicationResponse
	for _, app := range template.Applications {
		// 获取关联仓库以补充 department 和 language
		var repo model.Repo
		if err := r.db.First(&repo, app.RepoID).Error; err != nil {
			repo = model.Repo{}
		}

		resp := model.ApplicationResponse{
			ID:              app.ID,
			RepoID:          app.RepoID,
			CName:           app.CName,
			EName:           app.EName,
			ListenPort:      app.ListenPort,
			HealthCheckType: app.HealthCheckType,
			HealthCheckURL:  app.HealthCheckURL,
			Description:     app.Description,
			CreatedAt:       app.CreatedAt,
			UpdatedAt:       app.UpdatedAt,
			Department:      repo.RepoDepartment,
			Language:        repo.RepoLanguage,
		}
		responses = append(responses, resp)
	}

	return responses, nil
}