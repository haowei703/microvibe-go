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

	// 添加 CORS 中间件（必须在所有路由之前）
	r.Use(middleware.CORSMiddleware())

	// 添加设备信息中间件
	r.Use(middleware.DeviceMiddleware())

	// Prometheus 指标采集
	r.Use(middleware.PrometheusMiddleware())

	// Prometheus 指标暴露
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查
	r.GET("/health", handler.HealthCheck(db, redisClient))

	// 静态文件服务 (用于访问上传的视频、封面和HLS流)
	r.Static("/uploads", "./uploads")

	// 初始化 Repository 层（内置装饰器缓存）
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

	// 初始化信令服务（传入 SFU 客户端、启用状态和配置）
	signalingService := service.NewLiveSignalingService(liveService, sfuClient, cfg.SFU.Enabled, cfg)

	searchService := service.NewSearchService(searchRepo, followRepo, likeRepo, favoriteRepo)
	messageService := service.NewMessageService(messageRepo, notificationRepo, userRepo, videoRepo)
	messageSignalingService := service.NewMessageSignalingService(cfg)

	// 注入信令服务到消息服务（启用实时推送）
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

	// 初始化推荐引擎
	recommendEngine := recommend.NewEngine(db, redisClient)

	// 后置注入依赖，避免循环引用
	if vs, ok := videoService.(interface {
		SetRecommendEngine(*recommend.Engine)
	}); ok {
		vs.SetRecommendEngine(recommendEngine)
	}
	if cs, ok := commentService.(interface {
		SetRecommendEngine(*recommend.Engine)
	}); ok {
		cs.SetRecommendEngine(recommendEngine)
	}
	if hs, ok := videoHistoryService.(interface {
		SetRecommendEngine(*recommend.Engine)
	}); ok {
		hs.SetRecommendEngine(recommendEngine)
	}
	if ss, ok := shareService.(interface {
		SetRecommendEngine(*recommend.Engine)
	}); ok {
		ss.SetRecommendEngine(recommendEngine)
	}

	if ms, ok := messageService.(interface {
		SetSignalingService(service.MessageSignalingService)
	}); ok {
		ms.SetSignalingService(messageSignalingService)
	}
	if vs, ok := videoService.(interface{ SetHashtagService(service.HashtagService) }); ok {
		vs.SetHashtagService(hashtagService)
	}
	if vs, ok := videoService.(interface{ SetMessageService(service.MessageService) }); ok {
		vs.SetMessageService(messageService)
	}
	if vs, ok := videoService.(interface {
		SetStatsService(service.VideoStatsService)
	}); ok {
		vs.SetStatsService(videoStatsService)
	}
	if vs, ok := videoService.(interface {
		SetUserRepo(repository.UserRepository)
	}); ok {
		vs.SetUserRepo(userRepo)
	}
	if cs, ok := commentService.(interface{ SetMessageService(service.MessageService) }); ok {
		cs.SetMessageService(messageService)
	}
	if cs, ok := commentService.(interface {
		SetStatsService(service.VideoStatsService)
	}); ok {
		cs.SetStatsService(videoStatsService)
	}
	if hs, ok := videoHistoryService.(interface {
		SetStatsService(service.VideoStatsService)
	}); ok {
		hs.SetStatsService(videoStatsService)
	}
	if ss, ok := shareService.(interface {
		SetStatsService(service.VideoStatsService)
	}); ok {
		ss.SetStatsService(videoStatsService)
	}
	if us, ok := userService.(interface{ SetMessageService(service.MessageService) }); ok {
		us.SetMessageService(messageService)
	}

	// 初始化 Handler 层
	userHandler := handler.NewUserHandler(userService, userVisitorService, cfg)
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

	// OAuth Handler（可选）
	oauthHandler, err := handler.NewOAuthHandler(cfg, userService)
	if err != nil {
		logger.Error("初始化 OAuth 处理器失败", zap.Error(err))
	}

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 公开接口
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// 认证相关（无需登录）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		// 文件上传相关
		upload := v1.Group("/upload")
		upload.Use(middleware.AuthMiddleware(cfg))
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

		// 分类相关
		v1.GET("/categories", categoryHandler.GetCategories)

		// 用户相关
		users := v1.Group("/users")
		{
			// 公开接口（支持可选登录，用于记录访客）
			users.GET("/:id", middleware.OptionalAuthMiddleware(cfg), userHandler.GetUserInfo)

			// 关注/粉丝列表（支持游客，游客 is_followed 始终为 false）
			users.GET("/:id/following", middleware.OptionalAuthMiddleware(cfg), userHandler.GetUserFollowings)
			users.GET("/:id/followers", middleware.OptionalAuthMiddleware(cfg), userHandler.GetUserFollowers)

			// 需要登录的接口
			users.Use(middleware.AuthMiddleware(cfg))
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PUT("/me", userHandler.UpdateUserInfo)
				users.POST("/:id/follow", userHandler.Follow)
				users.DELETE("/:id/follow", userHandler.Unfollow)
				users.PUT("/me/privacy", userHandler.UpdatePrivacySettings)

				// 黑名单相关
				users.POST("/blacklist", blacklistHandler.BlockUser)
				users.DELETE("/blacklist/:id", blacklistHandler.UnblockUser)
				users.GET("/blacklist", blacklistHandler.GetBlacklist)

				// 播放历史相关
				users.GET("/me/history", videoHistoryHandler.GetHistory)
				users.DELETE("/me/history", videoHistoryHandler.ClearHistory)
				users.DELETE("/me/history/:id", videoHistoryHandler.DeleteHistory)

				// 访客相关
				users.GET("/me/visitors", userVisitorHandler.GetVisitors)
				users.GET("/me/visited", userVisitorHandler.GetVisited)

				// 用户作品相关 (包含非公开作品)
				users.GET("/me/videos", videoHandler.GetMyVideos)

				// 创作者统计
				users.GET("/me/stats", videoStatsHandler.GetCreatorStats)
				users.GET("/me/stats/trending", videoStatsHandler.GetCreatorTrendingStats)

				// 收到的评论聚合
				users.GET("/me/comments/received", commentHandler.GetReceivedComments)
				users.GET("/me/comments/sent", commentHandler.GetSentComments)
			}
		}

		// 视频相关
		videos := v1.Group("/videos")
		{
			// 推荐流（支持游客和登录用户）
			videos.GET("/feed", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetRecommendFeed)
			videos.GET("/hot", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetHotFeed)

			// 视频详情（公开）
			videos.GET("/:id", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetVideoDetail)
			videos.GET("/:id/likers", videoHandler.GetVideoLikers)
			videos.GET("/:id/favoriters", videoHandler.GetVideoFavoriters)

			// 需要登录的接口
			authenticated := videos.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				// 上传和管理
				authenticated.POST("", videoHandler.CreateVideo) // 兼容老接口
				authenticated.POST("/upload", videoHandler.UploadVideo)
				authenticated.PUT("/:id", videoHandler.UpdateVideo)
				authenticated.DELETE("/:id", videoHandler.DeleteVideo)

				// 审核 (理论上应该是管理员权限，这里简化)
				authenticated.POST("/:id/audit", videoHandler.AuditVideo)

				// 关注流
				authenticated.GET("/follow", videoHandler.GetFollowFeed)
				// 朋友流 (双向关注)
				authenticated.GET("/friends", videoHandler.GetFriendsFeed)

				// 播放历史记录
				authenticated.POST("/:id/history", videoHistoryHandler.ReportProgress)

				// 互动操作
				authenticated.POST("/:id/like", videoHandler.LikeVideo)
				authenticated.DELETE("/:id/like", videoHandler.UnlikeVideo)
				authenticated.POST("/:id/favorite", videoHandler.FavoriteVideo)
				authenticated.DELETE("/:id/favorite", videoHandler.UnfavoriteVideo)

				// 视频统计（创作者数据）
				authenticated.GET("/:id/stats", videoStatsHandler.GetVideoStats)
				authenticated.GET("/:id/stats/daily", videoStatsHandler.GetVideoDailyStats)

				// 分享操作
				authenticated.POST("/:id/share", shareHandler.ShareVideo)
			}
			// 获取分享数（公开）
			videos.GET("/:id/share/count", shareHandler.GetShareCount)
			// 记录播放（公开，无需登录）
			videos.POST("/:id/play", videoHandler.RecordPlay)
		}

		// 用户视频列表（需要登录）
		v1.GET("/users/:id/videos", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetUserVideos)
		v1.GET("/users/:id/favorites", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetUserFavorites)
		v1.GET("/users/:id/likes", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetUserLikes)

		// 评论相关
		comments := v1.Group("/comments")
		{
			// 获取视频评论列表（公开）
			comments.GET("/video/:video_id", commentHandler.GetVideoComments)

			// 获取评论详情（公开）
			comments.GET("/:id", commentHandler.GetCommentDetail)

			// 获取评论回复列表（公开）
			comments.GET("/:id/replies", commentHandler.GetReplies)

			// 需要登录的接口
			authenticated := comments.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				// 创建评论
				authenticated.POST("", commentHandler.CreateComment)

				// 删除评论
				authenticated.DELETE("/:id", commentHandler.DeleteComment)

				// 点赞评论
				authenticated.POST("/:id/like", commentHandler.LikeComment)

				// 取消点赞评论
				authenticated.DELETE("/:id/like", commentHandler.UnlikeComment)

				// 置顶评论 (创作者管理)
				authenticated.POST("/:id/pin", commentHandler.TogglePinComment)
			}
		}

		// 直播相关
		live := v1.Group("/live")
		{
			// 公开接口
			live.GET("/list", liveHandler.ListLiveStreams)                // 直播列表
			live.GET("/:id", liveHandler.GetLiveStream)                   // 直播详情
			live.GET("/room/:room_id", liveHandler.GetLiveStreamByRoomID) // 根据房间ID获取直播
			live.POST("/join/:room_id", liveHandler.JoinLiveStream)       // 加入直播间（统计用，WebSocket层已鉴权）
			live.POST("/leave/:room_id", liveHandler.LeaveLiveStream)     // 离开直播间（统计用，WebSocket层已鉴权）

			// WebSocket 信令服务器（内部处理 token 鉴权）
			live.GET("/ws", signalingService.HandleWebSocket)

			// 需要登录的接口
			authenticated := live.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				// 直播间管理
				authenticated.POST("/create", liveHandler.CreateLiveStream) // 创建直播间
				authenticated.POST("/start", liveHandler.StartLiveStream)   // 开始直播
				authenticated.POST("/end", liveHandler.EndLiveStream)       // 结束直播
				authenticated.GET("/my", liveHandler.GetMyLiveStream)       // 我的直播间
				authenticated.DELETE("/:id", liveHandler.DeleteLiveStream)  // 删除直播间
				authenticated.POST("/:id/like", liveHandler.IncrementLike)  // 点赞
			}
		}

		// 搜索相关
		search := v1.Group("/search")
		{
			// 公开接口
			search.GET("", middleware.OptionalAuthMiddleware(cfg), searchHandler.Search)              // 综合搜索
			search.GET("/videos", middleware.OptionalAuthMiddleware(cfg), searchHandler.SearchVideos) // 搜索视频
			search.GET("/users", middleware.OptionalAuthMiddleware(cfg), searchHandler.SearchUsers)   // 搜索用户
			search.GET("/hashtags", searchHandler.SearchHashtags)                                     // 搜索话题
			search.GET("/hot", searchHandler.GetHotSearches)                                          // 热搜榜
			search.GET("/suggestions", searchHandler.GetSearchSuggestions)                            // 搜索建议

			// 需要登录的接口
			authenticated := search.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				authenticated.GET("/history", searchHandler.GetSearchHistory)      // 搜索历史
				authenticated.DELETE("/history", searchHandler.ClearSearchHistory) // 清空搜索历史
			}
		}

		// 消息相关
		messages := v1.Group("/messages")
		{
			// WebSocket 实时消息（内部处理 token 鉴权）
			messages.GET("/ws", messageSignalingService.HandleWebSocket)

			// 需要登录的 REST 接口
			authenticated := messages.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				authenticated.GET("/conversations", messageHandler.GetConversationList)              // 会话列表
				authenticated.POST("/conversations", messageHandler.CreateConversation)              // 创建/获取会话
				authenticated.GET("/conversations/:id", messageHandler.GetConversationMessages)      // 根据会话ID获取消息
				authenticated.POST("/send", messageHandler.SendMessage)                              // 发送消息
				authenticated.POST("/conversations/:id/read", messageHandler.MarkConversationAsRead) // 标记会话已读
				authenticated.POST("/:id/read", messageHandler.MarkAsRead)                           // 标记消息已读
				authenticated.DELETE("/:id", messageHandler.DeleteMessage)                           // 删除消息
				authenticated.GET("/unread/count", messageHandler.GetUnreadMessageCount)             // 未读消息数
			}
		}

		// 通知相关
		notifications := v1.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(cfg))
		{
			notifications.GET("", messageHandler.GetNotificationList)                     // 通知列表
			notifications.POST("/:id/read", messageHandler.MarkNotificationAsRead)        // 标记通知已读
			notifications.POST("/read-all", messageHandler.MarkAllNotificationsAsRead)    // 标记所有通知已读
			notifications.GET("/unread/count", messageHandler.GetUnreadNotificationCount) // 未读通知数
		}

		// 话题标签相关
		hashtags := v1.Group("/hashtags")
		{
			// 公开接口
			hashtags.GET("/hot", hashtagHandler.GetHotHashtags)                                                  // 热门话题
			hashtags.GET("/:id", hashtagHandler.GetHashtagDetail)                                                // 话题详情
			hashtags.GET("/:id/videos", middleware.OptionalAuthMiddleware(cfg), hashtagHandler.GetHashtagVideos) // 话题视频列表

			// 需要登录的接口
			authenticated := hashtags.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				authenticated.POST("", hashtagHandler.CreateHashtag) // 创建话题
			}
		}

		// 举报相关
		reports := v1.Group("/reports")
		reports.Use(middleware.AuthMiddleware(cfg))
		{
			reports.POST("", reportHandler.CreateReport)
			reports.GET("", reportHandler.GetMyReports)
		}
	}

	// Admin 管理路由
	admin := v1.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg), middleware.AdminMiddleware())
	{
		// 内容审核
		admin.GET("/videos", adminHandler.ListVideos)
		admin.POST("/videos/:id/audit", adminHandler.AuditVideo)
		admin.POST("/comments/:id/audit", adminHandler.AuditComment)

		// 用户管理
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users/:id/status", adminHandler.UpdateUserStatus)

		// 推荐干预
		admin.POST("/videos/:id/top", adminHandler.SetVideoTop)
		admin.POST("/videos/:id/weight", adminHandler.UpdateVideoWeight)

		// 搜索维护
		admin.GET("/search/hot", adminHandler.ListHotSearches)
		admin.DELETE("/search/hot", adminHandler.DeleteHotSearch)
		admin.POST("/search/hot/weight", adminHandler.UpdateHotSearchWeight)

		// 举报管理
		admin.GET("/reports", adminHandler.ListReports)
	}

	return r
}
