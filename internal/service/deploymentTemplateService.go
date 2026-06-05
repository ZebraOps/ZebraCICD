package service

import (
	"context"
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type DeploymentTemplateService struct {
	templateRepo *handler.DeploymentTemplateRepository
	historyRepo  *handler.DeploymentTemplateHistoryRepository
}

func NewDeploymentTemplateService(templateRepo *handler.DeploymentTemplateRepository, historyRepo *handler.DeploymentTemplateHistoryRepository) *DeploymentTemplateService {
	return &DeploymentTemplateService{
		templateRepo: templateRepo,
		historyRepo:  historyRepo,
	}
}

// CreateDeploymentTemplate 创建部署模板并保存历史记录
func (s *DeploymentTemplateService) CreateDeploymentTemplate(template *model.DeploymentTemplate) error {
	if err := s.templateRepo.Create(template); err != nil {
		return err
	}

	// 创建初始历史记录
	history := &model.DeploymentTemplateHistory{
		DeploymentTemplateID: template.ID,
		Modifier:             template.Creator,
		Name:                 template.Name,
		DisplayName:          template.DisplayName,
		Description:          template.Description,
		TemplateType:         template.TemplateType,
		Content:              template.Content,
		Variables:            template.Variables,
		Version:              template.Version,
		ChangeReason:         "创建模板",
		CreatedAt:            template.CreatedAt,
	}
	return s.historyRepo.Create(history)
}

// GetDeploymentTemplateByID 根据ID获取部署模板
func (s *DeploymentTemplateService) GetDeploymentTemplateByID(ctx context.Context, id uint) (*model.DeploymentTemplate, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// ListDeploymentTemplatesWithConditions 根据条件分页获取部署模板列表
func (s *DeploymentTemplateService) ListDeploymentTemplatesWithConditions(conditions types.DeploymentTemplateQueryConditions, page, size int) ([]model.DeploymentTemplate, int64, error) {
	return s.templateRepo.ListWithConditions(conditions, page, size)
}

// UpdateDeploymentTemplate 更新部署模板并保存历史记录
func (s *DeploymentTemplateService) UpdateDeploymentTemplate(ctx context.Context, template *model.DeploymentTemplate, changeReason string) error {
	// 先获取旧模板信息用于创建历史记录
	_, err := s.templateRepo.GetByID(ctx, template.ID)
	if err != nil {
		return err
	}

	// 更新模板
	if err := s.templateRepo.Update(template); err != nil {
		return err
	}

	// 创建新的历史记录
	history := &model.DeploymentTemplateHistory{
		DeploymentTemplateID: template.ID,
		Modifier:             template.Updater,
		Name:                 template.Name,
		DisplayName:          template.DisplayName,
		Description:          template.Description,
		TemplateType:         template.TemplateType,
		Content:              template.Content,
		Variables:            template.Variables,
		Version:              template.Version,
		ChangeReason:         changeReason,
		CreatedAt:            template.UpdatedAt,
	}
	return s.historyRepo.Create(history)
}

// DeleteDeploymentTemplate 删除部署模板
func (s *DeploymentTemplateService) DeleteDeploymentTemplate(id uint) error {
	return s.templateRepo.Delete(id)
}

// AddApplicationToTemplate 添加应用到部署模板关联
func (s *DeploymentTemplateService) AddApplicationToTemplate(templateID, applicationID uint) error {
	return s.templateRepo.AddApplicationToTemplate(templateID, applicationID)
}

// RemoveApplicationFromTemplate 从部署模板移除应用关联
func (s *DeploymentTemplateService) RemoveApplicationFromTemplate(templateID, applicationID uint) error {
	return s.templateRepo.RemoveApplicationFromTemplate(templateID, applicationID)
}

// GetApplicationsByTemplateID 根据部署模板ID获取关联的应用列表
func (s *DeploymentTemplateService) GetApplicationsByTemplateID(templateID uint) ([]model.ApplicationResponse, error) {
	return s.templateRepo.GetApplicationsByTemplateID(templateID)
}

// GetDeploymentTemplateHistory 获取部署模板历史记录
func (s *DeploymentTemplateService) GetDeploymentTemplateHistory(templateID uint) ([]model.DeploymentTemplateHistory, error) {
	return s.historyRepo.GetHistoryByTemplateID(templateID)
}

// GetDeploymentTemplateHistoryPaginated 获取部署模板历史记录（分页）
func (s *DeploymentTemplateService) GetDeploymentTemplateHistoryPaginated(templateID uint, page, size int) ([]types.DeploymentTemplateHistoryResponse, int64, error) {
	return s.historyRepo.GetHistoryByTemplateIDPaginated(templateID, page, size)
}

// GetLatestHistory 获取最新的历史记录
func (s *DeploymentTemplateService) GetLatestHistory(templateID uint) (*model.DeploymentTemplateHistory, error) {
	return s.historyRepo.GetLatestHistory(templateID)
}

// RollbackDeploymentTemplate 回退部署模板到指定历史版本
func (s *DeploymentTemplateService) RollbackDeploymentTemplate(templateID, historyID uint) (*model.DeploymentTemplate, error) {
	history, err := s.historyRepo.GetByID(historyID)
	if err != nil {
		return nil, fmt.Errorf("历史记录不存在")
	}
	if history.DeploymentTemplateID != templateID {
		return nil, fmt.Errorf("历史记录与模板不匹配")
	}

	template, err := s.templateRepo.GetByID(context.Background(), templateID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在")
	}

	template.Name = history.Name
	template.DisplayName = history.DisplayName
	template.Description = history.Description
	template.TemplateType = history.TemplateType
	template.Content = history.Content
	template.Variables = history.Variables
	template.Version = history.Version
	template.Updater = history.Modifier

	if err := s.templateRepo.Update(template); err != nil {
		return nil, fmt.Errorf("回退失败: %v", err)
	}

	rollbackHistory := &model.DeploymentTemplateHistory{
		DeploymentTemplateID: template.ID,
		Modifier:             template.Updater,
		Name:                 template.Name,
		DisplayName:          template.DisplayName,
		Description:          template.Description,
		TemplateType:         template.TemplateType,
		Content:              template.Content,
		Variables:            template.Variables,
		Version:              template.Version,
		ChangeReason:         "回退到历史版本 #" + fmt.Sprintf("%d", historyID),
	}
	_ = s.historyRepo.Create(rollbackHistory)

	return template, nil
}