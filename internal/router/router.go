package router

import (
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/config"
	"microvibe-go/internal/handler"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/repository"
	"microvibe-go/internal/service"
	"microvibe-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	r.Use(middleware.CORSMiddleware(cfg))

	// 安全响应头
	r.Use(middleware.SecurityHeaders())

	// 设备信息中间件
	r.Use(middleware.DeviceMiddleware())

	// Prometheus 指标采集
	r.Use(middleware.PrometheusMiddleware())

	// Prometheus 指标暴露
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查
	r.GET("/health", handler.HealthCheck(db, redisClient))

	// 静态文件服务
	r.Static("/uploads", "./uploads")

	// Token 黑名单
	tokenBlacklist := middleware.NewTokenBlacklist(redisClient)

	// 速率限制器
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimit.Enabled)

	// 初始化 Repository 层
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	commentRepo := repository.NewCommentRepository(db, redisClient)
	liveRepo := repository.NewLiveStreamRepository(db)
	banRepo := repository.NewLiveBanRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	hashtagRepo := repository.NewHashtagRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	blacklistRepo := repository.NewBlacklistRepository(db)
	reportRepo := repository.NewReportRepository(db)
	shareRepo := repository.NewShareRepository(db)
	videoStatsRepo := repository.NewVideoStatsRepository(db)
	videoHistoryRepo := repository.NewVideoHistoryRepository(db)
	userVisitorRepo := repository.NewUserVisitorRepository(db)
	behaviorRepo := repository.NewBehaviorRepository(db)

	// 初始化 Service 层
	userService := service.NewUserService(userRepo, followRepo, profileRepo)
	videoService := service.NewVideoService(videoRepo, likeRepo, favoriteRepo, followRepo, cfg)
	commentService := service.NewCommentService(commentRepo, videoRepo)
	liveService := service.NewLiveStreamService(liveRepo, banRepo, cfg)

	// 初始化 SFU 客户端服务（如果启用）
	var sfuClient service.SFUClientService
	if cfg.SFU.Enabled {
		sfuClient, _ = service.NewSFUClientService(&cfg.SFU)
	}

	// 初始化信令服务
	signalingService := service.NewLiveSignalingService(liveService, sfuClient, cfg.SFU.Enabled, cfg)

	searchService := service.NewSearchService(searchRepo, followRepo, likeRepo, favoriteRepo)
	messageService := service.NewMessageService(messageRepo, notificationRepo, userRepo, videoRepo)
	messageSignalingService := service.NewMessageSignalingService(cfg)

	// 注入信令服务到消息服务
	if ms, ok := messageService.(interface {
		SetSignalingService(service.MessageSignalingService)
	}); ok {
		ms.SetSignalingService(messageSignalingService)
	}

	hashtagService := service.NewHashtagService(hashtagRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	blacklistService := service.NewBlacklistService(blacklistRepo, userRepo)
	reportService := service.NewReportService(reportRepo, userRepo, videoRepo, commentRepo)
	shareService := service.NewShareService(shareRepo, videoRepo)
	videoHistoryService := service.NewVideoHistoryService(videoHistoryRepo, behaviorRepo, likeRepo, favoriteRepo, followRepo)
	videoStatsService := service.NewVideoStatsService(videoStatsRepo, videoRepo, followRepo)
	userVisitorService := service.NewUserVisitorService(userVisitorRepo, followRepo)
	adminService := service.NewAdminService(userRepo, videoRepo, commentRepo, searchRepo, reportRepo)

	// 推荐引擎
	recommendEngine := recommend.NewEngine(db, redisClient)

	// 后置注入依赖
	if vs, ok := videoService.(interface{ SetRecommendEngine(*recommend.Engine) }); ok {
		vs.SetRecommendEngine(recommendEngine)
	}
	if cs, ok := commentService.(interface{ SetRecommendEngine(*recommend.Engine) }); ok {
		cs.SetRecommendEngine(recommendEngine)
	}
	if hs, ok := videoHistoryService.(interface{ SetRecommendEngine(*recommend.Engine) }); ok {
		hs.SetRecommendEngine(recommendEngine)
	}
	if ss, ok := shareService.(interface{ SetRecommendEngine(*recommend.Engine) }); ok {
		ss.SetRecommendEngine(recommendEngine)
	}
	if ms, ok := messageService.(interface{ SetSignalingService(service.MessageSignalingService) }); ok {
		ms.SetSignalingService(messageSignalingService)
	}
	if vs, ok := videoService.(interface{ SetHashtagService(service.HashtagService) }); ok {
		vs.SetHashtagService(hashtagService)
	}
	if vs, ok := videoService.(interface{ SetMessageService(service.MessageService) }); ok {
		vs.SetMessageService(messageService)
	}
	if vs, ok := videoService.(interface{ SetStatsService(service.VideoStatsService) }); ok {
		vs.SetStatsService(videoStatsService)
	}
	if vs, ok := videoService.(interface{ SetUserRepo(repository.UserRepository) }); ok {
		vs.SetUserRepo(userRepo)
	}
	if cs, ok := commentService.(interface{ SetMessageService(service.MessageService) }); ok {
		cs.SetMessageService(messageService)
	}
	if cs, ok := commentService.(interface{ SetStatsService(service.VideoStatsService) }); ok {
		cs.SetStatsService(videoStatsService)
	}
	if hs, ok := videoHistoryService.(interface{ SetStatsService(service.VideoStatsService) }); ok {
		hs.SetStatsService(videoStatsService)
	}
	if ss, ok := shareService.(interface{ SetStatsService(service.VideoStatsService) }); ok {
		ss.SetStatsService(videoStatsService)
	}
	if us, ok := userService.(interface{ SetMessageService(service.MessageService) }); ok {
		us.SetMessageService(messageService)
	}

	// 初始化 Handler 层
	userHandler := handler.NewUserHandler(userService, userVisitorService, cfg, tokenBlacklist)
	adminHandler := handler.NewAdminHandler(adminService)
	videoHandler := handler.NewVideoHandler(recommendEngine, videoService)
	commentHandler := handler.NewCommentHandler(commentService)
	liveHandler := handler.NewLiveStreamHandler(liveService, cfg)
	searchHandler := handler.NewSearchHandler(searchService)
	messageHandler := handler.NewMessageHandler(messageService)
	hashtagHandler := handler.NewHashtagHandler(hashtagService, videoService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	blacklistHandler := handler.NewBlacklistHandler(blacklistService)
	reportHandler := handler.NewReportHandler(reportService)
	shareHandler := handler.NewShareHandler(shareService)
	videoHistoryHandler := handler.NewVideoHistoryHandler(videoHistoryService)
	videoStatsHandler := handler.NewVideoStatsHandler(videoStatsService)
	userVisitorHandler := handler.NewUserVisitorHandler(userVisitorService)
	fileHandler := handler.NewFileHandler(cfg)

	// OAuth Handler
	oauthHandler, err := handler.NewOAuthHandler(cfg, userService)
	if err != nil {
		logger.Error("初始化 OAuth 处理器失败", zap.Error(err))
	}

	// Auth 中间件简写
	auth := func() gin.HandlerFunc { return middleware.AuthMiddleware(cfg, tokenBlacklist) }
	optAuth := func() gin.HandlerFunc { return middleware.OptionalAuthMiddleware(cfg, tokenBlacklist) }

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// 认证相关（限流 + 公开）
		authGroup := v1.Group("/auth")
		authGroup.Use(rateLimiter.Middleware(middleware.AuthRateLimit()))
		{
			authGroup.POST("/register", userHandler.Register)
			authGroup.POST("/login", userHandler.Login)
			authGroup.POST("/logout", rateLimiter.Middleware(middleware.AuthRateLimit()), auth(), userHandler.Logout)
		}

		// 文件上传（限流 + 认证）
		upload := v1.Group("/upload")
		upload.Use(rateLimiter.Middleware(middleware.UploadRateLimit()), auth())
		{
			upload.POST("/image", fileHandler.UploadImage)
			upload.POST("/video", fileHandler.UploadVideo)
			upload.POST("/audio", fileHandler.UploadAudio)
		}

		if oauthHandler != nil {
			logger.Info("注册 OAuth2/OIDC 认证路由")
			oauth := v1.Group("/oauth")
			{
				oauth.GET("/login", oauthHandler.Login)
				oauth.GET("/callback", oauthHandler.Callback)
			}
		} else {
			logger.Warn("OAuth 未启用或初始化失败，跳过 OAuth 路由注册")
		}

		// 分类
		v1.GET("/categories", categoryHandler.GetCategories)

		// 用户相关
		users := v1.Group("/users")
		{
			users.GET("/:id", optAuth(), userHandler.GetUserInfo)
			users.GET("/:id/following", optAuth(), userHandler.GetUserFollowings)
			users.GET("/:id/followers", optAuth(), userHandler.GetUserFollowers)

			users.Use(auth())
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PUT("/me", userHandler.UpdateUserInfo)
				users.POST("/:id/follow", userHandler.Follow)
				users.DELETE("/:id/follow", userHandler.Unfollow)
				users.PUT("/me/privacy", userHandler.UpdatePrivacySettings)
				users.POST("/blacklist", blacklistHandler.BlockUser)
				users.DELETE("/blacklist/:id", blacklistHandler.UnblockUser)
				users.GET("/blacklist", blacklistHandler.GetBlacklist)
				users.GET("/me/history", videoHistoryHandler.GetHistory)
				users.DELETE("/me/history", videoHistoryHandler.ClearHistory)
				users.DELETE("/me/history/:id", videoHistoryHandler.DeleteHistory)
				users.GET("/me/visitors", userVisitorHandler.GetVisitors)
				users.GET("/me/visited", userVisitorHandler.GetVisited)
				users.GET("/me/videos", videoHandler.GetMyVideos)
				users.GET("/me/stats", videoStatsHandler.GetCreatorStats)
				users.GET("/me/stats/trending", videoStatsHandler.GetCreatorTrendingStats)
				users.GET("/me/comments/received", commentHandler.GetReceivedComments)
				users.GET("/me/comments/sent", commentHandler.GetSentComments)
			}
		}

		// 视频相关
		videos := v1.Group("/videos")
		{
			videos.GET("/feed", optAuth(), videoHandler.GetRecommendFeed)
			videos.GET("/hot", optAuth(), videoHandler.GetHotFeed)
			videos.GET("/:id", optAuth(), videoHandler.GetVideoDetail)
			videos.GET("/:id/likers", videoHandler.GetVideoLikers)
			videos.GET("/:id/favoriters", videoHandler.GetVideoFavoriters)

			authenticated := videos.Group("")
			authenticated.Use(auth())
			{
				authenticated.POST("", videoHandler.CreateVideo)
				authenticated.POST("/upload", videoHandler.UploadVideo)
				authenticated.PUT("/:id", videoHandler.UpdateVideo)
				authenticated.DELETE("/:id", videoHandler.DeleteVideo)
				authenticated.POST("/:id/audit", videoHandler.AuditVideo)
				authenticated.GET("/follow", videoHandler.GetFollowFeed)
				authenticated.GET("/friends", videoHandler.GetFriendsFeed)
				authenticated.POST("/:id/history", videoHistoryHandler.ReportProgress)
				authenticated.POST("/:id/like", videoHandler.LikeVideo)
				authenticated.DELETE("/:id/like", videoHandler.UnlikeVideo)
				authenticated.POST("/:id/favorite", videoHandler.FavoriteVideo)
				authenticated.DELETE("/:id/favorite", videoHandler.UnfavoriteVideo)
				authenticated.GET("/:id/stats", videoStatsHandler.GetVideoStats)
				authenticated.GET("/:id/stats/daily", videoStatsHandler.GetVideoDailyStats)
				authenticated.POST("/:id/share", shareHandler.ShareVideo)
			}
			videos.GET("/:id/share/count", shareHandler.GetShareCount)
			videos.POST("/:id/play", videoHandler.RecordPlay)
		}

		v1.GET("/users/:id/videos", optAuth(), videoHandler.GetUserVideos)
		v1.GET("/users/:id/favorites", optAuth(), videoHandler.GetUserFavorites)
		v1.GET("/users/:id/likes", optAuth(), videoHandler.GetUserLikes)

		// 评论
		comments := v1.Group("/comments")
		{
			comments.GET("/video/:video_id", commentHandler.GetVideoComments)
			comments.GET("/:id", commentHandler.GetCommentDetail)
			comments.GET("/:id/replies", commentHandler.GetReplies)

			authenticated := comments.Group("")
			authenticated.Use(auth())
			{
				authenticated.POST("", commentHandler.CreateComment)
				authenticated.DELETE("/:id", commentHandler.DeleteComment)
				authenticated.POST("/:id/like", commentHandler.LikeComment)
				authenticated.DELETE("/:id/like", commentHandler.UnlikeComment)
				authenticated.POST("/:id/pin", commentHandler.TogglePinComment)
			}
		}

		// 直播
		live := v1.Group("/live")
		{
			live.GET("/list", liveHandler.ListLiveStreams)
			live.GET("/:id", liveHandler.GetLiveStream)
			live.GET("/room/:room_id", liveHandler.GetLiveStreamByRoomID)
			live.POST("/join/:room_id", liveHandler.JoinLiveStream)
			live.POST("/leave/:room_id", liveHandler.LeaveLiveStream)
			live.GET("/ws", signalingService.HandleWebSocket)

			authenticated := live.Group("")
			authenticated.Use(auth())
			{
				authenticated.POST("/create", liveHandler.CreateLiveStream)
				authenticated.POST("/start", liveHandler.StartLiveStream)
				authenticated.POST("/end", liveHandler.EndLiveStream)
				authenticated.GET("/my", liveHandler.GetMyLiveStream)
				authenticated.DELETE("/:id", liveHandler.DeleteLiveStream)
				authenticated.POST("/:id/like", liveHandler.IncrementLike)
			}
		}

		// 搜索（限流）
		search := v1.Group("/search")
		search.Use(rateLimiter.Middleware(middleware.SearchRateLimit()))
		{
			search.GET("", optAuth(), searchHandler.Search)
			search.GET("/videos", optAuth(), searchHandler.SearchVideos)
			search.GET("/users", optAuth(), searchHandler.SearchUsers)
			search.GET("/hashtags", searchHandler.SearchHashtags)
			search.GET("/hot", searchHandler.GetHotSearches)
			search.GET("/suggestions", searchHandler.GetSearchSuggestions)

			authenticated := search.Group("")
			authenticated.Use(auth())
			{
				authenticated.GET("/history", searchHandler.GetSearchHistory)
				authenticated.DELETE("/history", searchHandler.ClearSearchHistory)
			}
		}

		// 消息
		messages := v1.Group("/messages")
		{
			messages.GET("/ws", messageSignalingService.HandleWebSocket)

			authenticated := messages.Group("")
			authenticated.Use(auth())
			{
				authenticated.GET("/conversations", messageHandler.GetConversationList)
				authenticated.POST("/conversations", messageHandler.CreateConversation)
				authenticated.GET("/conversations/:id", messageHandler.GetConversationMessages)
				authenticated.POST("/send", messageHandler.SendMessage)
				authenticated.POST("/conversations/:id/read", messageHandler.MarkConversationAsRead)
				authenticated.POST("/:id/read", messageHandler.MarkAsRead)
				authenticated.DELETE("/:id", messageHandler.DeleteMessage)
				authenticated.GET("/unread/count", messageHandler.GetUnreadMessageCount)
			}
		}

		// 通知
		notifications := v1.Group("/notifications")
		notifications.Use(auth())
		{
			notifications.GET("", messageHandler.GetNotificationList)
			notifications.POST("/:id/read", messageHandler.MarkNotificationAsRead)
			notifications.POST("/read-all", messageHandler.MarkAllNotificationsAsRead)
			notifications.GET("/unread/count", messageHandler.GetUnreadNotificationCount)
		}

		// 话题
		hashtags := v1.Group("/hashtags")
		{
			hashtags.GET("/hot", hashtagHandler.GetHotHashtags)
			hashtags.GET("/:id", hashtagHandler.GetHashtagDetail)
			hashtags.GET("/:id/videos", optAuth(), hashtagHandler.GetHashtagVideos)

			authenticated := hashtags.Group("")
			authenticated.Use(auth())
			{
				authenticated.POST("", hashtagHandler.CreateHashtag)
			}
		}

		// 举报
		reports := v1.Group("/reports")
		reports.Use(auth())
		{
			reports.POST("", reportHandler.CreateReport)
			reports.GET("", reportHandler.GetMyReports)
		}
	}

	// Admin 管理路由
	admin := v1.Group("/admin")
	admin.Use(auth(), middleware.AdminMiddleware())
	{
		admin.GET("/videos", adminHandler.ListVideos)
		admin.POST("/videos/:id/audit", adminHandler.AuditVideo)
		admin.POST("/comments/:id/audit", adminHandler.AuditComment)
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users/:id/status", adminHandler.UpdateUserStatus)
		admin.POST("/videos/:id/top", adminHandler.SetVideoTop)
		admin.POST("/videos/:id/weight", adminHandler.UpdateVideoWeight)
		admin.GET("/search/hot", adminHandler.ListHotSearches)
		admin.DELETE("/search/hot", adminHandler.DeleteHotSearch)
		admin.POST("/search/hot/weight", adminHandler.UpdateHotSearchWeight)
		admin.GET("/reports", adminHandler.ListReports)
	}

	return r
}
