package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

func CreateJenkinsCredentialHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	var req model.JenkinsCredential
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateCredential(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

func ListJenkinsCredentialsHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		// 前端按列搜索凭据ID时会传 credential_id，这里兼容映射到模糊搜索字段
		name = strings.TrimSpace(c.Query("credential_id"))
	}
	credType := strings.TrimSpace(c.Query("credential_type"))
	status := strings.TrimSpace(c.Query("status"))

	var platformID uint
	if pidStr := c.Query("jenkins_platform_id"); pidStr != "" {
		if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
			platformID = uint(pid)
		}
	}

	page, size := 1, 10
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

	conditions := types.JenkinsCredentialQueryConditions{
		Name:              name,
		CredentialType:    credType,
		Status:            status,
		JenkinsPlatformID: platformID,
	}
	creds, total, err := svc.ListCredentials(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.PageSuccess(c, total, creds)
}

func GetJenkinsCredentialHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	cred, err := svc.GetCredentialByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "jenkins credential not found")
		return
	}
	types.Success(c, cred)
}

func UpdateJenkinsCredentialHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	existing, err := svc.GetCredentialByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "jenkins credential not found")
		return
	}
	var req model.JenkinsCredential
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Scope != "" {
		existing.Scope = req.Scope
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if err := svc.UpdateCredential(existing); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existing)
}

func DeleteJenkinsCredentialHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	if err := svc.DeleteCredential(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "jenkins credential deleted successfully"})
}

func SyncJenkinsCredentialsHandler(c *gin.Context, svc *service.JenkinsCredentialService) {
	var req struct {
		JenkinsPlatformID uint `json:"jenkins_platform_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, "jenkins_platform_id is required")
		return
	}
	result, err := svc.SyncCredentials(req.JenkinsPlatformID)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, result)
}

func RegisterJenkinsCredentialRoutes(r *gin.Engine, svc *service.JenkinsCredentialService) {
	g := r.Group("/api/jenkins-credentials")
	{
		g.POST("/sync", func(c *gin.Context) { SyncJenkinsCredentialsHandler(c, svc) })
		g.POST("", func(c *gin.Context) { CreateJenkinsCredentialHandler(c, svc) })
		g.GET("", func(c *gin.Context) { ListJenkinsCredentialsHandler(c, svc) })
		g.GET("/:id", func(c *gin.Context) { GetJenkinsCredentialHandler(c, svc) })
		g.PUT("/:id", func(c *gin.Context) { UpdateJenkinsCredentialHandler(c, svc) })
		g.DELETE("/:id", func(c *gin.Context) { DeleteJenkinsCredentialHandler(c, svc) })
	}
}
