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

// CreateBuildTemplateHandler 创建构建模板处理函数
func CreateBuildTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
	var req model.BuildTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// 自动填充 creator：优先取网关注入的用户名，其次取请求体字段
	if req.Creator == "" {
		req.Creator = c.GetString("user_name")
	}
	if err := svc.CreateTemplate(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// GetBuildTemplateHandler 根据ID获取构建模板处理函数
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
func ListBuildTemplatesHandler(c *gin.Context, svc *service.BuildTemplateService) {
	// 获取查询参数
	name := c.Query("name")
	language := c.Query("language")
	department := c.Query("department")
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
	templates, total, err := svc.ListTemplates(name, language, department, creator, updater, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回分页结果
	types.PageSuccess(c, total, templates)
}

// UpdateBuildTemplateHandler 更新构建模板处理函数
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
	if req.Department != "" {
		existingTemplate.Department = req.Department
	}
	if req.Creator != "" {
		existingTemplate.Creator = req.Creator
	}
	// updater 强制使用网关注入的当前用户名，确保修改人始终为实际操作者
	existingTemplate.Updater = c.GetString("user_name")
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

		// 关联应用和构建模板
		g.POST("/:id/applications/:applicationId", func(c *gin.Context) {
			AssociateApplicationWithTemplateHandler(c, svc)
		})

		// 取消应用和构建模板关联
		g.DELETE("/:id/applications/:applicationId", func(c *gin.Context) {
			DisassociateApplicationWithTemplateHandler(c, svc)
		})

		// 获取构建模板关联的应用列表
		g.GET("/:id/applications", func(c *gin.Context) {
			getBuildTemplateApplicationsHandler(c, svc)
		})
	}
}

// AssociateApplicationWithTemplateHandler 关联应用和构建模板
func AssociateApplicationWithTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
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

	types.Success(c, gin.H{"message": "构建模板关联应用成功"})
}

// DisassociateApplicationWithTemplateHandler 取消应用和构建模板关联
func DisassociateApplicationWithTemplateHandler(c *gin.Context, svc *service.BuildTemplateService) {
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

// getBuildTemplateApplicationsHandler 获取构建模板关联的应用列表
func getBuildTemplateApplicationsHandler(c *gin.Context, svc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	apps, err := svc.GetApplicationsByTemplateID(uint(id))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, apps)
}