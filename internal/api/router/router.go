package router

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"fiction-book/internal/api/handler"
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	_ "fiction-book/openapi" // swagger docs
	"fiction-book/pkg/config"
)

// Setup 设置路由
func Setup(r *gin.Engine, h *handler.Handler, cfg *config.Config) {
	// 全局中间件
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// 可选的 Sentry 中间件
	if cfg.Sentry.Enabled {
		r.Use(middleware.Sentry())
	}

	// 可选的 OpenTelemetry 中间件
	if cfg.Tracing.Enabled {
		r.Use(middleware.Tracing(cfg.Tracing.ServiceName))
	}

	// 可选的 Pprof 性能分析
	if cfg.Pprof.Enabled {
		middleware.Pprof(r)
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 静态文件服务
	r.Static("/upload", "upload")

	// 健康检查
	r.GET("/health", h.HealthCheck)

	// API 版本分组
	v1 := r.Group("/api/v1")
	{
		// 认证相关
		auth := v1.Group("/auth")
		{
			auth.POST("/login", middleware.Validation(&dto.LoginRequest{}), h.Login)
			auth.POST("/email/verify", middleware.Validation(&dto.EmailVerifyRequest{}), h.EmailVerify)
		}

		// 用户模块
		users := v1.Group("/users")
		{
			users.POST("", middleware.Validation(&dto.CreateUserRequest{}), h.CreateUser)
			users.GET("", middleware.Validation(&dto.ListUsersRequest{}), h.ListUsers)
			users.GET("/:id", middleware.Validation(&dto.GetUserRequest{}), h.GetUser)
			users.PUT("/:id", middleware.Auth(cfg), middleware.Validation(&dto.UpdateUserRequest{}), h.UpdateUser)
			users.DELETE("/:id", middleware.Auth(cfg), middleware.AdminOnly(), middleware.Validation(&dto.DeleteUserRequest{}), h.DeleteUser)
			users.POST("/:id/avatar", middleware.Auth(cfg), middleware.Validation(&dto.UploadAvatarRequest{}), h.UploadAvatar)
			users.PUT("/:id/password", middleware.Validation(&dto.UpdateUserPasswordRequest{}), h.UpdateUserPassword)
			users.PUT("/:id/email", middleware.Auth(cfg), middleware.Validation(&dto.UpdateUserEmailRequest{}), h.UpdateUserEmail)
		}

		// 书籍模块
		books := v1.Group("/books")
		{
			books.POST("/create", middleware.Validation(&dto.CreateBookRequest{}), h.CreateBook)
			books.GET("/:id", middleware.Validation(&dto.GetBookRequest{}), h.GetBook)
			books.GET("/list", middleware.Validation(&dto.ListBooksRequest{}), h.BookList)
		}
		// 书籍章节模块
		bookChapters := v1.Group("/chapters")
		{
			bookChapters.POST("/create", middleware.Validation(&dto.CreatBookChapterRequest{}), h.AddBookChapter)
			bookChapters.GET("/:id/guest", middleware.Validation(&dto.GetBookChapterRequest{}), h.GuestGetBookChapter)
		}
	}
}
