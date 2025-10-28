package middleware

import (
	"net/http"

	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
// 这个中间件用于保护需要身份验证的 API 端点
// 它会验证每个请求中的 JWT 令牌，确保只有已认证的用户才能访问受保护的资源
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 步骤 1：优先从 Cookie 中读取 JWT 令牌
		// 因为登录时将 token 存储在 HttpOnly Cookie 中（更安全，防止 XSS 攻击）
		tokenCookie, err := c.Cookie("access_token")
		if err == nil && tokenCookie != "" {
			// 成功从 Cookie 中获取到 token
			token = tokenCookie
		} else {
			// 如果 Cookie 中没有 token，尝试从 Authorization 请求头中读取
			// 这样可以同时支持 Cookie 和 Bearer Token 两种认证方式
			token, err = utils.GetAccessToken(c)
			if err != nil {
				// 两种方式都失败，返回未授权错误
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required: no token found in cookie or authorization header"})
				c.Abort()
				return
			}
		}

		// 步骤 2：检查令牌是否为空
		// 防止空令牌通过验证
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is required"})
			c.Abort()
			return
		}

		// 步骤 3：验证 JWT 令牌的有效性
		// 检查令牌是否：
		// - 格式正确
		// - 签名有效
		// - 未过期
		// - 未被篡改
		claims, err := utils.ValidateToken(token)
		if err != nil {
			// 如果令牌验证失败（过期、无效、被篡改等）
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// 步骤 4：将用户信息存储到请求上下文中
		// 这样后续的处理器就可以直接获取用户信息，无需重复验证
		c.Set("userID", claims.UserID) // 存储用户ID，用于数据查询和权限控制
		c.Set("role", claims.Role)     // 存储用户角色，用于权限判断

		// 步骤 5：继续执行下一个处理器
		// 只有通过所有验证的请求才能到达这里
		c.Next()
	}
}
