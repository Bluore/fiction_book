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
			users.PUT("/information", middleware.Auth(cfg), middleware.Validation(&dto.UpdateUserRequest{}), h.UpdateUser)
			users.DELETE("/delete", middleware.Auth(cfg), middleware.AdminOnly(), middleware.Validation(&dto.DeleteUserRequest{}), h.DeleteUser)
			users.POST("/avatar", middleware.Auth(cfg), middleware.Validation(&dto.UploadAvatarRequest{}), h.UploadAvatar)
			users.PUT("/password", middleware.Validation(&dto.UpdateUserPasswordRequest{}), h.UpdateUserPassword)
			users.PUT("/email", middleware.Auth(cfg), middleware.Validation(&dto.UpdateUserEmailRequest{}), h.UpdateUserEmail)
			users.GET("/info", middleware.Auth(cfg), h.GetMyUserInfo)
		}

		// 书籍模块
		books := v1.Group("/books")
		{
			books.POST("/create", middleware.Auth(cfg), middleware.Validation(&dto.CreateBookRequest{}), h.CreateBook)
			books.GET("/:id", middleware.Validation(&dto.GetBookRequest{}), h.GetBook)
			books.GET("/list", middleware.Validation(&dto.ListBooksRequest{}), h.BookList)
			books.POST("/:id/cover", middleware.Auth(cfg), middleware.Validation(&dto.UploadBookCoverRequest{}), h.UploadBookCoverImage)
			books.GET("/:id/chapters/list", middleware.Validation(&dto.GetBookChapterListRequest{}), h.GetBookChapterList)
		}

		// 书籍章节模块
		bookChapters := v1.Group("/chapters")
		{
			bookChapters.POST("/create", middleware.Auth(cfg), middleware.Validation(&dto.CreatBookChapterRequest{}), h.AddBookChapter)
			bookChapters.GET("/:id/guest", middleware.Validation(&dto.GetBookChapterRequest{}), h.GuestGetBookChapter)
			bookChapters.GET("/:id", middleware.Auth(cfg), middleware.Validation(&dto.GetBookChapterRequest{}), h.GetBookChapter)
			bookChapters.PUT("/:id", middleware.Auth(cfg), middleware.Validation(&dto.UpdateBookChapterRequest{}), h.UpdateBookChapter)
			bookChapters.DELETE("/:id", middleware.Auth(cfg), middleware.AdminOnly(), middleware.Validation(&dto.DeleteBookChapterRequest{}), h.DeleteBookChapter)
			bookChapters.GET("/thermal/daily", middleware.Validation(&dto.GetBookListByThermalDaily{}), h.GetBookListThermalDaily)
			bookChapters.GET("/:id/info", h.GetBookChapterInfo)
		}

		// 消费模块
		buys := v1.Group("/buy")
		{
			buys.POST("/book", middleware.Auth(cfg), middleware.Validation(&dto.BuyBookRequest{}), h.BuyBook)
			buys.POST("/chapter", middleware.Auth(cfg), middleware.Validation(&dto.BuyBookChapterRequest{}), h.BuyBookChapter)
		}

		// 资产管理
		assets := v1.Group("/assets")
		{
			assets.GET("", middleware.Auth(cfg), middleware.Validation(&dto.GetUserAssetRequest{}), h.GetUserAsset)
		}

		// 书籍购买管理
		purchases := v1.Group("/purchases/book")
		{
			purchases.GET("", middleware.Auth(cfg), middleware.Validation(&dto.GetUserPurchasedBookRequest{}), h.GetUserPurchasedBook)
			purchases.GET("/:book_id", middleware.Auth(cfg), middleware.Validation(&dto.CheckUserPurchasedBookRequest{}), h.CheckUserPurchasedBook)
		}

		// 章节购买管理
		chapterPurchases := v1.Group("/purchases/chapters")
		{
			chapterPurchases.GET("", middleware.Auth(cfg), middleware.Validation(&dto.GetUserPurchasedChapterRequest{}), h.GetUserPurchasedChapter)
			chapterPurchases.GET("/:chapter_id", middleware.Auth(cfg), middleware.Validation(&dto.CheckUserPurchasedChapterRequest{}), h.CheckUserPurchasedChapter)
		}

		// 书架管理
		bookshelf := v1.Group("/bookshelf")
		{
			bookshelf.GET("/:user_id", middleware.Validation(&dto.ListUserBookshelfRequest{}), h.ListUserBookshelf)
			bookshelf.POST("/add", middleware.Auth(cfg), middleware.Validation(&dto.CreateUserBookshelfRequest{}), h.CreateUserBookshelf)
		}
		//历史管理
		history := v1.Group("/history")
		{
			history.GET("/:user_id", middleware.Auth(cfg), middleware.Validation(&dto.ListUserHistoryRequest{}), h.ListUserHistory)
		}

		// 订单管理
		orders := v1.Group("/orders")
		{
			orders.GET("/:id", middleware.Auth(cfg), middleware.Validation(&dto.GetOrderRequest{}), h.GetOrderByID)
			orders.GET("", middleware.Auth(cfg), middleware.Validation(&dto.GetOrderListRequest{}), h.GetOrderList)
			orders.POST("/create/gold", middleware.Auth(cfg), middleware.Validation(&dto.CreateOrderRequest{}), h.CreateOrder)
		}
	}
}
