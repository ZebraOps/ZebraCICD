package handler

import (
	"strings"

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

// Delete 删除模板，并删除关联的历史记录和多对多关联
func (r *BuildTemplateRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除关联的历史记录
		if err := tx.Where("template_id = ?", id).Delete(&model.TemplateHistory{}).Error; err != nil {
			return err
		}

		// 2. 删除多对多关联 build_template_applications
		if err := tx.Exec("DELETE FROM build_template_applications WHERE build_template_id = ?", id).Error; err != nil {
			return err
		}

		// 3. 删除历史遗留的 repo_templates 关联表记录
		if err := tx.Exec("DELETE FROM repo_templates WHERE build_template_id = ?", id).Error; err != nil {
			// repo_templates 表可能不存在（新版本已废弃），忽略错误
			// 只处理外键约束导致的冲突，表不存在时此处会跳过
			if !isTableNotExistError(err) {
				return err
			}
		}

		// 4. 删除模板记录
		return tx.Delete(&model.BuildTemplate{}, id).Error
	})
}

// isTableNotExistError 判断是否为"表不存在"类型的错误
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// PostgreSQL: relation/table does not exist
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not exist") ||
		strings.Contains(msg, "no such table")
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