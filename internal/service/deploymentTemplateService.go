package service

import (
	"context"

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
		Parameters:           template.Parameters,
		Version:              template.Version,
		ChangeReason:         "创建模板",
		CreatedAt:            template.CreatedAt,
	}
	return s.historyRepo.Create(history)
}

// GetDeploymentTemplateByID 根据ID获取部署模板
func (s *DeploymentTemplateService) GetDeploymentTemplateByID(ctx context.Context, id uint) (*model.DeploymentTemplate, error) {
	// 假设 tracer 已经通过某种方式获取（如从上下文或依赖注入）
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
		Parameters:           template.Parameters,
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

// AddRepoToTemplate 添加仓库到部署模板关联
func (s *DeploymentTemplateService) AddRepoToTemplate(templateID, repoID uint) error {
	return s.templateRepo.AddRepoToTemplate(templateID, repoID)
}

// RemoveRepoFromTemplate 从部署模板移除仓库关联
func (s *DeploymentTemplateService) RemoveRepoFromTemplate(templateID, repoID uint) error {
	return s.templateRepo.RemoveRepoFromTemplate(templateID, repoID)
}

// GetReposByTemplateID 根据部署模板ID获取关联的仓库列表
func (s *DeploymentTemplateService) GetReposByTemplateID(templateID uint) ([]model.Repo, error) {
	return s.templateRepo.GetReposByTemplateID(templateID)
}

// GetDeploymentTemplateHistory 获取部署模板历史记录
func (s *DeploymentTemplateService) GetDeploymentTemplateHistory(templateID uint) ([]model.DeploymentTemplateHistory, error) {
	return s.historyRepo.GetHistoryByTemplateID(templateID)
}

// GetLatestHistory 获取最新的历史记录
func (s *DeploymentTemplateService) GetLatestHistory(templateID uint) (*model.DeploymentTemplateHistory, error) {
	return s.historyRepo.GetLatestHistory(templateID)
}
