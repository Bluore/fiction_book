package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/d60-Lab/gin-template/pkg/config"
	"github.com/d60-Lab/gin-template/pkg/jwt"
	"github.com/d60-Lab/gin-template/pkg/response"
)

// Auth JWT 认证中间件
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := jwt.ParseToken(token, cfg.JWT.Secret)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// AdminOnly 管理员权限中间件
// 注意：此中间件需要配合 Casbin 或用户角色系统使用
// 使用方法：在路由中添加 middleware.AdminOnly("admin") 来限制只有管理员可访问
func AdminOnly(allowedRoles ...string) gin.HandlerFunc {
	// 默认允许 admin 角色
	if len(allowedRoles) == 0 {
		allowedRoles = []string{"admin"}
	}

	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 从上下文获取用户角色（需要在 Auth 中间件中设置）
		role, exists := c.Get("userRole")
		if !exists {
			// 如果没有设置角色，默认拒绝访问
			// 实际项目中可以从数据库/缓存查询用户角色
			response.Forbidden(c, "permission denied: role not found")
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok {
			response.Forbidden(c, "permission denied: invalid role type")
			c.Abort()
			return
		}

		// 检查用户角色是否在允许列表中
		allowed := false
		for _, r := range allowedRoles {
			if userRole == r {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Forbidden(c, "permission denied: insufficient privileges")
			c.Abort()
			return
		}

		_ = userID // 可用于日志记录
		c.Next()
	}
}
