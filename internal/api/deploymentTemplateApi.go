package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
	"github.com/gin-gonic/gin"
)

// CreateDeploymentTemplateHandler 创建部署模板
func CreateDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	var req model.DeploymentTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// 自动填充 creator：优先取网关注入的用户名，其次取请求体字段
	if req.Creator == "" {
		req.Creator = c.GetString("user_name")
	}
	if err := svc.CreateDeploymentTemplate(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListDeploymentTemplatesHandler 获取部署模板列表
func ListDeploymentTemplatesHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	// 解析查询参数
	name := c.Query("name")
	templateType := c.Query("template_type")
	status := c.Query("status")
	creator := c.Query("creator")
	department := c.Query("department")

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
	conditions := types.DeploymentTemplateQueryConditions{
		Name:         name,
		TemplateType: templateType,
		Status:       status,
		Creator:      creator,
		Department:   department,
	}

	// 调用服务层获取分页数据
	templates, total, err := svc.ListDeploymentTemplatesWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, templates)
}

// GetDeploymentTemplateHandler 根据ID获取部署模板
func GetDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 传递请求上下文
	template, err := svc.GetDeploymentTemplateByID(c.Request.Context(), uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "deployment template not found")
		return
	}
	types.Success(c, template)
}

// UpdateDeploymentTemplateHandler 更新部署模板
func UpdateDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查部署模板是否存在
	existingTemplate, err := svc.GetDeploymentTemplateByID(c.Request.Context(), uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "deployment template not found")
		return
	}

	var req model.DeploymentTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段
	if req.Name != "" {
		existingTemplate.Name = req.Name
	}
	if req.Description != "" {
		existingTemplate.Description = req.Description
	}
	if req.TemplateType != "" {
		existingTemplate.TemplateType = req.TemplateType
	}
	if req.Content != "" {
		existingTemplate.Content = req.Content
	}
	if req.Variables != "" {
		existingTemplate.Variables = req.Variables
	}
	if req.Version != "" {
		existingTemplate.Version = req.Version
	}
	if req.Status != "" {
		existingTemplate.Status = req.Status
	}
			// updater 强制使用网关注入的当前用户名，确保修改人始终为实际操作者
			existingTemplate.Updater = c.GetString("user_name")
	if req.Department != "" {
		existingTemplate.Department = req.Department
	}
	existingTemplate.UpdatedAt = timeutil.Now()

	// 从请求头或请求体获取修改原因
	changeReason := c.PostForm("change_reason")
	if changeReason == "" {
		changeReason = "模板更新"
	}

	if err := svc.UpdateDeploymentTemplate(c, existingTemplate, changeReason); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingTemplate)
}

// DeleteDeploymentTemplateHandler 删除部署模板
func DeleteDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteDeploymentTemplate(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "deployment template deleted successfully"})
}

// GetDeploymentTemplateHistoryHandler 获取部署模板历史记录
func GetDeploymentTemplateHistoryHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

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

	// 调用服务层获取分页数据
	history, total, err := svc.GetDeploymentTemplateHistoryPaginated(uint(templateID), page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	types.PageSuccess(c, total, history)
}

// RegisterDeploymentTemplateRoutes 注册部署模板相关路由
func RegisterDeploymentTemplateRoutes(r *gin.Engine, svc *service.DeploymentTemplateService) {
	g := r.Group("/api/templates/deployment")
	{
		// 创建部署模板
		g.POST("", func(c *gin.Context) {
			CreateDeploymentTemplateHandler(c, svc)
		})

		// 获取部署模板列表
		g.GET("", func(c *gin.Context) {
			ListDeploymentTemplatesHandler(c, svc)
		})

		// 根据ID获取部署模板
		g.GET("/:id", func(c *gin.Context) {
			GetDeploymentTemplateHandler(c, svc)
		})

		// 更新部署模板
		g.PUT("/:id", func(c *gin.Context) {
			UpdateDeploymentTemplateHandler(c, svc)
		})

		// 删除部署模板
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteDeploymentTemplateHandler(c, svc)
		})

		// 获取模板修改历史
		g.GET("/:id/history", func(c *gin.Context) {
			GetDeploymentTemplateHistoryHandler(c, svc)
		})

		// 关联应用和部署模板
		g.POST("/:id/applications/:applicationId", func(c *gin.Context) {
			AssociateApplicationWithDeployTemplateHandler(c, svc)
		})

		// 取消应用和部署模板关联
		g.DELETE("/:id/applications/:applicationId", func(c *gin.Context) {
			DisassociateApplicationWithDeploymentTemplateHandler(c, svc)
		})

		// 获取部署模板关联的应用列表
		g.GET("/:id/applications", func(c *gin.Context) {
			GetApplicationsByDeployTemplateHandler(c, svc)
		})

			// 回退部署模板到指定历史版本
			g.POST("/:id/rollback/:historyId", func(c *gin.Context) {
				RollbackDeploymentTemplateHandler(c, svc)
			})
	}
}

// AssociateApplicationWithDeployTemplateHandler 关联应用和部署模板
func AssociateApplicationWithDeployTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	applicationIDStr := c.Param("applicationId")

	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	applicationID, err := strconv.Atoi(applicationIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid application id format")
		return
	}

	if err := svc.AddApplicationToTemplate(uint(templateID), uint(applicationID)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "部署模板关联应用成功"})
}

// DisassociateApplicationWithDeploymentTemplateHandler 取消应用和部署模板关联
func DisassociateApplicationWithDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	applicationIDStr := c.Param("applicationId")

	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	applicationID, err := strconv.Atoi(applicationIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid application id format")
		return
	}

	if err := svc.RemoveApplicationFromTemplate(uint(templateID), uint(applicationID)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "association removed successfully"})
}

// GetApplicationsByDeployTemplateHandler 根据部署模板ID获取关联的应用列表
func GetApplicationsByDeployTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	apps, err := svc.GetApplicationsByTemplateID(uint(templateID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, apps)
}
// RollbackDeploymentTemplateHandler 回退部署模板到指定历史版本
func RollbackDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}
	historyID, err := strconv.Atoi(c.Param("historyId"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid history id format")
		return
	}
	template, err := svc.RollbackDeploymentTemplate(uint(templateID), uint(historyID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, template)
}
