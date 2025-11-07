package main

import (
	"microvibe-go/internal/config"
	"microvibe-go/internal/database"
	"microvibe-go/internal/repository"
	"microvibe-go/internal/router"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/cache"
	"microvibe-go/pkg/event"
	"microvibe-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		panic("加载配置失败: " + err.Error())
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Server.Mode); err != nil {
		panic("初始化日志失败: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("应用启动", zap.String("mode", cfg.Server.Mode))

	// 初始化数据库连接
	db, err := database.InitPostgres(cfg)
	if err != nil {
		logger.Fatal("连接 PostgreSQL 失败", zap.Error(err))
	}
	logger.Info("PostgreSQL 连接成功")

	redisClient, err := database.InitRedis(cfg)
	if err != nil {
		logger.Fatal("连接 Redis 失败", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Redis 连接成功")

	// 初始化缓存系统
	if err := cache.InitCaches(cfg); err != nil {
		logger.Error("初始化缓存失败", zap.Error(err))
		// 缓存初始化失败不影响应用启动，降级到无缓存模式
	} else {
		logger.Info("缓存系统初始化成功")
	}
	defer cache.CloseCaches()

	// 初始化事件总线
	eventBus := event.GetGlobalEventBus()
	if err := eventBus.Start(); err != nil {
		logger.Fatal("启动事件总线失败", zap.Error(err))
	}
	defer eventBus.Stop()
	logger.Info("事件总线启动成功")

	// 注册直播事件处理器
	liveRepo := repository.NewLiveStreamRepository(db)
	liveEventHandler := service.NewLiveEventHandler(liveRepo)
	if err := liveEventHandler.RegisterHandlers(eventBus); err != nil {
		logger.Fatal("注册直播事件处理器失败", zap.Error(err))
	}
	logger.Info("直播事件处理器注册成功")

	// 设置 Gin 运行模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := router.Setup(db, redisClient, cfg)

	// 启动服务器
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	logger.Info("服务器启动", zap.String("address", addr))

	if err := r.Run(addr); err != nil {
		logger.Fatal("启动服务器失败", zap.Error(err))
	}
}
