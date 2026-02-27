package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateEnvHandler 创建环境
// @Summary 创建环境
// @Description 创建一个新的环境
// @Tags environments
// @Accept json
// @Produce json
// @Param env body model.Environment true "环境信息"
// @Success 201 {object} model.Environment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/environments [post]
func CreateEnvHandler(c *gin.Context, svc *service.EnvService) {
	var req model.Environment
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateEnv(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListEnvsHandler 获取环境列表
// @Summary 获取环境列表
// @Description 获取所有环境的列表，支持按名称、类型、状态等条件查询
// @Tags environments
// @Produce json
// @Param name query string false "环境名称"
// @Param type query string false "环境类型"
// @Param status query string false "环境状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.Environment}}
// @Failure 500 {object} map[string]string
// @Router /api/environments [get]
func ListEnvsHandler(c *gin.Context, svc *service.EnvService) {
	// 解析查询参数
	name := c.Query("name")
	envType := c.Query("type")
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
	conditions := types.EnvQueryConditions{
		Name:   name,
		Type:   envType,
		Status: status,
	}

	// 调用服务层获取分页数据
	envs, total, err := svc.ListEnvsWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, envs)
}

// GetEnvHandler 根据ID获取环境
// @Summary 根据ID获取环境
// @Description 根据环境ID获取环境详情
// @Tags environments
// @Produce json
// @Param id path int true "环境ID"
// @Success 200 {object} model.Environment
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/environments/{id} [get]
func GetEnvHandler(c *gin.Context, svc *service.EnvService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	env, err := svc.GetEnvByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "environment not found")
		return
	}
	types.Success(c, env)
}

// UpdateEnvHandler 更新环境
// @Summary 更新环境
// @Description 根据环境ID更新环境信息
// @Tags environments
// @Accept json
// @Produce json
// @Param id path int true "环境ID"
// @Param env body model.Environment true "环境信息"
// @Success 200 {object} model.Environment
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/environments/{id} [put]
func UpdateEnvHandler(c *gin.Context, svc *service.EnvService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查环境是否存在
	existingEnv, err := svc.GetEnvByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "environment not found")
		return
	}

	var req model.Environment
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段
	if req.Name != "" {
		existingEnv.Name = req.Name
	}
	if req.Description != "" {
		existingEnv.Description = req.Description
	}
	if req.Type != "" {
		existingEnv.Type = req.Type
	}
	if req.Status != "" {
		existingEnv.Status = req.Status
	}
	if req.Config != "" {
		existingEnv.Config = req.Config
	}

	if err := svc.UpdateEnv(existingEnv); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingEnv)
}

// DeleteEnvHandler 删除环境
// @Summary 删除环境
// @Description 根据环境ID删除环境（软删除）
// @Tags environments
// @Produce json
// @Param id path int true "环境ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/environments/{id} [delete]
func DeleteEnvHandler(c *gin.Context, svc *service.EnvService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteEnv(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "environment deleted successfully"})
}

// RegisterEnvRoutes 注册环境相关路由
func RegisterEnvRoutes(r *gin.Engine, svc *service.EnvService) {
	g := r.Group("/api/environments")
	{
		// 创建环境
		g.POST("", func(c *gin.Context) {
			CreateEnvHandler(c, svc)
		})

		// 获取环境列表
		g.GET("", func(c *gin.Context) {
			ListEnvsHandler(c, svc)
		})

		// 根据ID获取环境
		g.GET("/:id", func(c *gin.Context) {
			GetEnvHandler(c, svc)
		})

		// 更新环境
		g.PUT("/:id", func(c *gin.Context) {
			UpdateEnvHandler(c, svc)
		})

		// 删除环境
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteEnvHandler(c, svc)
		})
	}
}
