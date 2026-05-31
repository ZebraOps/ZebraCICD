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

// GetApplicationsWithDeploymentCount 获取应用服务列表并包含部署配置数量，支持按部门/语言过滤，支持分页
func (r *ApplicationRepository) GetApplicationsWithDeploymentCount(repoID uint, department, language string, page, size int) ([]model.ApplicationResponse, int64, error) {
	baseQuery := r.db.Table("applications").
		Select("applications.*, repos.repo_department AS department, repos.repo_language AS language").
		Joins("LEFT JOIN repos ON repos.id = applications.repo_id")

	if repoID > 0 {
		baseQuery = baseQuery.Where("applications.repo_id = ?", repoID)
	}
	if department != "" {
		baseQuery = baseQuery.Where("repos.repo_department LIKE ?", "%"+department+"%")
	}
	if language != "" {
		baseQuery = baseQuery.Where("repos.repo_language LIKE ?", "%"+language+"%")
	}

	// 先统计总数
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size

	type row struct {
		model.Application
		Department string
		Language   string
	}
	var rows []row
	if err := baseQuery.Order("applications.id DESC").Offset(offset).Limit(size).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	var responses []model.ApplicationResponse
	for _, rr := range rows {
		var count int64
		r.db.Model(&model.ApplicationDeployment{}).Where("application_id = ?", rr.ID).Count(&count)

		resp := model.ApplicationResponse{
			ID:              rr.ID,
			RepoID:          rr.RepoID,
			CName:           rr.CName,
			EName:           rr.EName,
			ListenPort:      rr.ListenPort,
			HealthCheckType: rr.HealthCheckType,
			HealthCheckURL:  rr.HealthCheckURL,
			Description:     rr.Description,
			Department:      rr.Department,
			Language:        rr.Language,
			CreatedAt:       rr.CreatedAt,
			UpdatedAt:       rr.UpdatedAt,
			DeploymentCount: count,
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
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
		Preload("Server").
		Preload("BuildTemplate").Preload("DeploymentTemplate").First(&deployment, id).Error; err != nil {
		return nil, err
	}
	return &deployment, nil
}

// ListByApplicationID 根据应用服务ID获取部署配置列表
func (r *ApplicationDeploymentRepository) ListByApplicationID(appID uint) ([]model.ApplicationDeployment, error) {
	var deployments []model.ApplicationDeployment
	if err := r.db.Where("application_id = ?", appID).
		Preload("Environment").Preload("K8sCluster").Preload("Server").
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
		Preload("K8sCluster").Preload("Server").Preload("BuildTemplate").
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

// CheckUniqueDeployment 检查(应用,环境,部署目标)组合的唯一性
func (r *ApplicationDeploymentRepository) CheckUniqueDeployment(appID, envID uint, deployTarget string, excludeID *uint) (bool, error) {
	query := r.db.Model(&model.ApplicationDeployment{}).
		Where("application_id = ? AND environment_id = ? AND deploy_target = ?", appID, envID, deployTarget)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

// ListByAppAndEnv 根据应用ID和环境ID获取部署配置列表（用于任务创建自动填充）
func (r *ApplicationDeploymentRepository) ListByAppAndEnv(appID, envID uint) ([]model.ApplicationDeployment, error) {
	var deployments []model.ApplicationDeployment
	if err := r.db.Where("application_id = ? AND environment_id = ?", appID, envID).
		Preload("Environment").Preload("K8sCluster").Preload("Server").
		Preload("BuildTemplate").Preload("DeploymentTemplate").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}
