package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// internal/api/deployApi.go 增强版
// CreateDeployTask 创建部署任务
// @Summary 创建部署任务
// @Description 创建一个新的部署任务，触发Jenkins构建并部署到K8s
// @Tags deploys
// @Accept json
// @Produce json
// @Param task body model.DeployTask true "部署任务信息"
// @Success 202 {object} map[string]uint
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/deploys [post]
func createDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	var req model.DeployTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证必需字段
	if req.ProjectID == 0 || req.EnvID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ProjectID and EnvID are required"})
		return
	}

	if req.K8sClusterID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "K8sClusterID is required"})
		return
	}

	if req.JenkinsJobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JenkinsJobName is required"})
		return
	}

	if req.HarborProject == "" || req.ImageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HarborProject and ImageName are required"})
		return
	}

	if req.DeploymentName == "" {
		req.DeploymentName = fmt.Sprintf("app-%d", req.ProjectID)
	}

	if req.K8sNamespace == "" {
		req.K8sNamespace = "default"
	}

	if req.GitRef == "" {
		req.GitRef = "main" // 默认分支
	}

	if err := svc.CreateTask(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"task_id": req.ID})
}

// GetDeployTask 获取部署任务
// @Summary 根据ID获取部署任务
// @Description 根据任务ID获取部署任务详情
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} model.DeployTask
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/deploys/{id} [get]
func getDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	t, err := svc.GetTask(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// ListDeployTasks 获取部署任务列表
// @Summary 获取部署任务列表
// @Description 分页查询部署任务，支持按状态、项目ID筛选
// @Tags deploys
// @Produce json
// @Param status query string false "任务状态"
// @Param project_id query int false "项目ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} types.PageResponse
// @Failure 500 {object} map[string]string
// @Router /api/deploys [get]
func listDeployTasksHandler(c *gin.Context, svc *service.DeployService) {
	status := c.Query("status")
	projectIDStr := c.Query("project_id")

	var projectID uint
	if projectIDStr != "" {
		if id, err := strconv.Atoi(projectIDStr); err == nil {
			projectID = uint(id)
		}
	}

	page := 1
	size := 20
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

	tasks, total, err := svc.ListTasks(status, projectID, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, tasks)
}

// DeleteDeployTask 删除部署任务
// @Summary 删除部署任务
// @Description 根据ID删除部署任务
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/deploys/{id} [delete]
func deleteDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := svc.DeleteTask(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// RegisterDeployRoutes 注册部署相关路由
func RegisterDeployRoutes(r *gin.Engine, svc *service.DeployService) {
	g := r.Group("/api/deploys")
	{
		// list
		g.GET("", func(c *gin.Context) {
			listDeployTasksHandler(c, svc)
		})

		// create deploy task
		g.POST("", func(c *gin.Context) {
			createDeployTaskHandler(c, svc)
		})

		g.GET("/:id", func(c *gin.Context) {
			getDeployTaskHandler(c, svc)
		})

		g.DELETE("/:id", func(c *gin.Context) {
			deleteDeployTaskHandler(c, svc)
		})
	}
}
