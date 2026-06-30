package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateLanguageHandler 创建开发语言
func CreateLanguageHandler(c *gin.Context, svc *service.LanguageService) {
	var req model.Language
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateLanguage(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListLanguagesHandler 获取开发语言列表
func ListLanguagesHandler(c *gin.Context, svc *service.LanguageService) {
	name := c.Query("name")
	status := c.Query("status")

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

	conditions := types.LanguageQueryConditions{
		Name:   name,
		Status: status,
	}

	langs, total, err := svc.ListLanguagesWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, langs)
}

// GetLanguageHandler 根据ID获取开发语言
func GetLanguageHandler(c *gin.Context, svc *service.LanguageService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	lang, err := svc.GetLanguageByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "language not found")
		return
	}
	types.Success(c, lang)
}

// UpdateLanguageHandler 更新开发语言
func UpdateLanguageHandler(c *gin.Context, svc *service.LanguageService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	existing, err := svc.GetLanguageByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "language not found")
		return
	}

	var req model.Language
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Icon != "" {
		existing.Icon = req.Icon
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if req.SortOrder != 0 {
		existing.SortOrder = req.SortOrder
	}
	if req.Description != "" {
		existing.Description = req.Description
	}

	if err := svc.UpdateLanguage(existing); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existing)
}

// DeleteLanguageHandler 删除开发语言
func DeleteLanguageHandler(c *gin.Context, svc *service.LanguageService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteLanguage(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "language deleted successfully"})
}

// RegisterLanguageRoutes 注册开发语言相关路由
func RegisterLanguageRoutes(r *gin.Engine, svc *service.LanguageService) {
	g := r.Group("/api/languages")
	{
		g.POST("", func(c *gin.Context) {
			CreateLanguageHandler(c, svc)
		})
		g.GET("", func(c *gin.Context) {
			ListLanguagesHandler(c, svc)
		})
		g.GET("/:id", func(c *gin.Context) {
			GetLanguageHandler(c, svc)
		})
		g.PUT("/:id", func(c *gin.Context) {
			UpdateLanguageHandler(c, svc)
		})
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteLanguageHandler(c, svc)
		})
	}
}
