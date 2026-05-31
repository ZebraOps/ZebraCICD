package handler

import (
	"context"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type DeploymentTemplateRepository struct {
	db *gorm.DB
}

func NewDeploymentTemplateRepository(db *gorm.DB) *DeploymentTemplateRepository {
	return &DeploymentTemplateRepository{db: db}
}

// Create 创建部署模板
func (r *DeploymentTemplateRepository) Create(template *model.DeploymentTemplate) error {
	return r.db.Create(template).Error
}

// GetByID 修改函数签名，增加 ctx 参数（保留原 ctx）
func (r *DeploymentTemplateRepository) GetByID(ctx context.Context, id uint) (*model.DeploymentTemplate, error) {
	var template model.DeploymentTemplate
	if err := r.db.WithContext(ctx).First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// ListWithConditions 根据条件分页获取部署模板列表
func (r *DeploymentTemplateRepository) ListWithConditions(conditions types.DeploymentTemplateQueryConditions, page, size int) ([]model.DeploymentTemplate, int64, error) {
	var templates []model.DeploymentTemplate
	var total int64

	offset := (page - 1) * size

	// 构建查询条件
	db := r.db.Model(&model.DeploymentTemplate{})

	if conditions.Name != "" {
		db = db.Where("name LIKE ?", "%"+conditions.Name+"%")
	}

	if conditions.TemplateType != "" {
		db = db.Where("template_type = ?", conditions.TemplateType)
	}

	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	if conditions.Creator != "" {
		db = db.Where("creator LIKE ?", "%"+conditions.Creator+"%")
	}

	if conditions.Department != "" {
		db = db.Where("department LIKE ?", "%"+conditions.Department+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// Update 更新部署模板
func (r *DeploymentTemplateRepository) Update(template *model.DeploymentTemplate) error {
	return r.db.Save(template).Error
}

// Delete 删除部署模板，并删除关联的历史修改记录
func (r *DeploymentTemplateRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除关联的历史修改记录
		if err := tx.Where("deployment_template_id = ?", id).Delete(&model.DeploymentTemplateHistory{}).Error; err != nil {
			return err
		}
		// 2. 删除多对多关联 deployment_template_applications
		if err := tx.Exec("DELETE FROM deployment_template_applications WHERE deployment_template_id = ?", id).Error; err != nil {
			return err
		}
		// 3. 删除历史遗留的 repo_templates 关联表记录（部署模板也可能有残留）
		if err := tx.Exec("DELETE FROM repo_templates WHERE deployment_template_id = ?", id).Error; err != nil {
			if !isTableNotExistError(err) {
				return err
			}
		}
		// 4. 删除部署模板（硬删除）
		if err := tx.Where("id = ?", id).Delete(&model.DeploymentTemplate{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// AddApplicationToTemplate 添加应用到部署模板关联
func (r *DeploymentTemplateRepository) AddApplicationToTemplate(templateID, applicationID uint) error {
	template := &model.DeploymentTemplate{ID: templateID}
	app := &model.Application{ID: applicationID}
	return r.db.Model(template).Association("Applications").Append(app)
}

// RemoveApplicationFromTemplate 从部署模板移除应用关联
func (r *DeploymentTemplateRepository) RemoveApplicationFromTemplate(templateID, applicationID uint) error {
	template := &model.DeploymentTemplate{ID: templateID}
	app := &model.Application{ID: applicationID}
	return r.db.Model(template).Association("Applications").Delete(app)
}

// GetApplicationsByTemplateID 根据部署模板ID获取关联的应用列表
func (r *DeploymentTemplateRepository) GetApplicationsByTemplateID(templateID uint) ([]model.ApplicationResponse, error) {
	var template model.DeploymentTemplate
	if err := r.db.Preload("Applications").First(&template, templateID).Error; err != nil {
		return nil, err
	}

	if len(template.Applications) == 0 {
		return []model.ApplicationResponse{}, nil
	}

	var responses []model.ApplicationResponse
	for _, app := range template.Applications {
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