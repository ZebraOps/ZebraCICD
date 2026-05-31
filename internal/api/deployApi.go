package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

	// 设置默认部署类型
	if req.DeployType == "" {
		req.DeployType = "k8s"
	}

	// 根据部署类型验证
	switch req.DeployType {
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
	default:
		types.Error(c, http.StatusBadRequest, "DeployType must be 'k8s' or 'docker'")
		return
	}

	if req.JenkinsJobName == "" {
		types.Error(c, http.StatusBadRequest, "JenkinsJobName is required")
		return
	}

	if req.HarborProject == "" || req.ImageName == "" {
		types.Error(c, http.StatusBadRequest, "HarborProject and ImageName are required")
		return
	}

	if req.DeploymentName == "" {
		req.DeploymentName = fmt.Sprintf("app-%d", req.ProjectID)
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
// @Description 分页查询部署任务，支持按状态、项目ID筛选
// @Tags deploys
// @Produce json
// @Param status query string false "任务状态"
// @Param project_id query int false "项目ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} types.Response
// @Failure 500 {object} types.Response
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

	output, err := svc.GetTaskConsole(uint(id))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"output": output})
}

// streamDeployTaskConsoleHandler WebSocket 实时推送 Jenkins 构建日志
// @Summary 流式获取任务控制台日志
// @Description 通过 WebSocket 实时推送部署任务的 Jenkins 构建日志
// @Tags deploys
// @Param id path int true "任务ID"
// @Router /api/deploys/{id}/console/stream [get]
func streamDeployTaskConsoleHandler(c *gin.Context, svc *service.DeployService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 定期轮询 Jenkins 日志并推送到 WebSocket 客户端
	var lastOutput string
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 获取任务详情以检查状态
			task, err := svc.GetTask(uint(id))
			if err != nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"task not found"}`))
				return
			}

			// 获取控制台输出
			output, err := svc.GetTaskConsole(uint(id))
			if err != nil {
				// Jenkins 构建可能尚未开始，静默跳过本次推送
				continue
			}

			// 仅在内容有变化时推送，避免重复发送
			if output != lastOutput {
				msg := fmt.Sprintf(`{"output":"%s","status":"%s"}`, output, task.Status)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					return // 客户端已断开
				}
				lastOutput = output
			}

			// 任务达到终态时发送关闭消息并退出
			if task.Status == "SUCCESS" || task.Status == "FAILED" {
				closeMsg := fmt.Sprintf(`{"output":"%s","status":"%s","finished":true}`, output, task.Status)
				_ = conn.WriteMessage(websocket.TextMessage, []byte(closeMsg))
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "task finished"))
				return
			}

		case <-c.Request.Context().Done():
			// 请求上下文取消（客户端断开或超时）
			return
		}
	}
}

// RegisterDeployRoutes 注册部署相关路由
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

		g.GET("/:id", func(c *gin.Context) {
			getDeployTaskHandler(c, svc)
		})

		g.GET("/:id/console", func(c *gin.Context) {
			getDeployTaskConsoleHandler(c, svc)
		})

		// WebSocket 实时日志流
		g.GET("/:id/console/stream", func(c *gin.Context) {
			streamDeployTaskConsoleHandler(c, svc)
		})

		g.DELETE("/:id", func(c *gin.Context) {
			deleteDeployTaskHandler(c, svc)
		})
	}
}