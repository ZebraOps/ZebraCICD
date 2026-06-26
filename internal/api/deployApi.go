package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys [post]
func createDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	var req model.DeployTask
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 验证必需字段
	if req.ProjectID == 0 || req.EnvID == 0 {
		types.Error(c, http.StatusBadRequest, "ProjectID and EnvID are required")
		return
	}

	// 设置默认部署目标
	if req.DeployTarget == "" && req.DeployType != "" {
		req.DeployTarget = req.DeployType // 兼容旧数据
	}
	if req.DeployTarget == "" {
		req.DeployTarget = "k8s"
	}
	// 同步 DeployType 用于兼容
	req.DeployType = req.DeployTarget

	// 根据部署目标验证
	switch req.DeployTarget {
	case "k8s":
		if req.K8sClusterID == 0 {
			types.Error(c, http.StatusBadRequest, "K8sClusterID is required for k8s deployment")
			return
		}
		if req.K8sNamespace == "" {
			req.K8sNamespace = "default"
		}
	case "docker":
		if req.ServerID == 0 {
			types.Error(c, http.StatusBadRequest, "ServerID is required for docker deployment")
			return
		}
	case "linux":
		if req.ServerID == 0 {
			types.Error(c, http.StatusBadRequest, "ServerID is required for linux deployment")
			return
		}
		if req.DeployPath == "" {
			types.Error(c, http.StatusBadRequest, "DeployPath is required for linux deployment")
			return
		}
	default:
		types.Error(c, http.StatusBadRequest, "DeployTarget must be 'k8s', 'docker', or 'linux'")
		return
	}

	if req.JenkinsJobName == "" {
		types.Error(c, http.StatusBadRequest, "JenkinsJobName is required")
		return
	}

	if req.RegistryProject == "" || req.ImageName == "" {
		types.Error(c, http.StatusBadRequest, "RegistryProject and ImageName are required")
		return
	}

	if req.GitRef == "" {
		req.GitRef = "main" // 默认分支
	}

	if err := svc.CreateTask(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"task_id": req.ID})
}

// GetDeployTask 获取部署任务
// @Summary 根据ID获取部署任务
// @Description 根据任务ID获取部署任务详情
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 404 {object} types.Response
// @Router /api/deploys/{id} [get]
func getDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	t, err := svc.GetTask(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "task not found")
		return
	}
	types.Success(c, t)
}

// ListDeployTasks 获取部署任务列表
// @Summary 获取部署任务列表
// @Description 分页查询部署任务，支持按状态、项目ID、环境ID、归属部门筛选
// @Tags deploys
// @Produce json
// @Param status query string false "任务状态"
// @Param project_id query int false "项目ID"
// @Param env_id query int false "环境ID"
// @Param department query string false "归属部门"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys [get]
func listDeployTasksHandler(c *gin.Context, svc *service.DeployService) {
	status := c.Query("status")
	projectIDStr := c.Query("project_id")
	envIDStr := c.Query("env_id")
	department := c.Query("department")

	var projectID uint
	if projectIDStr != "" {
		if id, err := strconv.Atoi(projectIDStr); err == nil {
			projectID = uint(id)
		}
	}

	var envID uint
	if envIDStr != "" {
		if id, err := strconv.Atoi(envIDStr); err == nil {
			envID = uint(id)
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

	tasks, total, err := svc.ListTasks(status, projectID, envID, department, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, tasks)
}

// DeleteDeployTask 删除部署任务
// @Summary 删除部署任务
// @Description 根据ID删除部署任务，同时删除关联的Jenkins Job
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id} [delete]
func deleteDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteTask(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "deleted"})
}

// batchDeleteRequest 批量删除请求体
type batchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// BatchDeleteDeployTasks 批量删除部署任务
// @Summary 批量删除部署任务
// @Description 批量删除部署任务，同时删除关联的Jenkins Job
// @Tags deploys
// @Accept json
// @Produce json
// @Param ids body []uint true "任务ID列表"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/batch-delete [post]
func batchDeleteDeployTasksHandler(c *gin.Context, svc *service.DeployService) {
	var req batchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := svc.BatchDeleteTasks(req.IDs); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": fmt.Sprintf("%d tasks deleted", len(req.IDs))})
}

// retryDeployTaskHandler 重试失败的部署任务
// @Summary 重试失败的部署任务
// @Description 将失败的任务重置为PENDING状态并重新入队执行
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/retry [post]
func retryDeployTaskHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	task, err := svc.RetryTask(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "只能重试失败的任务") || strings.Contains(err.Error(), "任务不存在") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{"task_id": task.ID, "retry_count": task.RetryCount})
}

// retryDeployTaskStageHandler 从指定阶段重试失败的任务
// @Summary 从指定阶段重试失败的任务
// @Description 支持从 BUILDING（完整重试）或 DEPLOYING（仅部署）阶段重试失败的任务
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Param stage body object true "重试阶段 {\"stage\": \"BUILDING\"|\"DEPLOYING\"}"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/retry-stage [post]
func retryDeployTaskStageHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	var req struct {
		Stage string `json:"stage" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, "stage is required (BUILDING or DEPLOYING)")
		return
	}

	if req.Stage != "BUILDING" && req.Stage != "DEPLOYING" {
		types.Error(c, http.StatusBadRequest, "stage must be BUILDING or DEPLOYING")
		return
	}

	task, err := svc.RetryTaskFromStage(uint(id), req.Stage)
	if err != nil {
		if strings.Contains(err.Error(), "只能重试") || strings.Contains(err.Error(), "不支持的阶段") ||
			strings.Contains(err.Error(), "构建阶段未成功") || strings.Contains(err.Error(), "任务不存在") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{"task_id": task.ID, "retry_count": task.RetryCount, "stage": req.Stage})
}

// getAvailableTemplatesHandler 获取创建任务时可选的构建/部署模板
// @Summary 获取创建任务时可选的构建/部署模板
// @Description 根据应用ID获取可用的构建模板和部署模板列表
// @Tags deploys
// @Produce json
// @Param app_id query int true "应用ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/templates [get]
func getAvailableTemplatesHandler(c *gin.Context, svc *service.DeployService) {
	appIDStr := c.Query("app_id")
	if appIDStr == "" {
		types.Error(c, http.StatusBadRequest, "app_id is required")
		return
	}
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid app_id format")
		return
	}

	templates, err := svc.GetAvailableTemplatesForTask(uint(appID), 0)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, templates)
}

// getDeployTaskConsoleHandler 获取 Jenkins 控制台输出
// @Summary 获取部署任务Jenkins控制台输出
// @Description 获取指定部署任务的Jenkins构建控制台日志
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/console [get]
func getDeployTaskConsoleHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	task, err := svc.GetTask(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "task not found")
		return
	}

	output, err := svc.GetTaskConsole(uint(id))
	if err != nil {
		if errors.Is(err, service.ErrNoJenkinsBuildInfo) {
			types.Success(c, gin.H{"output": "", "status": task.Status})
			return
		}
		// 控制台输出获取失败但仍返回状态信息
		types.Success(c, gin.H{"output": "", "status": task.Status, "error": err.Error()})
		return
	}

	types.Success(c, gin.H{"output": output, "status": task.Status})
}

// RegisterDeployRoutes 注册部署相关路由
// getTaskStagesHandler 返回任务的所有阶段历史记录
// @Summary 获取任务阶段历史
// @Description 获取指定部署任务的CICD流程各阶段执行详情
// @Tags deploys
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/stages [get]
func getTaskStagesHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	stages, err := svc.GetTaskStages(uint(id))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, "获取阶段历史失败: "+err.Error())
		return
	}

	types.Success(c, stages)
}

// getRollbackHistoryHandler 获取可回滚的历史部署任务列表
// @Summary 获取可回滚的历史版本
// @Description 获取与当前任务相同部署配置的历史成功版本列表
// @Tags deploys
// @Produce json
// @Param id path int true "当前任务ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/rollback-history [get]
func getRollbackHistoryHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

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

	tasks, total, err := svc.GetRollbackHistory(uint(id), page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, tasks)
}

// rollbackDeployRequest 回滚请求体
type rollbackDeployRequest struct {
	HistoryTaskID uint `json:"history_task_id" binding:"required"`
}

// rollbackDeployHandler 执行部署回滚
// @Summary 执行部署回滚
// @Description 基于历史任务创建新的回滚任务，使用历史镜像版本重新部署
// @Tags deploys
// @Accept json
// @Produce json
// @Param id path int true "当前任务ID"
// @Param request body rollbackDeployRequest true "回滚参数"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/rollback [post]
func rollbackDeployHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	currentTaskID, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	var req rollbackDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, "history_task_id is required")
		return
	}

	newTask, err := svc.RollbackDeployment(uint(currentTaskID), req.HistoryTaskID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "不匹配") || strings.Contains(err.Error(), "只能回滚") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{
		"task_id":       newTask.ID,
		"image_tag":     newTask.ImageTag,
		"is_rollback":   newTask.IsRollback,
		"rollback_from": newTask.RollbackFrom,
	})
}

// triggerBuildHandler 手动触发构建
// @Summary 手动触发构建
// @Description 触发手动执行模式任务的构建阶段
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/trigger-build [post]
func triggerBuildHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	task, err := svc.TriggerBuild(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "只有手动") || strings.Contains(err.Error(), "状态") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{
		"task_id":     task.ID,
		"status":      task.Status,
		"build_status": task.BuildStatus,
		"image_tag":   task.ImageTag,
	})
}

// triggerDeployHandler 手动触发部署
// @Summary 手动触发部署
// @Description 触发手动执行模式任务的部署阶段（构建需已完成）
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/trigger-deploy [post]
func triggerDeployHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	task, err := svc.TriggerDeploy(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "只有手动") || strings.Contains(err.Error(), "尚未完成") || strings.Contains(err.Error(), "状态") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{
		"task_id":      task.ID,
		"status":       task.Status,
		"deploy_status": task.DeployStatus,
	})
}

// triggerAllHandler 一键执行构建+部署
// @Summary 一键执行构建和部署
// @Description 一键触发手动执行模式任务的构建和部署
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/trigger [post]
func triggerAllHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	task, err := svc.TriggerAll(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "只有手动") || strings.Contains(err.Error(), "状态") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{
		"task_id":      task.ID,
		"status":       task.Status,
		"build_status": task.BuildStatus,
		"deploy_status": task.DeployStatus,
	})
}

// cancelScheduleHandler 取消定时任务
// @Summary 取消定时任务
// @Description 取消计划执行的定时任务
// @Tags deploys
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} types.Response
// @Failure 400 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/{id}/cancel-schedule [delete]
func cancelScheduleHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.CancelScheduledTask(uint(id)); err != nil {
		if strings.Contains(err.Error(), "只有 SCHEDULED") {
			types.Error(c, http.StatusBadRequest, err.Error())
		} else {
			types.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	types.Success(c, gin.H{"message": "scheduled task cancelled"})
}

// listScheduledTasksHandler 获取定时任务列表
// @Summary 获取定时任务列表
// @Description 获取待执行的定时任务列表
// @Tags deploys
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} types.Response
// @Failure 500 {object} types.Response
// @Router /api/deploys/scheduled [get]
func listScheduledTasksHandler(c *gin.Context, svc *service.DeployService) {
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

	tasks, total, err := svc.ListScheduledTasks(page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, tasks)
}

func RegisterDeployRoutes(r *gin.Engine, svc *service.DeployService) {
	g := r.Group("/api/deploys")
	{
		// 获取可用模板
		g.GET("/templates", func(c *gin.Context) {
			getAvailableTemplatesHandler(c, svc)
		})

		// list
		g.GET("", func(c *gin.Context) {
			listDeployTasksHandler(c, svc)
		})

		// create deploy task
		g.POST("", func(c *gin.Context) {
			createDeployTaskHandler(c, svc)
		})

		// batch delete
		g.POST("/batch-delete", func(c *gin.Context) {
			batchDeleteDeployTasksHandler(c, svc)
		})

		// retry failed deploy task
		g.POST("/:id/retry", func(c *gin.Context) {
			retryDeployTaskHandler(c, svc)
		})

		// retry failed deploy task from specific stage
		g.POST("/:id/retry-stage", func(c *gin.Context) {
			retryDeployTaskStageHandler(c, svc)
		})

		// 回滚相关
		g.GET("/:id/rollback-history", func(c *gin.Context) {
			getRollbackHistoryHandler(c, svc)
		})
		g.POST("/:id/rollback", func(c *gin.Context) {
			rollbackDeployHandler(c, svc)
		})

		// 手动触发相关
		g.POST("/:id/trigger-build", func(c *gin.Context) {
			triggerBuildHandler(c, svc)
		})
		g.POST("/:id/trigger-deploy", func(c *gin.Context) {
			triggerDeployHandler(c, svc)
		})
		g.POST("/:id/trigger", func(c *gin.Context) {
			triggerAllHandler(c, svc)
		})
		g.DELETE("/:id/cancel-schedule", func(c *gin.Context) {
			cancelScheduleHandler(c, svc)
		})

		// 定时任务列表
		g.GET("/scheduled", func(c *gin.Context) {
			listScheduledTasksHandler(c, svc)
		})

		g.GET("/:id", func(c *gin.Context) {
			getDeployTaskHandler(c, svc)
		})

		g.GET("/:id/console", func(c *gin.Context) {
			getDeployTaskConsoleHandler(c, svc)
		})

		// 获取任务阶段历史
		g.GET("/:id/stages", func(c *gin.Context) {
			getTaskStagesHandler(c, svc)
		})

		g.DELETE("/:id", func(c *gin.Context) {
			deleteDeployTaskHandler(c, svc)
		})
	}
}
