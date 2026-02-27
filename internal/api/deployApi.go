package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
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

// RegisterDeployRoutes 注册部署相关路由
func RegisterDeployRoutes(r *gin.Engine, svc *service.DeployService) {
	g := r.Group("/api/deploys")
	{
		// create deploy task
		g.POST("", func(c *gin.Context) {
			createDeployTaskHandler(c, svc)
		})

		g.GET("/:id", func(c *gin.Context) {
			getDeployTaskHandler(c, svc)
		})
	}
}
