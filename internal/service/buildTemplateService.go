package service

import (
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
func (s *BuildTemplateService) ListTemplates(name, language, creator, updater string, page, size int) ([]model.BuildTemplateResponse, int64, error) {
	return s.templateRepo.List(name, language, creator, updater, page, size)
}

// GetTemplateHistoryPaginated 获取模板修改历史（分页）
func (s *BuildTemplateService) GetTemplateHistoryPaginated(templateID uint, page, size int) ([]types.TemplateHistoryResponse, int64, error) {
	return s.historyRepo.GetHistoryByTemplateIDPaginated(templateID, page, size)
}

// UpdateTemplate 更新模板并保存历史记录
func (s *BuildTemplateService) UpdateTemplate(template *model.BuildTemplate) error {
	// 先获取旧模板信息用于创建历史记录
	//oldTemplate, err := s.templateRepo.GetByID(template.ID)
	//if err != nil {
	//	return err
	//}

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

// AddRepoToTemplate 关联模板和仓库
func (s *BuildTemplateService) AddRepoToTemplate(templateID, repoID uint) error {
	return s.templateRepo.AddRepoToTemplate(templateID, repoID)
}

// RemoveRepoFromTemplate 取消模板和仓库关联
func (s *BuildTemplateService) RemoveRepoFromTemplate(templateID, repoID uint) error {
	return s.templateRepo.RemoveRepoFromTemplate(templateID, repoID)
}

// GetTemplatesByRepoID 根据仓库ID获取模板
func (s *BuildTemplateService) GetTemplatesByRepoID(repoID uint) ([]model.BuildTemplate, error) {
	return s.templateRepo.GetTemplatesByRepoID(repoID)
}
