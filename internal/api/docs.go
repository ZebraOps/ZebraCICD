package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterDocsRoutes registers Swagger UI and documentation endpoints
func RegisterDocsRoutes(r *gin.Engine) {
	r.StaticFile("/favicon.ico", "./docs/favicon.ico")

	// 从本地文件服务 Swagger UI，而不是使用 CDN
	r.GET("/swagger/*any", ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "NAME_OF_ENVIRONMENT_VARIABLE"))

	// 保留原有的API文档接口
	r.GET("/docs/swagger.json", func(c *gin.Context) {
		c.File("docs/swagger.json")
	})

	r.GET("/docs", func(c *gin.Context) {
		html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Zebra CICD API Docs</title>
  <!-- 使用国内 CDN 或本地资源 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@4.18.3/swagger-ui.css" />
  <link rel="shortcut icon" href="/favicon.ico" type="image/x-icon"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@4.18.3/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      const ui = SwaggerUIBundle({
        url: '/docs/swagger.json',
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
        ],
        layout: "BaseLayout"
      })
      window.ui = ui
    }
  </script>
</body>
</html>`
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	})
}
