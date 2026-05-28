package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateCloudProviderHandler 创建云厂商
// @Summary 创建云厂商
// @Description 创建一个新的云厂商
// @Tags cloud-providers
// @Accept json
// @Produce json
// @Param provider body model.CloudProvider true "云厂商信息"
// @Success 201 {object} model.CloudProvider
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vendors [post]
func CreateCloudProviderHandler(c *gin.Context, svc *service.CloudProviderService) {
	var req model.CloudProvider
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateCloudProvider(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListCloudProvidersHandler 获取云厂商列表
// @Summary 获取云厂商列表
// @Description 获取所有云厂商的列表，支持按名称、提供商、状态等条件查询
// @Tags cloud-providers
// @Produce json
// @Param name query string false "云厂商名称"
// @Param provider query string false "提供商标识"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.CloudProvider}}
// @Failure 500 {object} map[string]string
// @Router /api/vendors [get]
func ListCloudProvidersHandler(c *gin.Context, svc *service.CloudProviderService) {
	// 解析查询参数
	name := c.Query("name")
	provider := c.Query("provider")
	status := c.Query("status")

	// 解析分页参数
	page := 1
	size := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := c.Query("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			size = s
		}
	}

	// 构建查询条件
	conditions := types.CloudProviderQueryConditions{
		Name:     name,
		Provider: provider,
		Status:   status,
	}

	// 调用服务层获取分页数据
	providers, total, err := svc.ListCloudProvidersWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, providers)
}

// GetCloudProviderHandler 根据ID获取云厂商
// @Summary 根据ID获取云厂商
// @Description 根据云厂商ID获取云厂商详情
// @Tags cloud-providers
// @Produce json
// @Param id path int true "云厂商ID"
// @Success 200 {object} model.CloudProvider
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vendors/{id} [get]
func GetCloudProviderHandler(c *gin.Context, svc *service.CloudProviderService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	provider, err := svc.GetCloudProviderByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "cloud provider not found")
		return
	}
	types.Success(c, provider)
}

// UpdateCloudProviderHandler 更新云厂商
// @Summary 更新云厂商
// @Description 根据云厂商ID更新云厂商信息
// @Tags cloud-providers
// @Accept json
// @Produce json
// @Param id path int true "云厂商ID"
// @Param provider body model.CloudProvider true "云厂商信息"
// @Success 200 {object} model.CloudProvider
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vendors/{id} [put]
func UpdateCloudProviderHandler(c *gin.Context, svc *service.CloudProviderService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查云厂商是否存在
	existingProvider, err := svc.GetCloudProviderByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "cloud provider not found")
		return
	}

	var req model.CloudProvider
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段
	if req.Name != "" {
		existingProvider.Name = req.Name
	}
	if req.DisplayName != "" {
		existingProvider.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		existingProvider.Description = req.Description
	}
	if req.Region != "" {
		existingProvider.Region = req.Region
	}
	if req.AccessKey != "" {
		existingProvider.AccessKey = req.AccessKey
	}
	if req.SecretKey != "" {
		existingProvider.SecretKey = req.SecretKey
	}
	if req.Endpoint != "" {
		existingProvider.Endpoint = req.Endpoint
	}
	if req.Config != "" {
		existingProvider.Config = req.Config
	}
	if req.Status != "" {
		existingProvider.Status = req.Status
	}

	if err := svc.UpdateCloudProvider(existingProvider); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingProvider)
}

// DeleteCloudProviderHandler 删除云厂商
// @Summary 删除云厂商
// @Description 根据云厂商ID删除云厂商（软删除）
// @Tags cloud-providers
// @Produce json
// @Param id path int true "云厂商ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/vendors/{id} [delete]
func DeleteCloudProviderHandler(c *gin.Context, svc *service.CloudProviderService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteCloudProvider(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "cloud provider deleted successfully"})
}

// RegisterCloudProviderRoutes 注册云厂商相关路由
func RegisterCloudProviderRoutes(r *gin.Engine, svc *service.CloudProviderService) {
	g := r.Group("/api/vendors")
	{
		// 创建云厂商
		g.POST("", func(c *gin.Context) {
			CreateCloudProviderHandler(c, svc)
		})

		// 获取云厂商列表
		g.GET("", func(c *gin.Context) {
			ListCloudProvidersHandler(c, svc)
		})

		// 根据ID获取云厂商
		g.GET("/:id", func(c *gin.Context) {
			GetCloudProviderHandler(c, svc)
		})

		// 更新云厂商
		g.PUT("/:id", func(c *gin.Context) {
			UpdateCloudProviderHandler(c, svc)
		})

		// 删除云厂商
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteCloudProviderHandler(c, svc)
		})
	}
}
