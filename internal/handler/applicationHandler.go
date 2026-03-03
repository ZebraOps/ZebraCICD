package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"gorm.io/gorm"
)

type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// Create 创建应用服务（返回持久化后的实体，包含ID和时间戳）
func (r *ApplicationRepository) Create(application *model.Application) error {
	return r.db.Create(application).Error
}

// GetByID 根据ID获取应用服务
func (r *ApplicationRepository) GetByID(id uint) (*model.Application, error) {
	var application model.Application
	if err := r.db.Preload("Repo").Preload("Deployments").First(&application, id).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

// ListByRepoID 根据仓库ID获取应用服务列表
func (r *ApplicationRepository) ListByRepoID(repoID uint) ([]model.Application, error) {
	var applications []model.Application
	if err := r.db.Where("repo_id = ?", repoID).Preload("Deployments").Find(&applications).Error; err != nil {
		return nil, err
	}
	return applications, nil
}

// Update 更新应用服务
func (r *ApplicationRepository) Update(application *model.Application) error {
	return r.db.Save(application).Error
}

// Delete 删除应用服务
func (r *ApplicationRepository) Delete(id uint) error {
	return r.db.Delete(&model.Application{}, id).Error
}

// GetApplicationsWithDeploymentCount 获取应用服务列表并包含部署配置数量
func (r *ApplicationRepository) GetApplicationsWithDeploymentCount(repoID uint) ([]model.ApplicationResponse, error) {
	var applications []model.Application
	if err := r.db.Where("repo_id = ?", repoID).Find(&applications).Error; err != nil {
		return nil, err
	}

	var responses []model.ApplicationResponse
	for _, app := range applications {
		var count int64
		r.db.Model(&model.ApplicationDeployment{}).Where("application_id = ?", app.ID).Count(&count)

		response := model.ApplicationResponse{
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
			DeploymentCount: count,
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// ApplicationDeploymentRepository 应用部署配置Repository
type ApplicationDeploymentRepository struct {
	db *gorm.DB
}

func NewApplicationDeploymentRepository(db *gorm.DB) *ApplicationDeploymentRepository {
	return &ApplicationDeploymentRepository{db: db}
}

// Create 创建应用部署配置
func (r *ApplicationDeploymentRepository) Create(deployment *model.ApplicationDeployment) error {
	return r.db.Create(deployment).Error
}

// GetByID 根据ID获取应用部署配置
func (r *ApplicationDeploymentRepository) GetByID(id uint) (*model.ApplicationDeployment, error) {
	var deployment model.ApplicationDeployment
	if err := r.db.Preload("Application").Preload("Environment").Preload("K8sCluster").
		Preload("BuildTemplate").Preload("DeploymentTemplate").First(&deployment, id).Error; err != nil {
		return nil, err
	}
	return &deployment, nil
}

// ListByApplicationID 根据应用服务ID获取部署配置列表
func (r *ApplicationDeploymentRepository) ListByApplicationID(appID uint) ([]model.ApplicationDeployment, error) {
	var deployments []model.ApplicationDeployment
	if err := r.db.Where("application_id = ?", appID).
		Preload("Environment").
		Preload("BuildTemplate").Preload("DeploymentTemplate").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

// ListByEnvironmentID 根据环境ID获取部署配置列表
func (r *ApplicationDeploymentRepository) ListByEnvironmentID(envID uint) ([]model.ApplicationDeployment, error) {
	var deployments []model.ApplicationDeployment
	if err := r.db.Where("environment_id = ?", envID).
		Preload("Application").Preload("Environment").
		Preload("K8sCluster").Preload("BuildTemplate").
		Preload("DeploymentTemplate").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

// Update 更新应用部署配置
func (r *ApplicationDeploymentRepository) Update(deployment *model.ApplicationDeployment) error {
	return r.db.Save(deployment).Error
}

// Delete 删除应用部署配置
func (r *ApplicationDeploymentRepository) Delete(id uint) error {
	return r.db.Delete(&model.ApplicationDeployment{}, id).Error
}

// CheckUniqueDeployment 检查环境和集群组合的唯一性
func (r *ApplicationDeploymentRepository) CheckUniqueDeployment(appID, envID uint, excludeID *uint) (bool, error) {
	query := r.db.Model(&model.ApplicationDeployment{}).
		Where("application_id = ? AND environment_id = ?", appID, envID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}
