package main

import (
	"context"
	"fiction-book/internal/kafka/outbox"
	"fiction-book/pkg/constant"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/api/router"
	"fiction-book/internal/wire"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/validator"
)

// @title fiction_book swagger
// @version 1.0
// @description fiction_book API文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name GNU
// @license.url https://opensource.org/license/gpl-3.0

// @host localhost:9929
// @BasePath /

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 初始化日志
	if err := logger.Init(cfg.Server.Mode); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting application...",
		zap.String("mode", cfg.Server.Mode),
		zap.Int("port", cfg.Server.Port),
	)

	// 初始化Kafka
	kafkaApp, err := wire.InitializeMassageQueue()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize kafka app: %v", err))
	}
	massageQueueStartContext, massageQueueCancel := context.WithCancel(context.Background())
	outbox.KafkaInit(massageQueueStartContext, cfg, kafkaApp)

	// 初始化 Sentry（如果启用）
	if cfg.Sentry.Enabled {
		err := middleware.InitSentry(middleware.SentryConfig{
			DSN:              cfg.Sentry.DSN,
			Environment:      cfg.Sentry.Environment,
			TracesSampleRate: cfg.Sentry.TracesSampleRate,
			Debug:            cfg.Sentry.Debug,
		})
		if err != nil {
			logger.Error("Failed to init Sentry", zap.Error(err))
		} else {
			logger.Info("Sentry initialized")
			defer sentry.Flush(2 * time.Second)
		}
	}

	// 初始化 OpenTelemetry（如果启用）
	if cfg.Tracing.Enabled {
		tp, err := middleware.InitTracing(middleware.TracingConfig{
			ServiceName:    cfg.Tracing.ServiceName,
			JaegerEndpoint: cfg.Tracing.JaegerEndpoint,
			Enabled:        cfg.Tracing.Enabled,
		})
		if err != nil {
			logger.Error("Failed to init tracing", zap.Error(err))
		} else if tp != nil {
			logger.Info("OpenTelemetry initialized")
			defer func() {
				if err := tp.Shutdown(context.Background()); err != nil {
					logger.Error("Error shutting down tracer provider", zap.Error(err))
				}
			}()
		}
	}

	// 初始化自定义验证器
	validator.Init()

	// 使用 Wire 初始化应用（自动注入所有依赖）
	app, err := wire.InitializeApp()
	if err != nil {
		logger.Fatal("Failed to initialize app", zap.Error(err))
	}

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	r := gin.New()
	router.Setup(r, app.Handler, cfg)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 在 goroutine 中启动服务
	go func() {
		logger.Info("Server is running",
			zap.String("addr", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	constant.DoneServer()
	logger.Info("Shutting down server...")

	// 设置 5 秒的超时时间用于优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭kafka消息队列
	massageQueueCancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
