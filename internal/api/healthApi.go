package api

import (
	"net/http"

	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthCheckHandler 健康检查接口
// @Summary 健康检查
// @Description 检查服务是否正常运行
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheckHandler(c *gin.Context, db *gorm.DB) {
	// 获取底层数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to get database connection",
		})
		return
	}

	// 检查数据库连接
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database connection failed",
		})
		return
	}

	types.Success(c, "Zebra CICD is running.")
}

// RegisterHealthRoutes 注册环境相关路由
func RegisterHealthRoutes(r *gin.Engine, db *gorm.DB) {
	g := r.Group("/health")
	{
		// 检查服务是否正常
		g.GET("", func(c *gin.Context) {
			HealthCheckHandler(c, db)
		})
	}
}
