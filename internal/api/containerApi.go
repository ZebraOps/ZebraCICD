package api

import (
	"net/http"
	"strconv"

	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket" // 添加 WebSocket 导入
)

// 定义 WebSocket Upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中，应该更严格地检查 Origin
		return true
	},
}

// ExecContainerHandler 在容器中执行命令
// @Summary 在容器中执行命令
// @Description 在指定容器中执行命令
// @Tags containers
// @Accept json
// @Produce json
// @Param id path int true "服务器ID"
// @Param containerID path string true "容器ID"
// @Param command body types.ContainerExecRequest true "执行命令"
// @Success 200 {object} types.ContainerExecResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/servers/{id}/containers/{containerID}/exec [post]
func ExecContainerHandler(c *gin.Context, svc *service.ServerService) {
	serverIDStr := c.Param("id")
	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid server id format")
		return
	}

	containerID := c.Param("containerID")

	var req types.ContainerExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		types.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := svc.ExecContainer(uint(serverID), containerID, req.Command)
	if err != nil {
		types.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	types.Success(c, result)
}

// AttachContainerHandler 连接到容器
// @Summary 连接到容器
// @Description 连接到指定容器（类似docker attach）
// @Tags containers
// @Produce json
// @Param id path int true "服务器ID"
// @Param containerID path string true "容器ID"
// @Success 101 {object} object "Switching Protocols"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/servers/{id}/containers/{containerID}/attach [get]
func AttachContainerHandler(c *gin.Context, svc *service.ServerService) {
	serverIDStr := c.Param("id")
	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		types.Error(c, http.StatusBadRequest, "invalid server id format")
		return
	}

	containerID := c.Param("containerID")

	// 升级到WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.S().Errorf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	if err := svc.AttachContainer(uint(serverID), containerID, conn); err != nil {
		log.S().Errorf("attach container failed: %v", err)
		return
	}
}

// RegisterContainerRoutes 注册容器相关路由
func RegisterContainerRoutes(r *gin.Engine, svc *service.ServerService) {
	g := r.Group("/api/servers/:id/containers")
	{
		// 在容器中执行命令
		g.POST("/:containerID/exec", func(c *gin.Context) {
			ExecContainerHandler(c, svc)
		})

		// 连接到容器
		g.GET("/:containerID/attach", func(c *gin.Context) {
			AttachContainerHandler(c, svc)
		})
	}
}
