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
func CreateRepoHandler(c *gin.Context, svc *service.RepoService) {
	var req model.Repo
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.RepoManager == "" {
		req.RepoManager = c.GetString("user_name")
	}
	if err := svc.CreateRepo(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

func ListReposHandler(c *gin.Context, svc *service.RepoService) {
	cName := c.Query("c_name")
	eName := c.Query("e_name")
	department := c.Query("repo_department")
	language := c.Query("language")
	manager := c.Query("repo_manager")

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

	conditions := types.RepoQueryConditions{
		CName:      cName,
		EName:      eName,
		Department: department,
		Language:   language,
		Manager:    manager,
	}

	repos, total, err := svc.ListReposWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.PageSuccess(c, total, repos)
}

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

func UpdateRepoHandler(c *gin.Context, svc *service.RepoService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
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
	if req.CName != "" {
		existingRepo.CName = req.CName
	}
	if req.EName != "" {
		existingRepo.EName = req.EName
	}
	if req.RepoNumber != "" {
		existingRepo.RepoNumber = req.RepoNumber
	}
	if req.RepoURL != "" {
		existingRepo.RepoURL = req.RepoURL
	}
	if req.RepoSSHURL != "" {
		existingRepo.RepoSSHURL = req.RepoSSHURL
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

func GetRepoURLFromGitLabHandler(c *gin.Context, svc *service.RepoService) {
	id := c.Param("repoID")
	repoURL, err := svc.GetRepoInfoFromGitLab(id)
	if err != nil {
		types.Error(c, http.StatusNotFound, err.Error())
		return
	}
	types.Success(c, repoURL)
}

func ListRepoBranchesHandler(c *gin.Context, svc *service.RepoService) {
	appIDStr := c.Query("application_id")
	if appIDStr == "" {
		types.Error(c, http.StatusBadRequest, "application_id is required")
		return
	}
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid application_id format")
		return
	}
	branches, err := svc.ListRepoBranchesByApp(uint(appID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, branches)
}

func ListRepoTagsHandler(c *gin.Context, svc *service.RepoService) {
	appIDStr := c.Query("application_id")
	if appIDStr == "" {
		types.Error(c, http.StatusBadRequest, "application_id is required")
		return
	}
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid application_id format")
		return
	}
	tags, err := svc.ListRepoTagsByApp(uint(appID))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, tags)
}

// RegisterRepoRoutes 注册仓库相关路由
func RegisterRepoRoutes(r *gin.Engine, svc *service.RepoService) {
	g := r.Group("/api/repos")
	{
		g.POST("", func(c *gin.Context) {
			CreateRepoHandler(c, svc)
		})
		g.GET("", func(c *gin.Context) {
			ListReposHandler(c, svc)
		})
		// 分支和标签列表（必须在 /:id 之前注册，避免路径冲突）
		g.GET("/branches", func(c *gin.Context) {
			ListRepoBranchesHandler(c, svc)
		})
		g.GET("/tags", func(c *gin.Context) {
			ListRepoTagsHandler(c, svc)
		})
		g.GET("/gitlab-url/:repoID", func(c *gin.Context) {
			GetRepoURLFromGitLabHandler(c, svc)
		})
		g.GET("/:id", func(c *gin.Context) {
			GetRepoByIDHandler(c, svc)
		})
		g.PUT("/:id", func(c *gin.Context) {
			UpdateRepoHandler(c, svc)
		})
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteRepoHandler(c, svc)
		})
	}
}