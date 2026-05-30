package middleware

import (
	"github.com/gin-gonic/gin"
)

const ContextUserID = "user_id"
const ContextUserName = "user_name"

// UserIdentity reads X-User-Id / X-User-Name injected by ZebraGateway
// and writes them into gin.Context for downstream handlers.
//
// The gateway auth middleware already deletes client-sent X-User-Id/X-User-Name
// headers (preventing forgery) and re-injects authenticated ones from JWT claims.
// This middleware simply extracts those verified headers and stores them in the
// Gin context so that handler functions can access them via c.GetString().
func UserIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-Id")
		userName := c.GetHeader("X-User-Name")
		if userID != "" {
			c.Set(ContextUserID, userID)
		}
		if userName != "" {
			c.Set(ContextUserName, userName)
		}
		c.Next()
	}
}