package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
	"github.com/gin-gonic/gin"
)

// CreateBuildTemplateHandler 创建构建模板处理函数
// @Summary 创建构建模板
// @Description 创建一个新的构建模板
// @Tags buildTemplates
// @Accept json
// @Produce json
// @Param template body model.BuildTemplate true "构建模板信息"
// @Success 201 {object} model.BuildTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build [post]
func CreateBuildTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	var req model.BuildTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateTemplate(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// GetBuildTemplateHandler 根据ID获取构建模板处理函数
// @Summary 根据ID获取构建模板
// @Description 根据模板ID获取构建模板详情
// @Tags buildTemplates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} model.BuildTemplate
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{id} [get]
func GetBuildTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	template, err := svc.GetTemplate(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "template not found")
		return
	}
	types.Success(c, template)
}

// ListBuildTemplatesHandler 获取构建模板列表处理函数
// @Summary 获取构建模板列表
// @Description 获取所有构建模板的列表，支持按名称、语言、创建者过滤，支持分页
// @Tags buildTemplates
// @Produce json
// @Param name query string false "模板名称"
// @Param language query string false "编程语言"
// @Param creator query string false "创建者"
// @Param updater query string false "修改人"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.BuildTemplate}}
// @Failure 500 {object} map[string]string
// @Router /api/templates/build [get]
func ListBuildTemplatesHandler(c *gin.Context, svc *service.BuildTemplateService) {
	// 获取查询参数
	name := c.Query("name")
	language := c.Query("language")
	creator := c.Query("creator")
	updater := c.Query("updater")

	// 获取分页参数
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

	// 调用服务层获取数据
	templates, total, err := svc.ListTemplates(name, language, creator, updater, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	types.PageSuccess(c, total, templates)
}

// UpdateBuildTemplateHandler 更新构建模板处理函数
// @Summary 更新构建模板
// @Description 根据模板ID更新构建模板信息
// @Tags buildTemplates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param template body model.BuildTemplate true "构建模板信息"
// @Success 200 {object} model.BuildTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{id} [put]
func UpdateBuildTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查模板是否存在
	existingTemplate, err := svc.GetTemplate(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "template not found")
		return
	}

	var req model.BuildTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 更新字段
	if req.Name != "" {
		existingTemplate.Name = req.Name
	}
	if req.Language != "" {
		existingTemplate.Language = req.Language
	}
	if req.Creator != "" {
		existingTemplate.Creator = req.Creator
	}
	if req.Updater != "" {
		existingTemplate.Updater = req.Updater
	}
	if req.Dockerfile != "" {
		existingTemplate.Dockerfile = req.Dockerfile
	}
	if req.Pipeline != "" {
		existingTemplate.Pipeline = req.Pipeline
	}

	existingTemplate.UpdatedAt = timeutil.Now()

	if err := svc.UpdateTemplate(existingTemplate); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingTemplate)
}

// DeleteBuildTemplateHandler 删除构建模板处理函数
// @Summary 删除构建模板
// @Description 根据模板ID删除构建模板
// @Tags buildTemplates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{id} [delete]
func DeleteBuildTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteTemplate(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "template deleted successfully"})
}

// GetTemplateHistoryHandler 获取模板修改历史
// @Summary 获取模板修改历史
// @Description 根据模板ID获取模板的修改历史记录，支持分页
// @Tags buildTemplates
// @Produce json
// @Param id path int true "模板ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.TemplateHistory}}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{id}/history [get]
func GetTemplateHistoryHandler(c *gin.Context, svc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
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
	history, total, err := svc.GetTemplateHistoryPaginated(uint(id), page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	types.PageSuccess(c, total, history)
}

// AssociateRepoWithTemplateHandler 关联仓库和模板
// @Summary 关联仓库和模板
// @Description 将仓库与构建模板进行关联
// @Tags template-repo
// @Produce json
// @Param templateId path int true "模板ID"
// @Param repoId path int true "仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{templateId}/repos/{repoId} [post]
func AssociateRepoWithTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	templateIDStr := c.Param("templateId")
	repoIDStr := c.Param("repoId")
	fmt.Println(templateIDStr, repoIDStr, 8888)

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

	types.Success(c, gin.H{"message": "构建模板关联服务成功。"})
}

// DisassociateRepoWithTemplateHandler 取消仓库和模板关联
// @Summary 取消仓库和模板关联
// @Description 取消仓库与构建模板的关联
// @Tags template-repo
// @Produce json
// @Param templateId path int true "模板ID"
// @Param repoId path int true "仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/templates/build/{templateId}/repos/{repoId} [delete]
func DisassociateRepoWithTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	templateIDStr := c.Param("templateId")
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

// RegisterTemplateRoutes 注册模板相关路由
func RegisterTemplateRoutes(r *gin.Engine, svc *service.BuildTemplateService) {
	g := r.Group("/api/templates/build")
	{
		// 创建模板
		g.POST("", func(c *gin.Context) {
			CreateBuildTemplateHandler(c, svc)
		})

		// 获取模板列表
		g.GET("", func(c *gin.Context) {
			ListBuildTemplatesHandler(c, svc)
		})

		// 根据ID获取模板
		g.GET("/:id", func(c *gin.Context) {
			GetBuildTemplateHandler(c, svc)
		})

		// 更新模板
		g.PUT("/:id", func(c *gin.Context) {
			UpdateBuildTemplateHandler(c, svc)
		})

		// 删除模板
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteBuildTemplateHandler(c, svc)
		})

		// 获取模板修改历史
		g.GET("/:id/history", func(c *gin.Context) {
			GetTemplateHistoryHandler(c, svc)
		})

		// 关联仓库和模板
		g.POST("/:templateId/repos/:repoId", func(c *gin.Context) {
			AssociateRepoWithTemplateHandler(c, svc)
		})

		// 取消仓库和模板关联
		g.DELETE("/{templateId}/repos/{repoId}", func(c *gin.Context) {
			DisassociateRepoWithTemplateHandler(c, svc)
		})
	}
}
