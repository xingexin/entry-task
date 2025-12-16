package router

import (
	"entry-task/httpserver/internal/handler"
	"entry-task/httpserver/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(userHandler *handler.UserHandler) *gin.Engine {
	// 创建 Gin Engine（不使用默认中间件）
	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())                // Panic 恢复
	r.Use(middleware.CORSMiddleware())   // CORS
	r.Use(middleware.LoggerMiddleware()) // 日志

	// API 路由组
	api := r.Group("/api/v1")
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/login", userHandler.Login)
			auth.POST("/logout", userHandler.Logout)
		}

		// 用户信息相关
		profile := api.Group("/profile")
		{
			profile.GET("", userHandler.GetProfile)
			profile.PATCH("/nickname", userHandler.UpdateNickname)
			profile.POST("/picture", userHandler.UploadProfilePicture)
			profile.GET("/picture", userHandler.GetProfilePicture)
		}
	}

	return r
}
