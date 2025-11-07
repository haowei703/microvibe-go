package router

import (
	"microvibe-go/internal/algorithm/recommend"
	"microvibe-go/internal/config"
	"microvibe-go/internal/handler"
	"microvibe-go/internal/middleware"
	"microvibe-go/internal/repository"
	"microvibe-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// 添加 CORS 中间件（必须在所有路由之前）
	r.Use(middleware.CORSMiddleware())

	// 健康检查
	r.GET("/health", handler.HealthCheck(db, redisClient))

	// 初始化 Repository 层（内置装饰器缓存）
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	liveRepo := repository.NewLiveStreamRepository(db)
	banRepo := repository.NewLiveBanRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	hashtagRepo := repository.NewHashtagRepository(db)

	// 初始化 Service 层
	userService := service.NewUserService(userRepo, followRepo, profileRepo)
	videoService := service.NewVideoService(videoRepo, likeRepo, favoriteRepo)
	commentService := service.NewCommentService(commentRepo, videoRepo)
	liveService := service.NewLiveStreamService(liveRepo, banRepo, cfg)

	// 初始化 SFU 客户端服务（如果启用）
	var sfuClient service.SFUClientService
	if cfg.SFU.Enabled {
		sfuClient, _ = service.NewSFUClientService(&cfg.SFU)
	}

	// 初始化信令服务（传入 SFU 客户端和启用状态）
	signalingService := service.NewLiveSignalingService(liveService, sfuClient, cfg.SFU.Enabled)

	searchService := service.NewSearchService(searchRepo)
	messageService := service.NewMessageService(messageRepo, notificationRepo, userRepo)
	hashtagService := service.NewHashtagService(hashtagRepo)

	// 初始化推荐引擎
	recommendEngine := recommend.NewEngine(db, redisClient)

	// 初始化 Handler 层
	userHandler := handler.NewUserHandler(userService, cfg)
	videoHandler := handler.NewVideoHandler(recommendEngine, videoService)
	commentHandler := handler.NewCommentHandler(commentService)
	liveHandler := handler.NewLiveStreamHandler(liveService, cfg)
	searchHandler := handler.NewSearchHandler(searchService)
	messageHandler := handler.NewMessageHandler(messageService)
	hashtagHandler := handler.NewHashtagHandler(hashtagService)

	// OAuth Handler（可选）
	oauthHandler, _ := handler.NewOAuthHandler(cfg, userService)

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

		// OAuth 认证（如果启用）
		if oauthHandler != nil {
			oauth := v1.Group("/oauth")
			{
				oauth.GET("/login", oauthHandler.Login)
				oauth.GET("/callback", oauthHandler.Callback)
			}
		}

		// 用户相关
		users := v1.Group("/users")
		{
			// 公开接口
			users.GET("/:id", userHandler.GetUserInfo)

			// 需要登录的接口
			users.Use(middleware.AuthMiddleware(cfg))
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PUT("/me", userHandler.UpdateUserInfo)
				users.POST("/:id/follow", userHandler.Follow)
				users.DELETE("/:id/follow", userHandler.Unfollow)
			}
		}

		// 视频相关
		videos := v1.Group("/videos")
		{
			// 推荐流（支持游客和登录用户）
			videos.GET("/feed", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetRecommendFeed)
			videos.GET("/hot", middleware.OptionalAuthMiddleware(cfg), videoHandler.GetHotFeed)

			// 视频详情（公开）
			videos.GET("/:id", videoHandler.GetVideoDetail)

			// 需要登录的接口
			authenticated := videos.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				// 上传和管理
				authenticated.POST("", videoHandler.CreateVideo)
				authenticated.PUT("/:id", videoHandler.UpdateVideo)
				authenticated.DELETE("/:id", videoHandler.DeleteVideo)

				// 关注流
				authenticated.GET("/follow", videoHandler.GetFollowFeed)

				// 互动操作
				authenticated.POST("/:id/like", videoHandler.LikeVideo)
				authenticated.DELETE("/:id/like", videoHandler.UnlikeVideo)
				authenticated.POST("/:id/favorite", videoHandler.FavoriteVideo)
				authenticated.DELETE("/:id/favorite", videoHandler.UnfavoriteVideo)
			}
		}

		// 用户视频列表（公开）
		v1.GET("/users/:id/videos", videoHandler.GetUserVideos)

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
			}
		}

		// 直播相关
		live := v1.Group("/live")
		{
			// 公开接口
			live.GET("/list", liveHandler.ListLiveStreams)                // 直播列表
			live.GET("/:id", liveHandler.GetLiveStream)                   // 直播详情
			live.GET("/room/:room_id", liveHandler.GetLiveStreamByRoomID) // 根据房间ID获取直播
			live.POST("/:id/like", liveHandler.IncrementLike)             // 点赞（可选是否需要登录）
			live.POST("/join/:room_id", liveHandler.JoinLiveStream)       // 加入直播间（统计用，WebSocket层已鉴权）
			live.POST("/leave/:room_id", liveHandler.LeaveLiveStream)     // 离开直播间（统计用，WebSocket层已鉴权）

			// WebSocket 信令服务器（需要登录）
			live.GET("/ws", middleware.OptionalAuthMiddleware(cfg), signalingService.HandleWebSocket)

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
			}
		}

		// 搜索相关
		search := v1.Group("/search")
		{
			// 公开接口
			search.GET("", middleware.OptionalAuthMiddleware(cfg), searchHandler.Search) // 综合搜索
			search.GET("/videos", searchHandler.SearchVideos)                            // 搜索视频
			search.GET("/users", searchHandler.SearchUsers)                              // 搜索用户
			search.GET("/hashtags", searchHandler.SearchHashtags)                        // 搜索话题
			search.GET("/hot", searchHandler.GetHotSearches)                             // 热搜榜
			search.GET("/suggestions", searchHandler.GetSearchSuggestions)               // 搜索建议

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
		messages.Use(middleware.AuthMiddleware(cfg))
		{
			messages.GET("/conversations", messageHandler.GetConversationList)                   // 会话列表
			messages.GET("/conversations/:user_id", messageHandler.GetConversationMessages)      // 会话消息
			messages.POST("/send", messageHandler.SendMessage)                                   // 发送消息
			messages.POST("/conversations/:user_id/read", messageHandler.MarkConversationAsRead) // 标记会话已读
			messages.POST("/:id/read", messageHandler.MarkAsRead)                                // 标记消息已读
			messages.DELETE("/:id", messageHandler.DeleteMessage)                                // 删除消息
			messages.GET("/unread/count", messageHandler.GetUnreadMessageCount)                  // 未读消息数
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
			hashtags.GET("/hot", hashtagHandler.GetHotHashtags)          // 热门话题
			hashtags.GET("/:id", hashtagHandler.GetHashtagDetail)        // 话题详情
			hashtags.GET("/:id/videos", hashtagHandler.GetHashtagVideos) // 话题视频列表

			// 需要登录的接口
			authenticated := hashtags.Group("")
			authenticated.Use(middleware.AuthMiddleware(cfg))
			{
				authenticated.POST("", hashtagHandler.CreateHashtag) // 创建话题
			}
		}
	}

	return r
}
