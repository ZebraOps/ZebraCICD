package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openzipkin/zipkin-go"
	"go.uber.org/zap"
)

// BodyLogWriter 包装 gin.ResponseWriter 以捕获响应体
type BodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w BodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// RequestLogger 创建增强版请求日志中间件
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 捕获请求体（如果需要）
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			// 恢复请求体，以便后续处理使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// 包装 ResponseWriter 以捕获响应体
		blw := &BodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 请求处理完成后记录日志
		duration := time.Since(start)

		// 获取基本信息
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// 从上下文中提取 traceId
		var traceID string
		if span := zipkin.SpanFromContext(c.Request.Context()); span != nil {
			traceID = span.Context().TraceID.String()
		}

		// 构造结构化日志
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.Duration("duration", duration),
			zap.String("trace_id", traceID),
		}

		//// 添加请求体信息（如果需要）
		//if len(reqBody) > 0 {
		//	fields = append(fields, zap.ByteString("request_body", reqBody))
		//}
		//
		//// 添加响应体信息（如果需要，注意性能影响）
		//if blw.body.Len() > 0 {
		//	fields = append(fields, zap.ByteString("response_body", blw.body.Bytes()))
		//}

		// 如果有错误信息则添加
		if errorMessage != "" {
			fields = append(fields, zap.String("error", errorMessage))
		}

		// 根据状态码决定日志级别
		if statusCode >= 500 {
			logger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request processed", fields...)
		}
	}
}
