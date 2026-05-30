package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

func CreateJenkinsPlatformHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
	var req model.JenkinsPlatform
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateJenkinsPlatform(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

func ListJenkinsPlatformsHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
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

	conditions := types.JenkinsPlatformQueryConditions{Name: name, Status: status}
	platforms, total, err := svc.ListJenkinsPlatformsWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.PageSuccess(c, total, platforms)
}

func GetJenkinsPlatformHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	platform, err := svc.GetJenkinsPlatformByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "jenkins platform not found")
		return
	}
	types.Success(c, platform)
}

func UpdateJenkinsPlatformHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	existing, err := svc.GetJenkinsPlatformByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "jenkins platform not found")
		return
	}
	var req model.JenkinsPlatform
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.URL != "" {
		existing.URL = req.URL
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		existing.Password = req.Password
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if err := svc.UpdateJenkinsPlatform(existing); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existing)
}

func DeleteJenkinsPlatformHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	if err := svc.DeleteJenkinsPlatform(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "jenkins platform deleted successfully"})
}

func ConnectJenkinsPlatformHandler(c *gin.Context, svc *service.JenkinsPlatformService) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}
	if err := svc.TestJenkinsPlatformConnection(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "connection successful"})
}

func RegisterJenkinsPlatformRoutes(r *gin.Engine, svc *service.JenkinsPlatformService) {
	g := r.Group("/api/jenkins-platforms")
	{
		g.POST("", func(c *gin.Context) { CreateJenkinsPlatformHandler(c, svc) })
		g.GET("", func(c *gin.Context) { ListJenkinsPlatformsHandler(c, svc) })
		g.GET("/:id", func(c *gin.Context) { GetJenkinsPlatformHandler(c, svc) })
		g.PUT("/:id", func(c *gin.Context) { UpdateJenkinsPlatformHandler(c, svc) })
		g.DELETE("/:id", func(c *gin.Context) { DeleteJenkinsPlatformHandler(c, svc) })
		g.POST("/:id/connect", func(c *gin.Context) { ConnectJenkinsPlatformHandler(c, svc) })
	}
}