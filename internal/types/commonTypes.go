package types

import (
	"net/http"

	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
	"github.com/gin-gonic/gin"
)

// Response 统一API响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Total   int64       `json:"total"`
	Records interface{} `json:"records"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, total int64, records interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: PageResponse{
			Total:   total,
			Records: records,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithHttpStatus 带HTTP状态码的错误响应
func ErrorWithHttpStatus(c *gin.Context, httpCode int, code int, message string) {
	c.JSON(httpCode, Response{
		Code:    code,
		Message: message,
	})
}

// TemplateHistoryResponse 模板修改历史响应结构
type TemplateHistoryResponse struct {
	ID         uint              `json:"id"`
	TemplateID uint              `json:"template_id"`
	Modifier   string            `json:"modifier"`
	Dockerfile string            `json:"dockerfile"`
	Pipeline   string            `json:"pipeline"`
	CreatedAt  timeutil.JSONTime `json:"created_at"`
}

type ClusterQueryConditions struct {
	Name        string `json:"name"`
	Enabled     *bool  `json:"enabled"`
	Vendor      string `json:"vendor"`
	Environment string `json:"environment"`
}

// RepoQueryConditions 仓库查询条件
type RepoQueryConditions struct {
	CName      string `json:"c_name"`
	EName      string `json:"e_name"`
	Department string `json:"department"`
	Language   string `json:"language"`
	Manager    string `json:"manager"`
}

type ServerQueryConditions struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	IsActive *bool  `json:"is_active"`
}

type EnvQueryConditions struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type CloudProviderQueryConditions struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

type DeploymentTemplateQueryConditions struct {
	Name         string `json:"name"`
	TemplateType string `json:"template_type"`
	Status       string `json:"status"`
	Creator      string `json:"creator"`
}

// ImageRepositoryQueryConditions 镜像仓库查询条件
type ImageRepositoryQueryConditions struct {
	Name string `json:"name"` // 仓库名称过滤
	URL  string `json:"url"`  // 仓库地址过滤
	Page int    `json:"page"` // 页码
	Size int    `json:"size"` // 每页数量
}
