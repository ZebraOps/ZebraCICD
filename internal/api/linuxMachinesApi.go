package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service" // 添加此行
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// GetServerHandler 根据ID获取服务器
// @Summary 根据ID获取服务器
// @Description 根据服务器ID获取服务器详情
// @Tags linux-machines
// @Produce json
// @Param id path int true "服务器ID"
// @Success 200 {object} model.Server
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines/{id} [get]
func GetServerHandler(c *gin.Context, svc *service.ServerService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	server, err := svc.GetServerByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "server not found")
		return
	}
	types.Success(c, server)
}

// ListServersHandler 获取服务器列表
// @Summary 获取服务器列表
// @Description 获取所有服务器的列表，支持按名称、主机等条件查询
// @Tags linux-machines
// @Produce json
// @Param name query string false "服务器名称"
// @Param host query string false "服务器主机地址"
// @Param isActive query bool false "是否激活"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.Server}}
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines [get]
func ListServersHandler(c *gin.Context, svc *service.ServerService) {
	// 解析查询参数
	name := c.Query("name")
	host := c.Query("host")
	isActiveStr := c.Query("isActive")

	var isActive *bool
	if isActiveStr != "" {
		isActiveVal, err := strconv.ParseBool(isActiveStr)
		if err == nil {
			isActive = &isActiveVal
		}
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

	// 构建查询条件
	conditions := types.ServerQueryConditions{
		Name:     name,
		Host:     host,
		IsActive: isActive,
	}

	// 调用服务层获取分页数据
	servers, total, err := svc.ListServersWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, servers)
}

// CreateServerHandler 创建服务器
// @Summary 创建服务器
// @Description 创建一个新的服务器连接信息
// @Tags linux-machines
// @Accept json
// @Produce json
// @Param server body model.Server true "服务器信息"
// @Success 201 {object} model.Server
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines [post]
func CreateServerHandler(c *gin.Context, svc *service.ServerService) {
	var req model.Server
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateServer(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// UpdateServerHandler 更新服务器
// @Summary 更新服务器
// @Description 根据服务器ID更新服务器信息
// @Tags linux-machines
// @Accept json
// @Produce json
// @Param id path int true "服务器ID"
// @Param server body model.Server true "服务器信息"
// @Success 200 {object} model.Server
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines/{id} [put]
func UpdateServerHandler(c *gin.Context, svc *service.ServerService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查服务器是否存在
	existingServer, err := svc.GetServerByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "server not found")
		return
	}

	var req model.Server
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段
	if req.Name != "" {
		existingServer.Name = req.Name
	}
	if req.Description != "" {
		existingServer.Description = req.Description
	}
	if req.Host != "" {
		existingServer.Host = req.Host
	}
	if req.Port != 0 {
		existingServer.Port = req.Port
	}
	if req.Username != "" {
		existingServer.Username = req.Username
	}
	if req.AuthType != "" {
		existingServer.AuthType = req.AuthType
	}
	if req.Password != "" {
		existingServer.Password = req.Password
	}
	if req.PrivateKey != "" {
		existingServer.PrivateKey = req.PrivateKey
	}
	existingServer.IsActive = req.IsActive

	if err := svc.UpdateServer(existingServer); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingServer)
}

// DeleteServerHandler 删除服务器
// @Summary 删除服务器
// @Description 根据服务器ID删除服务器
// @Tags linux-machines
// @Produce json
// @Param id path int true "服务器ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines/{id} [delete]
func DeleteServerHandler(c *gin.Context, svc *service.ServerService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteServer(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "server deleted successfully"})
}

// ConnectServerHandler 连接服务器
// @Summary 连接服务器
// @Description 测试连接到服务器
// @Tags linux-machines
// @Produce json
// @Param id path int true "服务器ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines/{id}/connect [post]
func ConnectServerHandler(c *gin.Context, svc *service.ServerService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.TestConnection(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "connection successful"})
}

// ListContainersHandler 获取Docker容器列表
// @Summary 获取Docker容器列表
// @Description 获取服务器上的Docker容器列表
// @Tags linux-machines
// @Produce json
// @Param id path int true "服务器ID"
// @Success 200 {array} model.DockerContainer
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/linux-machines/{id}/containers [get]
func ListContainersHandler(c *gin.Context, svc *service.ServerService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	containers, err := svc.ListContainers(uint(id))
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, containers)
}

// RegisterServerRoutes 注册服务器相关路由
func RegisterServerRoutes(r *gin.Engine, svc *service.ServerService) {
	g := r.Group("/api/linux-machines")
	{
		// 创建服务器
		g.POST("", func(c *gin.Context) {
			CreateServerHandler(c, svc)
		})

		// 获取服务器列表
		g.GET("", func(c *gin.Context) {
			ListServersHandler(c, svc)
		})

		// 根据ID获取服务器
		g.GET("/:id", func(c *gin.Context) {
			GetServerHandler(c, svc)
		})

		// 更新服务器
		g.PUT("/:id", func(c *gin.Context) {
			UpdateServerHandler(c, svc)
		})

		// 删除服务器
		g.DELETE("/:id", func(c *gin.Context) {
			DeleteServerHandler(c, svc)
		})

		// 测试连接服务器
		g.POST("/:id/connect", func(c *gin.Context) {
			ConnectServerHandler(c, svc)
		})

		// 获取容器列表
		g.GET("/:id/containers", func(c *gin.Context) {
			ListContainersHandler(c, svc)
		})
	}
}
