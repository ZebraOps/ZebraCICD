package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateImageRepositoryHandler 创建镜像仓库
// @Summary 创建镜像仓库
// @Description 创建一个新的镜像仓库
// @Tags image-repositories
// @Accept json
// @Produce json
// @Param repository body model.ImageRepository true "镜像仓库信息"
// @Success 201 {object} model.ImageRepository
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/image-registries [post]
func CreateImageRepositoryHandler(c *gin.Context, svc *service.ImageRepositoryService) {
	var req model.ImageRepository
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateRepository(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// GetImageRepositoryHandler 根据ID获取镜像仓库
// @Summary 根据ID获取镜像仓库
// @Description 根据镜像仓库ID获取镜像仓库详情
// @Tags image-repositories
// @Produce json
// @Param id path int true "镜像仓库ID"
// @Success 200 {object} model.ImageRepository
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/image-registries/{id} [get]
func GetImageRepositoryHandler(c *gin.Context, svc *service.ImageRepositoryService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	repository, err := svc.GetRepositoryByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "repository not found")
		return
	}
	types.Success(c, repository)
}

// ListImageRepositoriesHandler 获取镜像仓库列表处理函数
// @Summary 获取镜像仓库列表
// @Description 获取所有镜像仓库的列表，支持按名称、地址过滤，支持分页
// @Tags image-repositories
// @Produce json
// @Param name query string false "仓库名称"
// @Param url query string false "仓库地址"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.ImageRepository}}
// @Failure 500 {object} map[string]string
// @Router /api/image-registries [get]
func ListImageRepositoriesHandler(c *gin.Context, svc *service.ImageRepositoryService) {
	// 解析查询参数
	name := c.Query("name")
	url := c.Query("url")

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
	conditions := types.ImageRepositoryQueryConditions{
		Name: name,
		URL:  url,
		Page: page,
		Size: size,
	}

	// 调用服务层获取数据
	repositories, total, err := svc.ListRepositories(conditions)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	types.PageSuccess(c, total, repositories)
}

// UpdateImageRepositoryHandler 更新镜像仓库
// @Summary 更新镜像仓库
// @Description 根据镜像仓库ID更新镜像仓库信息
// @Tags image-repositories
// @Accept json
// @Produce json
// @Param id path int true "镜像仓库ID"
// @Param repository body model.ImageRepository true "镜像仓库信息"
// @Success 200 {object} model.ImageRepository
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/image-registries/{id} [put]
func UpdateImageRepositoryHandler(c *gin.Context, svc *service.ImageRepositoryService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	var req model.ImageRepository
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = uint(id)
	if err := svc.UpdateRepository(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// DeleteImageRepositoryHandler 删除镜像仓库
// @Summary 删除镜像仓库
// @Description 根据镜像仓库ID删除镜像仓库（软删除）
// @Tags image-repositories
// @Produce json
// @Param id path int true "镜像仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/image-registries/{id} [delete]
func DeleteImageRepositoryHandler(c *gin.Context, svc *service.ImageRepositoryService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	if err := svc.DeleteRepository(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "repository deleted successfully"})
}

// RegisterImageRepositoryRoutes 注册镜像仓库相关路由
func RegisterImageRepositoryRoutes(r *gin.Engine, svc *service.ImageRepositoryService) {
	g := r.Group("/api/image-registries")
	{
		// 创建镜像仓库
		g.POST("", func(c *gin.Context) {
			CreateImageRepositoryHandler(c, svc)
		})

		// 获取镜像仓库列表
		g.GET("", func(c *gin.Context) {
			ListImageRepositoriesHandler(c, svc)
		})

		// 获取单个镜像仓库
		g.GET("/:id", func(c *gin.Context) {
			GetImageRepositoryHandler(c, svc)
		})

		// 更新镜像仓库
		g.PUT("/:id", func(c *gin.Context) {
			UpdateImageRepositoryHandler(c, svc)
		})

		// 删除镜像仓库
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteImageRepositoryHandler(c, svc)
		})
	}
}
