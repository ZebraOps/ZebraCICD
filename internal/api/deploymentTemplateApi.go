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
// @Summary 创建部署模板
// @Description 创建一个新的部署模板
// @Tags deployment-templates
// @Accept json
// @Produce json
// @Param template body model.DeploymentTemplate true "部署模板信息"
// @Success 201 {object} model.DeploymentTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment [post]
func CreateDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	var req model.DeploymentTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateDeploymentTemplate(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListDeploymentTemplatesHandler 获取部署模板列表
// @Summary 获取部署模板列表
// @Description 获取所有部署模板的列表，支持按名称、类型、状态等条件查询
// @Tags deployment-templates
// @Produce json
// @Param name query string false "部署模板名称"
// @Param type query string false "模板类型"
// @Param status query string false "状态"
// @Param creator query string false "创建人"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.DeploymentTemplate}}
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment [get]
func ListDeploymentTemplatesHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	// 解析查询参数
	name := c.Query("name")
	templateType := c.Query("template_type")
	status := c.Query("status")
	creator := c.Query("creator")

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
// @Summary 根据ID获取部署模板
// @Description 根据部署模板ID获取部署模板详情
// @Tags deployment-templates
// @Produce json
// @Param id path int true "部署模板ID"
// @Success 200 {object} model.DeploymentTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id} [get]
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
// @Summary 更新部署模板
// @Description 根据部署模板ID更新部署模板信息
// @Tags deployment-templates
// @Accept json
// @Produce json
// @Param id path int true "部署模板ID"
// @Param template body model.DeploymentTemplate true "部署模板信息"
// @Success 200 {object} model.DeploymentTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id} [put]
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
	if req.DisplayName != "" {
		existingTemplate.DisplayName = req.DisplayName
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
	if req.Parameters != "" {
		existingTemplate.Parameters = req.Parameters
	}
	if req.Version != "" {
		existingTemplate.Version = req.Version
	}
	if req.Status != "" {
		existingTemplate.Status = req.Status
	}
	if req.Updater != "" {
		existingTemplate.Updater = req.Updater
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
// @Summary 删除部署模板
// @Description 根据部署模板ID删除部署模板（软删除）
// @Tags deployment-templates
// @Produce json
// @Param id path int true "部署模板ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id} [delete]
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

// AssociateRepoWithDeployTemplateHandler 关联仓库和部署模板
// @Summary 关联仓库和部署模板
// @Description 将仓库与部署模板进行关联
// @Tags deployment-template-repo
// @Produce json
// @Param id path int true "模板ID"
// @Param repoId path int true "仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id}/repos/{repoId} [post]
func AssociateRepoWithDeployTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	repoIDStr := c.Param("repoId")

	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	repoID, err := strconv.Atoi(repoIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid repo id format")
		return
	}

	if err := svc.AddRepoToTemplate(uint(templateID), uint(repoID)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "部署模板关联仓库成功。"})
}

// DisassociateRepoWithDeploymentTemplateHandler 取消仓库和部署模板关联
// @Summary 取消仓库和部署模板关联
// @Description 取消仓库与部署模板的关联
// @Tags deployment-template-repo
// @Produce json
// @Param id path int true "模板ID"
// @Param repoId path int true "仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id}/repos/{repoId} [delete]
func DisassociateRepoWithDeploymentTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	repoIDStr := c.Param("repoId")

	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	repoID, err := strconv.Atoi(repoIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid repo id format")
		return
	}

	if err := svc.RemoveRepoFromTemplate(uint(templateID), uint(repoID)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, gin.H{"message": "association removed successfully"})
}

// GetReposByTemplateHandler 根据部署模板ID获取关联的仓库列表
// @Summary 获取关联的仓库列表
// @Description 根据部署模板ID获取所有关联的仓库列表
// @Tags deployment-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {array} model.Repo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id}/repos [get]
func GetReposByTemplateHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	repos, err := svc.GetReposByTemplateID(uint(templateID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, repos)
}

// GetDeploymentTemplateHistoryHandler 获取部署模板历史记录
// @Summary 获取部署模板历史记录
// @Description 根据部署模板ID获取模板的修改历史记录
// @Tags deployment-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {array} model.DeploymentTemplateHistory
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/deployment/{id}/history [get]
func GetDeploymentTemplateHistoryHandler(c *gin.Context, svc *service.DeploymentTemplateService) {
	templateIDStr := c.Param("id")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid template id format")
		return
	}

	history, err := svc.GetDeploymentTemplateHistory(uint(templateID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, history)
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

		// 关联仓库和部署模板
		g.POST("/:id/repos/:repoId", func(c *gin.Context) {
			AssociateRepoWithDeployTemplateHandler(c, svc)
		})

		// 取消仓库和部署模板关联
		g.DELETE("/:id/repos/:repoId", func(c *gin.Context) {
			DisassociateRepoWithDeploymentTemplateHandler(c, svc)
		})

		// 获取关联的仓库列表
		g.GET("/:id/repos", func(c *gin.Context) {
			GetReposByTemplateHandler(c, svc)
		})

		// 获取模板修改历史
		g.GET("/:id/history", func(c *gin.Context) {
			GetDeploymentTemplateHistoryHandler(c, svc)
		})
	}
}
