package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// GetClusterByIDHandler 根据ID获取K8s集群
// @Summary 根据ID获取K8s集群
// @Description 根据集群ID获取K8s集群详情
// @Tags k8s
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} model.K8SCluster
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters/{id} [get]
func GetClusterByIDHandler(c *gin.Context, svc *service.K8SService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	cluster, err := svc.GetClusterByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "cluster not found")
		return
	}
	types.Success(c, cluster)
}

// ListClustersHandler 获取K8s集群列表
// @Summary 获取K8s集群列表
// @Description 获取所有K8s集群的列表，支持按名称、状态等条件查询
// @Tags k8s
// @Produce json
// @Param name query string false "集群名称"
// @Param enabled query bool false "是否启用"
// @Param vendor query string false "集群厂商"
// @Param environment query string false "集群环境"
// @Param page query int false "页码" default(1)
// @Param size query int false "页数" default(10)
// @Success 200 {object} types.Response{data=types.PageResponse{records=[]model.K8SCluster}}
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters [get]
func ListClustersHandler(c *gin.Context, svc *service.K8SService) {
	// 解析查询参数
	name := c.Query("name")
	enabledStr := c.Query("enabled")
	vendor := c.Query("vendor")
	environment := c.Query("environment")

	var enabled *bool
	if enabledStr != "" {
		enabledVal, err := strconv.ParseBool(enabledStr)
		if err == nil {
			enabled = &enabledVal
		}
	}
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
	conditions := types.ClusterQueryConditions{
		Name:        name,
		Enabled:     enabled,
		Vendor:      vendor,
		Environment: environment,
	}

	// 调用服务层获取分页数据
	clusters, total, err := svc.ListClustersWithConditions(conditions, page, size)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.PageSuccess(c, total, clusters)
}

// UpdateClusterHandler 更新K8s集群
// @Summary 更新K8s集群
// @Description 根据集群ID更新K8s集群信息
// @Tags k8s
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param cluster body model.K8SCluster true "K8s集群信息"
// @Success 200 {object} model.K8SCluster
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters/{id} [put]
func UpdateClusterHandler(c *gin.Context, svc *service.K8SService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// 检查集群是否存在
	existingCluster, err := svc.GetClusterByID(uint(id))
	if err != nil {
		types.Error(c, http.StatusNotFound, "cluster not found")
		return
	}

	var req model.K8SCluster
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 选择性更新字段
	if req.Name != "" {
		existingCluster.Name = req.Name
	}
	if req.Description != "" {
		existingCluster.Description = req.Description
	}
	if req.ApiServer != "" {
		existingCluster.ApiServer = req.ApiServer
	}
	if req.CaCert != "" {
		existingCluster.CaCert = req.CaCert
	}
	if req.ClientCert != "" {
		existingCluster.ClientCert = req.ClientCert
	}
	if req.ClientKey != "" {
		existingCluster.ClientKey = req.ClientKey
	}
	if req.Token != "" {
		existingCluster.Token = req.Token
	}
	if req.Namespace != "" {
		existingCluster.Namespace = req.Namespace
	}
	existingCluster.SkipVerify = req.SkipVerify
	existingCluster.IsActive = req.IsActive
	existingCluster.Vendor = req.Vendor
	existingCluster.Environment = req.Environment
	existingCluster.Enabled = req.Enabled

	if err := svc.UpdateCluster(existingCluster); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, existingCluster)
}

// DeleteClusterHandler 删除K8s集群
// @Summary 删除K8s集群
// @Description 根据集群ID删除K8s集群
// @Tags k8s
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters/{id} [delete]
func DeleteClusterHandler(c *gin.Context, svc *service.K8SService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := svc.DeleteCluster(uint(id)); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, gin.H{"message": "cluster deleted successfully"})
}

// CreateK8SClusterHandler 创建K8s集群
// @Summary 创建K8s集群
// @Description 创建一个新的K8s集群凭证
// @Tags k8s
// @Accept json
// @Produce json
// @Param cluster body model.K8SCluster true "K8s集群信息"
// @Success 201 {object} model.K8SCluster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters [post]
func CreateK8SClusterHandler(c *gin.Context, svc *service.K8SService) {
	var req model.K8SCluster
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.CreateCluster(&req); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, req)
}

// ConnectK8SClusterHandler 连接K8s集群
// @Summary 连接K8s集群
// @Description 测试连接到K8s集群
// @Tags k8s
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters/{id}/connect [post]
func ConnectK8SClusterHandler(c *gin.Context, svc *service.K8SService) {
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

// ListPodsHandler 获取Pod列表
// @Summary 获取Pod列表
// @Description 获取K8s集群中的Pod列表
// @Tags k8s
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string false "命名空间" default(default)
// @Success 200 {array} types.PodInfo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/k8s/clusters/{id}/pods [get]
func ListPodsHandler(c *gin.Context, svc *service.K8SService) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	pods, err := svc.ListPods(uint(id), namespace)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	types.Success(c, pods)
}

// RegisterK8SRoutes 注册K8s相关路由
func RegisterK8SRoutes(r *gin.Engine, svc *service.K8SService) {
	g := r.Group("/api/k8s")
	{
		clusters := g.Group("/clusters")
		{
			// 创建K8s集群
			clusters.POST("", func(c *gin.Context) {
				CreateK8SClusterHandler(c, svc)
			})

			// 获取集群列表
			clusters.GET("", func(c *gin.Context) {
				ListClustersHandler(c, svc)
			})

			// 根据ID获取集群
			clusters.GET("/:id", func(c *gin.Context) {
				GetClusterByIDHandler(c, svc)
			})

			// 更新集群
			clusters.PUT("/:id", func(c *gin.Context) {
				UpdateClusterHandler(c, svc)
			})

			// 测试连接K8s集群
			clusters.POST("/:id/connect", func(c *gin.Context) {
				ConnectK8SClusterHandler(c, svc)
			})

			// 获取Pod列表
			clusters.GET("/:id/pods", func(c *gin.Context) {
				ListPodsHandler(c, svc)
			})

			// 删除集群
			clusters.DELETE("/:id", func(c *gin.Context) {
				DeleteClusterHandler(c, svc)
			})
		}
	}
}
