package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateGitPlatformHandler 创建Git平台配置
func CreateGitPlatformHandler(c *gin.Context, svc *service.GitPlatformService) {
	var req model.GitPlatform
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateGitPlatform(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListGitPlatformsHandler 获取Git平台配置列表
func ListGitPlatformsHandler(c *gin.Context, svc *service.GitPlatformService) {
	name := c.Query("name")
	platformType := c.Query("platform_type")
	authType := c.Query("auth_type")
	status := c.Query("status")

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

	conditions := types.GitPlatformQueryConditions{
		Name:          name,
		PlatformType:  platformType,
		AuthType:      authType,
		Status:        status,
	}

	platforms, total, err := svc.ListGitPlatformsWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, platforms)
}

// GetGitPlatformHandler 根据ID获取Git平台配置
func GetGitPlatformHandler(c *gin.Context, svc *service.GitPlatformService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	platform, err := svc.GetGitPlatformByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "git platform not found")
		return
	}
	types.Success(c, platform)
}

// UpdateGitPlatformHandler 更新Git平台配置
func UpdateGitPlatformHandler(c *gin.Context, svc *service.GitPlatformService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	existing, err := svc.GetGitPlatformByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "git platform not found")
		return
	}

	var req model.GitPlatform
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.PlatformType != "" {
		existing.PlatformType = req.PlatformType
	}
	if req.URL != "" {
		existing.URL = req.URL
	}
	if req.APIUrl != "" {
		existing.APIUrl = req.APIUrl
	}
	if req.AuthType != "" {
		existing.AuthType = req.AuthType
	}
	if req.AuthConfig != "" {
		existing.AuthConfig = req.AuthConfig
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := svc.UpdateGitPlatform(existing); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existing)
}

// DeleteGitPlatformHandler 删除Git平台配置
func DeleteGitPlatformHandler(c *gin.Context, svc *service.GitPlatformService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteGitPlatform(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "git platform deleted successfully"})
}

// ConnectGitPlatformHandler 测试Git平台连通性
func ConnectGitPlatformHandler(c *gin.Context, svc *service.GitPlatformService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.TestGitPlatformConnection(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "connection successful"})
}

// ListGitPlatformProjectsHandler 获取指定Git平台的项目列表
func ListGitPlatformProjectsHandler(c *gin.Context, svc *service.GitPlatformService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	search := c.Query("search")
	page := 1
	size := 10
	if p, e := strconv.Atoi(c.Query("page")); e == nil && p > 0 {
		page = p
	}
	if s, e := strconv.Atoi(c.Query("size")); e == nil && s > 0 {
		size = s
	}
	projects, err := svc.ListPlatformProjects(uint(id), search, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, projects)
}

func RegisterGitPlatformRoutes(r *gin.Engine, svc *service.GitPlatformService) {
	g := r.Group("/api/git-platforms")
	{
		g.POST("", func(c *gin.Context) {
			CreateGitPlatformHandler(c, svc)
		})
		g.GET("", func(c *gin.Context) {
			ListGitPlatformsHandler(c, svc)
		})
		g.GET("/:id", func(c *gin.Context) {
			GetGitPlatformHandler(c, svc)
		})
		g.PUT("/:id", func(c *gin.Context) {
			UpdateGitPlatformHandler(c, svc)
		})
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteGitPlatformHandler(c, svc)
		})
		g.POST("/:id/connect", func(c *gin.Context) {
			ConnectGitPlatformHandler(c, svc)
		})
		g.GET("/:id/projects", func(c *gin.Context) {
			ListGitPlatformProjectsHandler(c, svc)
		})
	}
}