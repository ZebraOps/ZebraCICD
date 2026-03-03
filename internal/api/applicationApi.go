package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateApplicationHandler 创建应用服务
// @Summary 创建应用服务
// @Description 基于仓库创建应用服务
// @Tags applications
// @Accept json
// @Produce json
// @Param application body model.ApplicationRequest true "应用服务信息"
// @Success 201 {object} model.ApplicationResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/applications [post]
func CreateApplicationHandler(c *gin.Context, svc *service.ApplicationService) {
	var req model.ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	application, err := svc.CreateApplication(&req)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, application)
}

// ListApplicationsHandler 获取应用服务列表
// @Summary 获取应用服务列表
// @Description 根据仓库ID获取应用服务列表
// @Tags applications
// @Produce json
// @Param id query int true "仓库ID"
// @Success 200 {array} model.ApplicationResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/applications [get]
func ListApplicationsHandler(c *gin.Context, svc *service.ApplicationService) {
	repoIDStr := c.Query("id")
	if repoIDStr == "" {
		types.Error(c, http.StatusBadRequest, "repo_id is required")
		return
	}

	repoID, err := strconv.Atoi(repoIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid repo_id format")
		return
	}

	applications, err := svc.ListApplicationsByRepoID(uint(repoID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, applications)
}

// GetApplicationByIDHandler 根据ID获取应用服务
// @Summary 根据ID获取应用服务
// @Description 根据应用服务ID获取详细信息
// @Tags applications
// @Produce json
// @Param id path int true "应用服务ID"
// @Success 200 {object} model.Application
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/applications/{id} [get]
func GetApplicationByIDHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	application, err := svc.GetApplicationByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "application not found")
		return
	}

	types.Success(c, application)
}

// UpdateApplicationHandler 更新应用服务
// @Summary 更新应用服务
// @Description 根据ID更新应用服务信息
// @Tags applications
// @Accept json
// @Produce json
// @Param id path int true "应用服务ID"
// @Param application body model.ApplicationRequest true "应用服务信息"
// @Success 200 {object} model.Application
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/applications/{id} [put]
func UpdateApplicationHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	var req model.ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	application, err := svc.UpdateApplication(uint(id), &req)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, application)
}

// DeleteApplicationHandler 删除应用服务
// @Summary 删除应用服务
// @Description 根据ID删除应用服务及其相关部署配置
// @Tags applications
// @Produce json
// @Param id path int true "应用服务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/applications/{id} [delete]
func DeleteApplicationHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteApplication(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "application deleted successfully"})
}

// CreateApplicationDeploymentHandler 创建应用部署配置
// @Summary 创建应用部署配置
// @Description 为应用服务创建部署配置
// @Tags application-deployments
// @Accept json
// @Produce json
// @Param deployment body model.ApplicationDeploymentRequest true "部署配置信息"
// @Success 201 {object} model.ApplicationDeploymentResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template [post]
func CreateApplicationDeploymentHandler(c *gin.Context, svc *service.ApplicationService) {
	var req model.ApplicationDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	deployment, err := svc.CreateApplicationDeployment(&req)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, deployment)
}

// ListApplicationDeploymentsHandler 获取应用部署配置列表
// @Summary 获取应用部署配置列表
// @Description 根据应用服务ID获取部署配置列表
// @Tags application-deployments
// @Produce json
// @Param application_id query int true "应用服务ID"
// @Success 200 {array} model.ApplicationDeployment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template [get]
func ListApplicationDeploymentsHandler(c *gin.Context, svc *service.ApplicationService) {
	appIDStr := c.Query("application_id")
	if appIDStr == "" {
		types.Error(c, http.StatusBadRequest, "application_id is required")
		return
	}

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid application_id format")
		return
	}

	deployments, err := svc.ListDeploymentsByApplicationID(uint(appID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, deployments)
}

// GetApplicationDeploymentByIDHandler 根据ID获取应用部署配置
// @Summary 根据ID获取应用部署配置
// @Description 根据部署配置ID获取详细信息
// @Tags application-deployments
// @Produce json
// @Param id path int true "部署配置ID"
// @Success 200 {object} model.ApplicationDeploymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template/{id} [get]
func GetApplicationDeploymentByIDHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	deployment, err := svc.GetApplicationDeploymentByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "deployment not found")
		return
	}

	types.Success(c, deployment)
}

// UpdateApplicationDeploymentHandler 更新应用部署配置
// @Summary 更新应用部署配置
// @Description 根据ID更新部署配置信息
// @Tags application-deployments
// @Accept json
// @Produce json
// @Param id path int true "部署配置ID"
// @Param deployment body model.ApplicationDeploymentRequest true "部署配置信息"
// @Success 200 {object} model.ApplicationDeploymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template/{id} [put]
func UpdateApplicationDeploymentHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	var req model.ApplicationDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	deployment, err := svc.UpdateApplicationDeployment(uint(id), &req)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, deployment)
}

// DeleteApplicationDeploymentHandler 删除应用部署配置
// @Summary 删除应用部署配置
// @Description 根据ID删除部署配置
// @Tags application-deployments
// @Produce json
// @Param id path int true "部署配置ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template/{id} [delete]
func DeleteApplicationDeploymentHandler(c *gin.Context, svc *service.ApplicationService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteApplicationDeployment(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "deployment deleted successfully"})
}

// ListDeploymentsByEnvironmentHandler 根据环境获取部署配置列表
// @Summary 根据环境获取部署配置列表
// @Description 根据环境ID获取该环境下所有的部署配置
// @Tags application-deployments
// @Produce json
// @Param environment_id query int true "环境ID"
// @Success 200 {array} model.ApplicationDeployment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/application/template/environment [get]
func ListDeploymentsByEnvironmentHandler(c *gin.Context, svc *service.ApplicationService) {
	envIDStr := c.Query("environment_id")
	if envIDStr == "" {
		types.Error(c, http.StatusBadRequest, "environment_id is required")
		return
	}

	envID, err := strconv.Atoi(envIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid environment_id format")
		return
	}

	deployments, err := svc.ListDeploymentsByEnvironmentID(uint(envID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, deployments)
}

// RegisterApplicationRoutes 注册应用服务相关路由
func RegisterApplicationRoutes(r *gin.Engine, svc *service.ApplicationService) {
	appGroup := r.Group("/api/applications")
	{
		// 应用服务相关路由
		appGroup.POST("", func(c *gin.Context) {
			CreateApplicationHandler(c, svc)
		})
		appGroup.GET("", func(c *gin.Context) {
			ListApplicationsHandler(c, svc)
		})
		appGroup.GET("/:id", func(c *gin.Context) {
			GetApplicationByIDHandler(c, svc)
		})
		appGroup.PUT("/:id", func(c *gin.Context) {
			UpdateApplicationHandler(c, svc)
		})
		appGroup.DELETE("/:id", func(c *gin.Context) {
			DeleteApplicationHandler(c, svc)
		})
	}

	deployGroup := r.Group("/api/application/template")
	{
		// 应用部署配置相关路由
		deployGroup.POST("", func(c *gin.Context) {
			CreateApplicationDeploymentHandler(c, svc)
		})
		deployGroup.GET("", func(c *gin.Context) {
			ListApplicationDeploymentsHandler(c, svc)
		})
		deployGroup.GET("/environment", func(c *gin.Context) {
			ListDeploymentsByEnvironmentHandler(c, svc)
		})
		deployGroup.GET("/:id", func(c *gin.Context) {
			GetApplicationDeploymentByIDHandler(c, svc)
		})
		deployGroup.PUT("/:id", func(c *gin.Context) {
			UpdateApplicationDeploymentHandler(c, svc)
		})
		deployGroup.DELETE("/:id", func(c *gin.Context) {
			DeleteApplicationDeploymentHandler(c, svc)
		})
	}
}
