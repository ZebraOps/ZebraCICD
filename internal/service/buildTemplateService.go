package service

import (
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type BuildTemplateService struct {
	templateRepo *handler.BuildTemplateRepository
	historyRepo  *handler.TemplateHistoryRepository
}

func NewBuildTemplateService(templateRepo *handler.BuildTemplateRepository, historyRepo *handler.TemplateHistoryRepository) *BuildTemplateService {
	return &BuildTemplateService{
		templateRepo: templateRepo,
		historyRepo:  historyRepo,
	}
}

// CreateTemplate 创建模板并保存历史记录
func (s *BuildTemplateService) CreateTemplate(template *model.BuildTemplate) error {
	if err := s.templateRepo.Create(template); err != nil {
		return err
	}

	// 创建初始历史记录
	history := &model.TemplateHistory{
		TemplateID: template.ID,
		Modifier:   template.Creator,
		Dockerfile: template.Dockerfile,
		Pipeline:   template.Pipeline,
		CreatedAt:  template.CreatedAt,
	}
	return s.historyRepo.Create(history)
}

// GetTemplate 获取模板
func (s *BuildTemplateService) GetTemplate(id uint) (*model.BuildTemplate, error) {
	return s.templateRepo.GetByID(id)
}

// ListTemplates 获取模板列表，支持过滤和分页
func (s *BuildTemplateService) ListTemplates(name, language, department, creator, updater string, page, size int) ([]model.BuildTemplateResponse, int64, error) {
	return s.templateRepo.List(name, language, department, creator, updater, page, size)
}

// GetTemplateHistoryPaginated 获取模板修改历史（分页）
func (s *BuildTemplateService) GetTemplateHistoryPaginated(templateID uint, page, size int) ([]types.TemplateHistoryResponse, int64, error) {
	return s.historyRepo.GetHistoryByTemplateIDPaginated(templateID, page, size)
}

// UpdateTemplate 更新模板并保存历史记录
func (s *BuildTemplateService) UpdateTemplate(template *model.BuildTemplate) error {
	// 更新模板
	if err := s.templateRepo.Update(template); err != nil {
		return err
	}

	// 创建新的历史记录
	history := &model.TemplateHistory{
		TemplateID: template.ID,
		Modifier:   template.Updater,
		Dockerfile: template.Dockerfile,
		Pipeline:   template.Pipeline,
		CreatedAt:  template.UpdatedAt,
	}
	return s.historyRepo.Create(history)
}

// DeleteTemplate 删除模板
func (s *BuildTemplateService) DeleteTemplate(id uint) error {
	return s.templateRepo.Delete(id)
}

// AddApplicationToTemplate 关联模板和应用
func (s *BuildTemplateService) AddApplicationToTemplate(templateID, applicationID uint) error {
	return s.templateRepo.AddApplicationToTemplate(templateID, applicationID)
}

// RemoveApplicationFromTemplate 取消模板和应用关联
func (s *BuildTemplateService) RemoveApplicationFromTemplate(templateID, applicationID uint) error {
	return s.templateRepo.RemoveApplicationFromTemplate(templateID, applicationID)
}

// GetApplicationsByTemplateID 根据构建模板ID获取关联的应用列表
func (s *BuildTemplateService) GetApplicationsByTemplateID(templateID uint) ([]model.ApplicationResponse, error) {
	return s.templateRepo.GetApplicationsByTemplateID(templateID)
}

// RollbackTemplate 回退模板到指定历史版本
func (s *BuildTemplateService) RollbackTemplate(templateID, historyID uint) (*model.BuildTemplate, error) {
	// 获取历史记录
	var history model.TemplateHistory
	if err := s.historyRepo.GetByID(historyID, &history); err != nil {
		return nil, fmt.Errorf("历史记录不存在")
	}
	if history.TemplateID != templateID {
		return nil, fmt.Errorf("历史记录与模板不匹配")
	}

	// 获取当前模板
	template, err := s.templateRepo.GetByID(templateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在")
	}

	// 用历史版本的内容覆盖当前模板
	template.Dockerfile = history.Dockerfile
	template.Pipeline = history.Pipeline
	template.Updater = history.Modifier

	if err := s.templateRepo.Update(template); err != nil {
		return nil, fmt.Errorf("回退失败: %v", err)
	}

	// 创建新的历史记录（记录这次回退操作）
	rollbackHistory := &model.TemplateHistory{
		TemplateID: template.ID,
		Modifier:   template.Updater,
		Dockerfile: template.Dockerfile,
		Pipeline:   template.Pipeline,
	}
	_ = s.historyRepo.Create(rollbackHistory)

	return template, nil
}