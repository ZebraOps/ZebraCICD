package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
)

// CreateContainerOperationHandler records a new container operation
// @Summary 记录容器操作
// @Description 记录容器操作（重启、删除、终端等）
// @Tags container-operations
// @Accept json
// @Produce json
// @Param operation body model.ContainerOperation true "操作记录"
// @Success 200 {object} model.ContainerOperation
// @Failure 400 {object} map[string]string
// @Router /api/container-operations [post]
func CreateContainerOperationHandler(c *gin.Context, repo *handler.ContainerOperationRepository) {
	var op model.ContainerOperation
	if err := c.ShouldBindJSON(&op); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := repo.Create(&op); err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, op)
}

// ListContainerOperationsHandler returns paginated operation history
// @Summary 获取容器操作历史
// @Description 分页获取容器操作历史记录
// @Tags container-operations
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param operation_type query string false "操作类型"
// @Param target_type query string false "目标类型"
// @Param result query string false "操作结果"
// @Param operator query string false "操作人"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} types.PageResult
// @Router /api/container-operations [get]
func ListContainerOperationsHandler(c *gin.Context, repo *handler.ContainerOperationRepository) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	filters := map[string]interface{}{
		"operation_type": c.Query("operation_type"),
		"target_type":    c.Query("target_type"),
		"result":         c.Query("result"),
		"operator":       c.Query("operator"),
		"start_time":     c.Query("start_time"),
		"end_time":       c.Query("end_time"),
	}

	records, total, err := repo.List(page, size, filters)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, map[string]interface{}{
		"records": records,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// RegisterContainerOperationRoutes registers container operation history routes
func RegisterContainerOperationRoutes(r *gin.Engine, repo *handler.ContainerOperationRepository) {
	g := r.Group("/api/container-operations")
	{
		g.GET("", func(c *gin.Context) {
			ListContainerOperationsHandler(c, repo)
		})
		g.POST("", func(c *gin.Context) {
			CreateContainerOperationHandler(c, repo)
		})
	}
}
