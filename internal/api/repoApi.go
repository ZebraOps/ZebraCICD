package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateRepoHandler 创建仓库处理函数
// @Summary 创建仓库
// @Description 创建一个新的仓库
// @Tags repos
// @Accept json
// @Produce json
// @Param repo body model.Repo true "仓库信息"
// @Success 201 {object} model.Repo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/repos [post]
func CreateRepoHandler(c *gin.Context, svc *service.RepoService) {
	var req model.Repo
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateRepo(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ListReposHandler 获取仓库列表处理函数
// @Summary 获取仓库列表
// @Description 获取所有仓库的列表，支持按中文名称、英文名称、部门、编程语言、项目管理者等条件查询
// @Tags repos
// @Produce json
// @Param c_name query string false "中文名称"
// @Param e_name query string false "英文名称"
// @Param repo_department query string false "归属部门"
// @Param language query string false "编程语言"
// @Param repo_manager query string false "项目管理者"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.Repo}}
// @Failure 500 {object} map[string]string
// @Router /api/repos [get]
func ListReposHandler(c *gin.Context, svc *service.RepoService) {
	// 解析查询参数
	cName := c.Query("c_name")
	eName := c.Query("e_name")
	department := c.Query("repo_department")
	language := c.Query("language")
	manager := c.Query("repo_manager")

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
	conditions := types.RepoQueryConditions{
		CName:      cName,
		EName:      eName,
		Department: department,
		Language:   language,
		Manager:    manager,
	}

	// 调用服务层获取分页数据
	repos, total, err := svc.ListReposWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, repos)
}

// GetRepoByIDHandler 根据ID获取仓库处理函数
// @Summary 根据ID获取仓库
// @Description 根据仓库ID获取仓库详情
// @Tags repos
// @Produce json
// @Param id path int true "仓库ID"
// @Success 200 {object} model.Repo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/repos/{id} [get]
func GetRepoByIDHandler(c *gin.Context, svc *service.RepoService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	repo, err := svc.GetRepoByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "repo not found")
		return
	}
	types.Success(c, repo)
}

// UpdateRepoHandler 更新仓库处理函数
// @Summary 更新仓库
// @Description 根据仓库ID更新仓库信息
// @Tags repos
// @Accept json
// @Produce json
// @Param id path int true "仓库ID"
// @Param repo body model.Repo true "仓库信息"
// @Success 200 {object} model.Repo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/repos/{id} [put]
// UpdateRepoHandler 更新仓库处理函数
func UpdateRepoHandler(c *gin.Context, svc *service.RepoService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查仓库是否存在
	existingRepo, err := svc.GetRepoByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "repo not found")
		return
	}

	var req model.Repo
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段，只更新非零值字段
	if req.CName != "" {
		existingRepo.CName = req.CName
	}
	if req.EName != "" {
		existingRepo.EName = req.EName
	}
	if req.RepoURL != "" {
		existingRepo.RepoURL = req.RepoURL
	}
	if req.RepoManager != "" {
		existingRepo.RepoManager = req.RepoManager
	}
	if req.RepoDepartment != "" {
		existingRepo.RepoDepartment = req.RepoDepartment
	}
	if req.RepoLanguage != "" {
		existingRepo.RepoLanguage = req.RepoLanguage
	}
	if req.RepoDesc != "" {
		existingRepo.RepoDesc = req.RepoDesc
	}
	if req.RepoDeployType != "" {
		existingRepo.RepoDeployType = req.RepoDeployType
	}
	if req.RepoBuildPath != "" {
		existingRepo.RepoBuildPath = req.RepoBuildPath
	}

	if err := svc.UpdateRepo(existingRepo); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingRepo)
}

// DeleteRepoHandler 删除仓库处理函数
// @Summary 删除仓库
// @Description 根据仓库ID删除仓库
// @Tags repos
// @Produce json
// @Param id path int true "仓库ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/repos/{id} [delete]
func DeleteRepoHandler(c *gin.Context, svc *service.RepoService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteRepo(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "repo deleted successfully"})
}

// GetRepoURLFromGitLabHandler 根据repoID从GitLab获取仓库信息
// @Summary 根据repoID从GitLab获取仓库信息
// @Description 根据repoID从GitLab获取仓库信息
// @Tags repos
// @Produce json
// @Param repoID path string true "仓库ID"
// @Success 200 {object} types.Response{data=string}
// @Failure 200 {object} types.Response
// @Router /api/repos/gitlab-url/{repoID} [get]
func GetRepoURLFromGitLabHandler(c *gin.Context, svc *service.RepoService) {
	id := c.Param("repoID")

	repoURL, err := svc.GetRepoInfoFromGitLab(id)
	if err != nil {
		types.Error(c, http.StatusNotFound, err.Error())
		return
	}

	types.Success(c, repoURL)
}

// GetRepoTemplatesHandler 获取仓库关联的模板
// @Summary 获取仓库关联的模板
// @Description 根据仓库ID获取所有关联的构建模板
// @Tags repos
// @Produce json
// @Param id path int true "仓库ID"
// @Success 200 {array} model.BuildTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/repos/{id}/templates [get]
func GetRepoTemplatesHandler(c *gin.Context, templateSvc *service.BuildTemplateService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	templates, err := templateSvc.GetTemplatesByRepoID(uint(id))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, templates)
}

// RegisterRepoRoutes 注册仓库相关路由
func RegisterRepoRoutes(r *gin.Engine, svc *service.RepoService, templateSvc *service.BuildTemplateService) {
	g := r.Group("/api/repos")
	{
		// 创建仓库
		g.POST("", func(c *gin.Context) {
			CreateRepoHandler(c, svc)
		})

		// 获取仓库列表
		g.GET("", func(c *gin.Context) {
			ListReposHandler(c, svc)
		})

		// 根据ID获取仓库
		g.GET("/:id", func(c *gin.Context) {
			GetRepoByIDHandler(c, svc)
		})

		// 更新仓库
		g.PUT("/:id", func(c *gin.Context) {
			UpdateRepoHandler(c, svc)
		})

		// 删除仓库
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteRepoHandler(c, svc)
		})

		// 根据英文名称从GitLab获取仓库地址
		g.GET("/gitlab-url/:repoID", func(c *gin.Context) {
			GetRepoURLFromGitLabHandler(c, svc)
		})

		// 获取仓库关联的模板
		g.GET("/:id/templates", func(c *gin.Context) {
			GetRepoTemplatesHandler(c, templateSvc)
		})
	}
}
