package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/KamisAyaka/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	// 创建一个默认的 Gin 路由器
	router := gin.Default()

	// 健康检查端点，用于确认服务器是否正常运行
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Server is running"})
	})

	// 加载 .env 环境变量文件
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: unable to find .env")
	}

	// ==================== CORS 配置开始 ====================
	// CORS (Cross-Origin Resource Sharing) 跨域资源共享
	// 当前端（比如运行在 localhost:5173 的 React 应用）想要访问后端 API（运行在 localhost:8080）时，
	// 浏览器会进行跨域检查。没有 CORS 配置，浏览器会阻止这些请求。

	// 从环境变量中读取允许的前端域名列表
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if allowedOrigins != "" {
		// 如果设置了环境变量，按逗号分割成多个域名
		origins = strings.Split(allowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
			log.Println("Allowed Origin:", origins[i])
		}
	} else {
		// 如果没有设置，默认允许本地开发环境的 Vite 服务器（端口 5173）
		origins = []string{"http://localhost:5173"}
		log.Println("Allowed Origin: http://localhost:5173")
	}

	// 创建 CORS 配置对象
	config := cors.Config{}

	// AllowOrigins: 允许哪些域名访问这个 API
	// 例如: ["http://localhost:5173", "https://yourdomain.com"]
	config.AllowOrigins = origins

	// AllowMethods: 允许的 HTTP 方法
	// GET: 获取数据, POST: 创建数据, PUT/PATCH: 更新数据, DELETE: 删除数据, OPTIONS: 预检请求
	config.AllowMethods = []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}

	// AllowHeaders: 允许前端发送的请求头
	// Origin: 请求来源, Content-Type: 内容类型（如 application/json）, Authorization: 认证令牌
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}

	// ExposeHeaders: 允许前端 JavaScript 读取的响应头
	config.ExposeHeaders = []string{"Content-Length"}

	// AllowCredentials: 是否允许发送 Cookie 和认证信息
	// 设为 true 时，前端可以在请求中携带 cookies、HTTP 认证及客户端 SSL 证书
	config.AllowCredentials = true

	// MaxAge: 预检请求（OPTIONS）的结果可以被缓存多久
	// 12 小时内，浏览器不需要重复发送 OPTIONS 预检请求
	config.MaxAge = 12 * time.Hour

	// 将 CORS 中间件应用到路由器
	router.Use(cors.New(config))
	// ==================== CORS 配置结束 ====================

	// 使用日志中间件，记录所有请求
	router.Use(gin.Logger())

	// 连接到 MongoDB 数据库
	var client *mongo.Client = database.Connect()

	// 测试数据库连接是否成功
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to reach server: %v", err)
	}

	// 使用 defer 确保程序退出时断开数据库连接
	defer func() {
		err := client.Disconnect(context.Background())
		if err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// 设置不需要认证的路由（如：登录、注册）
	routes.SetupUnprotectedRoutes(router, client)

	// 设置需要认证的路由（如：获取用户信息、修改数据）
	routes.SetupProtectedRoutes(router, client)

	// 启动服务器，监听 8080 端口
	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
