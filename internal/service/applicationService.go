package service

import (
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"gorm.io/gorm"
)

type ApplicationService struct {
	appRepo    *handler.ApplicationRepository
	deployRepo *handler.ApplicationDeploymentRepository
	db         *gorm.DB
}

func NewApplicationService(
	appRepo *handler.ApplicationRepository,
	deployRepo *handler.ApplicationDeploymentRepository,
	db *gorm.DB) *ApplicationService {
	return &ApplicationService{
		appRepo:    appRepo,
		deployRepo: deployRepo,
		db:         db,
	}
}

// CreateApplication 创建应用服务
func (s *ApplicationService) CreateApplication(req *model.ApplicationRequest) (*model.ApplicationResponse, error) {
	// 验证仓库是否存在
	var repo model.Repo
	if err := s.db.First(&repo, req.RepoID).Error; err != nil {
		return nil, fmt.Errorf("仓库不存在: %v", err)
	}

	// 验证应用服务名称唯一性（在同一仓库内）
	var existingApp model.Application
	if err := s.db.Where("repo_id = ? AND (c_name = ? OR e_name = ?)", req.RepoID, req.CName, req.EName).First(&existingApp).Error; err == nil {
		return nil, fmt.Errorf("应用服务名称已存在")
	}

	app := &model.Application{
		ApplicationRequest: *req,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, err
	}

	resp := &model.ApplicationResponse{
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
		DeploymentCount: 0,
	}
	return resp, nil
}

// GetApplicationByID 根据ID获取应用服务
func (s *ApplicationService) GetApplicationByID(id uint) (*model.Application, error) {
	return s.appRepo.GetByID(id)
}

// ListApplicationsByRepoID 根据仓库ID获取应用服务列表
func (s *ApplicationService) ListApplicationsByRepoID(repoID uint) ([]model.ApplicationResponse, error) {
	return s.appRepo.GetApplicationsWithDeploymentCount(repoID)
}

// UpdateApplication 更新应用服务
func (s *ApplicationService) UpdateApplication(id uint, req *model.ApplicationRequest) (*model.Application, error) {
	// 获取现有应用服务
	existingApp, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("应用服务不存在: %v", err)
	}

	// 验证名称唯一性（排除自己）
	var existingApp2 model.Application
	if err := s.db.Where("(c_name = ? OR e_name = ?) AND id != ?", req.CName, req.EName, id).First(&existingApp2).Error; err == nil {
		return nil, fmt.Errorf("应用服务名称已存在")
	}

	// 更新字段
	existingApp.CName = req.CName
	existingApp.EName = req.EName
	existingApp.ListenPort = req.ListenPort
	existingApp.HealthCheckType = req.HealthCheckType
	existingApp.HealthCheckURL = req.HealthCheckURL
	existingApp.Description = req.Description

	if err := s.appRepo.Update(existingApp); err != nil {
		return nil, err
	}

	return existingApp, nil
}

// DeleteApplication 删除应用服务
func (s *ApplicationService) DeleteApplication(id uint) error {
	// 检查是否存在关联的部署配置
	var count int64
	if err := s.db.Model(&model.ApplicationDeployment{}).Where("application_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("存在关联的部署配置，无法删除应用服务")
	}

	return s.appRepo.Delete(id)
}

// CreateApplicationDeployment 创建应用部署配置
func (s *ApplicationService) CreateApplicationDeployment(req *model.ApplicationDeploymentRequest) (*model.ApplicationDeployment, error) {
	// 验证应用服务是否存在
	var app model.Application
	if err := s.db.First(&app, req.ApplicationID).Error; err != nil {
		return nil, fmt.Errorf("应用服务不存在: %v", err)
	}

	// 验证环境是否存在
	var env model.Environment
	if err := s.db.First(&env, req.EnvironmentID).Error; err != nil {
		return nil, fmt.Errorf("环境不存在: %v", err)
	}

	// 验证构建模板（如果提供）
	if req.BuildTemplateID != nil {
		var buildTemplate model.BuildTemplate
		if err := s.db.First(&buildTemplate, *req.BuildTemplateID).Error; err != nil {
			return nil, fmt.Errorf("构建模板不存在: %v", err)
		}
	}

	// 验证部署模板（如果提供）
	if req.DeploymentTemplateID != nil {
		var deployTemplate model.DeploymentTemplate
		if err := s.db.First(&deployTemplate, *req.DeploymentTemplateID).Error; err != nil {
			return nil, fmt.Errorf("部署模板不存在: %v", err)
		}
	}

	// 检查环境和集群组合的唯一性
	isUnique, err := s.deployRepo.CheckUniqueDeployment(req.ApplicationID, req.EnvironmentID, nil)
	if err != nil {
		return nil, fmt.Errorf("检查唯一性失败: %v", err)
	}
	if !isUnique {
		return nil, fmt.Errorf("该环境和集群的组合已存在")
	}

	deployment := &model.ApplicationDeployment{
		ApplicationDeploymentRequest: *req,
	}

	if err := s.deployRepo.Create(deployment); err != nil {
		return nil, err
	}

	// 获取完整的部署配置信息
	fullDeployment, err := s.deployRepo.GetByID(deployment.ID)
	if err != nil {
		return nil, err
	}

	response := &model.ApplicationDeployment{
		ID:                           fullDeployment.ID,
		ApplicationDeploymentRequest: *req,
		CreatedAt:                    fullDeployment.CreatedAt,
		UpdatedAt:                    fullDeployment.UpdatedAt,
		Application:                  fullDeployment.Application,
		Environment:                  fullDeployment.Environment,
		BuildTemplate:                fullDeployment.BuildTemplate,
		DeploymentTemplate:           fullDeployment.DeploymentTemplate,
	}

	return response, nil
}

// GetApplicationDeploymentByID 根据ID获取应用部署配置
func (s *ApplicationService) GetApplicationDeploymentByID(id uint) (*model.ApplicationDeployment, error) {
	return s.deployRepo.GetByID(id)
}

// ListDeploymentsByApplicationID 根据应用服务ID获取部署配置列表
func (s *ApplicationService) ListDeploymentsByApplicationID(appID uint) ([]model.ApplicationDeployment, error) {
	return s.deployRepo.ListByApplicationID(appID)
}

// ListDeploymentsByEnvironmentID 根据环境ID获取部署配置列表
func (s *ApplicationService) ListDeploymentsByEnvironmentID(envID uint) ([]model.ApplicationDeployment, error) {
	return s.deployRepo.ListByEnvironmentID(envID)
}

// UpdateApplicationDeployment 更新应用部署配置
func (s *ApplicationService) UpdateApplicationDeployment(id uint, req *model.ApplicationDeploymentRequest) (*model.ApplicationDeployment, error) {
	// 获取现有部署配置
	existingDeployment, err := s.deployRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("部署配置不存在: %v", err)
	}

	// 验证环境是否存在
	if req.EnvironmentID != 0 {
		var env model.Environment
		if err := s.db.First(&env, req.EnvironmentID).Error; err != nil {
			return nil, fmt.Errorf("环境不存在: %v", err)
		}
	}

	// 验证目标平台是否存在
	//if req.Platform != 0 {
	//	var cluster model.K8SCluster
	//	if err := s.db.First(&cluster, req.Platform).Error; err != nil {
	//		return nil, fmt.Errorf("K8s集群不存在: %v", err)
	//	}
	//}

	// 验证构建模板（如果提供）
	if req.BuildTemplateID != nil {
		var buildTemplate model.BuildTemplate
		if err := s.db.First(&buildTemplate, *req.BuildTemplateID).Error; err != nil {
			return nil, fmt.Errorf("构建模板不存在: %v", err)
		}
	}

	// 验证部署模板（如果提供）
	if req.DeploymentTemplateID != nil {
		var deployTemplate model.DeploymentTemplate
		if err := s.db.First(&deployTemplate, *req.DeploymentTemplateID).Error; err != nil {
			return nil, fmt.Errorf("部署模板不存在: %v", err)
		}
	}

	// 检查环境和集群组合的唯一性（排除自己）
	appID := existingDeployment.ApplicationID
	envID := req.EnvironmentID
	if envID == 0 {
		envID = existingDeployment.EnvironmentID
	}

	isUnique, err := s.deployRepo.CheckUniqueDeployment(appID, envID, &id)
	if err != nil {
		return nil, fmt.Errorf("检查唯一性失败: %v", err)
	}
	if !isUnique {
		return nil, fmt.Errorf("该环境和集群的组合已存在")
	}

	// 更新字段
	if req.EnvironmentID != 0 {
		existingDeployment.EnvironmentID = req.EnvironmentID
	}

	existingDeployment.Description = req.Description
	existingDeployment.BuildTemplateID = req.BuildTemplateID
	existingDeployment.DeploymentTemplateID = req.DeploymentTemplateID

	if err := s.deployRepo.Update(existingDeployment); err != nil {
		return nil, err
	}

	// 获取完整的部署配置信息
	fullDeployment, err := s.deployRepo.GetByID(existingDeployment.ID)
	if err != nil {
		return nil, err
	}

	response := &model.ApplicationDeployment{
		ID:                           fullDeployment.ID,
		ApplicationDeploymentRequest: *req,
		CreatedAt:                    fullDeployment.CreatedAt,
		UpdatedAt:                    fullDeployment.UpdatedAt,
		Application:                  fullDeployment.Application,
		Environment:                  fullDeployment.Environment,
		BuildTemplate:                fullDeployment.BuildTemplate,
		DeploymentTemplate:           fullDeployment.DeploymentTemplate,
	}

	return response, nil
}

// DeleteApplicationDeployment 删除应用部署配置
func (s *ApplicationService) DeleteApplicationDeployment(id uint) error {
	return s.deployRepo.Delete(id)
}
