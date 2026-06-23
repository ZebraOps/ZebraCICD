package service

import (
	"fmt"
	"strings"

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

// ListApplicationsByRepoID 根据仓库ID及可选部门/语言分页获取应用服务列表
func (s *ApplicationService) ListApplicationsByRepoID(repoID uint, department, language string, page, size int) ([]model.ApplicationResponse, int64, error) {
	return s.appRepo.GetApplicationsWithDeploymentCount(repoID, department, language, page, size)
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

// validateDeployTarget 校验部署目标及其条件性字段
func (s *ApplicationService) validateDeployTarget(req *model.ApplicationDeploymentRequest) error {
	switch req.DeployTarget {
	case "k8s":
		if req.K8sClusterID == nil || *req.K8sClusterID == 0 {
			return fmt.Errorf("K8s部署目标必须指定集群ID")
		}
		var cluster model.K8SCluster
		if err := s.db.First(&cluster, *req.K8sClusterID).Error; err != nil {
			return fmt.Errorf("K8s集群不存在: %v", err)
		}
	case "docker":
		if req.ServerID == nil || *req.ServerID == 0 {
			return fmt.Errorf("Docker部署目标必须指定服务器ID")
		}
		var server model.Server
		if err := s.db.First(&server, *req.ServerID).Error; err != nil {
			return fmt.Errorf("目标服务器不存在: %v", err)
		}
	case "linux":
		if req.ServerID == nil || *req.ServerID == 0 {
			return fmt.Errorf("Linux部署目标必须指定服务器ID")
		}
		var server model.Server
		if err := s.db.First(&server, *req.ServerID).Error; err != nil {
			return fmt.Errorf("目标服务器不存在: %v", err)
		}
		if req.DeployPath == "" {
			return fmt.Errorf("Linux部署目标必须指定部署路径(Nginx代理目录)")
		}
	default:
		return fmt.Errorf("部署目标必须是 k8s、docker 或 linux")
	}
	return nil
}

func (s *ApplicationService) validateCredentialMode(req *model.ApplicationDeploymentRequest) error {
	mode := strings.TrimSpace(req.CredentialMode)
	if mode == "" {
		mode = "auto_create"
	}
	if mode != "auto_create" && mode != "manual_select" {
		return fmt.Errorf("credential_mode 仅支持 auto_create 或 manual_select")
	}
	req.CredentialMode = mode

	if req.JenkinsPlatformID != nil && *req.JenkinsPlatformID > 0 {
		var platform model.JenkinsPlatform
		if err := s.db.First(&platform, *req.JenkinsPlatformID).Error; err != nil {
			return fmt.Errorf("Jenkins平台不存在: %v", err)
		}
	}

	if mode == "manual_select" {
		if req.JenkinsPlatformID == nil || *req.JenkinsPlatformID == 0 {
			return fmt.Errorf("手动选择模式必须先选择 Jenkins 平台")
		}
		if req.JenkinsCredentialID == nil || *req.JenkinsCredentialID == 0 {
			return fmt.Errorf("手动选择模式必须指定 Jenkins 凭据")
		}

		var cred model.JenkinsCredential
		if err := s.db.First(&cred, *req.JenkinsCredentialID).Error; err != nil {
			return fmt.Errorf("Jenkins凭据不存在: %v", err)
		}
		if cred.JenkinsPlatformID != *req.JenkinsPlatformID {
			return fmt.Errorf("所选 Jenkins 凭据不属于当前平台")
		}
		if cred.Status != "active" {
			return fmt.Errorf("所选 Jenkins 凭据不可用，当前状态: %s", cred.Status)
		}
		return nil
	}

	// 自动创建模式不依赖手动凭据
	req.JenkinsCredentialID = nil
	return nil
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

	// 校验部署目标及其条件性字段
	if err := s.validateDeployTarget(req); err != nil {
		return nil, err
	}

	if err := s.validateCredentialMode(req); err != nil {
		return nil, err
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

	// 检查(应用,环境,部署目标)组合的唯一性
	isUnique, err := s.deployRepo.CheckUniqueDeployment(req.ApplicationID, req.EnvironmentID, req.DeployTarget, nil)
	if err != nil {
		return nil, fmt.Errorf("检查唯一性失败: %v", err)
	}
	if !isUnique {
		return nil, fmt.Errorf("该应用在相同环境和部署目标下已存在配置")
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
		ApplicationDeploymentRequest: fullDeployment.ApplicationDeploymentRequest,
		CreatedAt:                    fullDeployment.CreatedAt,
		UpdatedAt:                    fullDeployment.UpdatedAt,
		Application:                  fullDeployment.Application,
		Environment:                  fullDeployment.Environment,
		BuildTemplate:                fullDeployment.BuildTemplate,
		DeploymentTemplate:           fullDeployment.DeploymentTemplate,
		K8sCluster:                   fullDeployment.K8sCluster,
		Server:                       fullDeployment.Server,
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

// ListDeploymentsByAppAndEnv 根据应用ID和环境ID获取部署配置列表（用于任务创建自动填充）
func (s *ApplicationService) ListDeploymentsByAppAndEnv(appID, envID uint) ([]model.ApplicationDeployment, error) {
	return s.deployRepo.ListByAppAndEnv(appID, envID)
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

	// 校验部署目标及其条件性字段
	if err := s.validateDeployTarget(req); err != nil {
		return nil, err
	}

	if err := s.validateCredentialMode(req); err != nil {
		return nil, err
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

	// 检查(应用,环境,部署目标)组合的唯一性（排除自己）
	appID := existingDeployment.ApplicationID
	envID := req.EnvironmentID
	if envID == 0 {
		envID = existingDeployment.EnvironmentID
	}
	deployTarget := req.DeployTarget
	if deployTarget == "" {
		deployTarget = existingDeployment.DeployTarget
	}

	isUnique, err := s.deployRepo.CheckUniqueDeployment(appID, envID, deployTarget, &id)
	if err != nil {
		return nil, fmt.Errorf("检查唯一性失败: %v", err)
	}
	if !isUnique {
		return nil, fmt.Errorf("该应用在相同环境和部署目标下已存在配置")
	}

	// 更新字段
	if req.EnvironmentID != 0 {
		existingDeployment.EnvironmentID = req.EnvironmentID
	}
	existingDeployment.DeployTarget = req.DeployTarget
	existingDeployment.BuildSource = req.BuildSource
	existingDeployment.Description = req.Description
	existingDeployment.BuildTemplateID = req.BuildTemplateID
	existingDeployment.DeploymentTemplateID = req.DeploymentTemplateID
	existingDeployment.K8sClusterID = req.K8sClusterID
	existingDeployment.K8sNamespace = req.K8sNamespace
	existingDeployment.ServerID = req.ServerID
	existingDeployment.DeployPath = req.DeployPath
	existingDeployment.JenkinsPlatformID = req.JenkinsPlatformID
	existingDeployment.JenkinsCredentialID = req.JenkinsCredentialID
	existingDeployment.GitPlatformID = req.GitPlatformID
	existingDeployment.ImageRepoID = req.ImageRepoID
	existingDeployment.CredentialMode = req.CredentialMode

	// 清空预加载的关联对象，避免 GORM Save 时被关联对象的主键覆盖外键字段
	existingDeployment.Application = nil
	existingDeployment.Environment = nil
	existingDeployment.BuildTemplate = nil
	existingDeployment.DeploymentTemplate = nil
	existingDeployment.K8sCluster = nil
	existingDeployment.Server = nil
	existingDeployment.JenkinsPlatform = nil
	existingDeployment.JenkinsCredential = nil
	existingDeployment.GitPlatform = nil
	existingDeployment.ImageRepository = nil

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
		ApplicationDeploymentRequest: fullDeployment.ApplicationDeploymentRequest,
		CreatedAt:                    fullDeployment.CreatedAt,
		UpdatedAt:                    fullDeployment.UpdatedAt,
		Application:                  fullDeployment.Application,
		Environment:                  fullDeployment.Environment,
		BuildTemplate:                fullDeployment.BuildTemplate,
		DeploymentTemplate:           fullDeployment.DeploymentTemplate,
		K8sCluster:                   fullDeployment.K8sCluster,
		Server:                       fullDeployment.Server,
	}

	return response, nil
}

// DeleteApplicationDeployment 删除应用部署配置
func (s *ApplicationService) DeleteApplicationDeployment(id uint) error {
	return s.deployRepo.Delete(id)
}
