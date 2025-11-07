package database

import (
	"log"
	"microvibe-go/internal/model"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	// 迁移所有模型
	err := db.AutoMigrate(
		// 用户相关
		&model.User{},
		&model.UserProfile{},
		&model.UserInterest{},

		// 视频相关
		&model.Video{},
		&model.Category{},
		&model.Hashtag{},
		&model.VideoHashtag{},
		&model.VideoStats{},

		// 社交相关
		&model.Comment{},
		&model.CommentLike{},
		&model.Like{},
		&model.Favorite{},
		&model.FavoriteFolder{},
		&model.Follow{},
		&model.Share{},

		// 消息相关
		&model.Message{},
		&model.Conversation{},
		&model.Notification{},

		// 直播相关（抖音风格完整功能）
		&model.LiveStream{},     // 直播间主表
		&model.LiveViewer{},     // 观众记录
		&model.LiveGift{},       // 礼物定义
		&model.LiveGiftRecord{}, // 礼物记录
		&model.LiveComment{},    // 弹幕评论
		&model.LiveProduct{},    // 直播商品
		&model.LiveAdmin{},      // 直播管理员
		&model.LiveBan{},        // 禁言记录
		&model.LiveShare{},      // 分享记录
		&model.LiveRankList{},   // 打赏榜
		&model.LiveFansClub{},   // 粉丝团

		// 行为相关
		&model.UserBehavior{},
		&model.SearchHistory{},
		&model.HotSearch{},
	)

	if err != nil {
		log.Printf("数据库迁移失败: %v", err)
		return err
	}

	// 创建索引
	createIndexes(db)

	log.Println("数据库迁移完成")
	return nil
}

// createIndexes 创建额外的索引
func createIndexes(db *gorm.DB) {
	log.Println("创建索引...")

	// ========== 社交相关索引 ==========
	// likes 表的组合唯一索引
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_video_like ON likes(user_id, video_id)")

	// comment_likes 表的组合唯一索引（防止重复点赞）
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_comment_like ON comment_likes(user_id, comment_id)")

	// favorites 表的组合唯一索引
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_video_favorite ON favorites(user_id, video_id)")

	// follows 表的组合唯一索引
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_followed ON follows(user_id, followed_id)")

	// video_hashtags 表的组合唯一索引
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_video_hashtag ON video_hashtags(video_id, hashtag_id)")

	// conversations 表的组合唯一索引
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user1_user2 ON conversations(user1_id, user2_id)")

	// ========== 行为相关索引 ==========
	// user_behaviors 表的性能索引
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_action_time ON user_behaviors(user_id, action, created_at)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_video_action_time ON user_behaviors(video_id, action, created_at)")

	// ========== 直播相关索引 ==========
	// live_streams 表的流媒体索引（用于 OBS 推流支持）
	db.Exec("CREATE INDEX IF NOT EXISTS idx_livestream_push_protocol ON live_streams(push_protocol)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_livestream_stream_type ON live_streams(stream_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_livestream_resolution ON live_streams(resolution)")

	log.Println("索引创建完成")
}

// SeedData 填充初始数据
func SeedData(db *gorm.DB) error {
	log.Println("开始填充初始数据...")

	// 检查是否已有分类数据
	var count int64
	db.Model(&model.Category{}).Count(&count)
	if count > 0 {
		log.Println("数据已存在，跳过填充")
		return nil
	}

	// 创建默认分类
	categories := []model.Category{
		{Name: "音乐", Description: "音乐视频", Icon: "🎵", Sort: 1, Status: 1},
		{Name: "舞蹈", Description: "舞蹈视频", Icon: "💃", Sort: 2, Status: 1},
		{Name: "美食", Description: "美食视频", Icon: "🍔", Sort: 3, Status: 1},
		{Name: "游戏", Description: "游戏视频", Icon: "🎮", Sort: 4, Status: 1},
		{Name: "搞笑", Description: "搞笑视频", Icon: "😂", Sort: 5, Status: 1},
		{Name: "运动", Description: "运动健身", Icon: "⚽", Sort: 6, Status: 1},
		{Name: "时尚", Description: "时尚穿搭", Icon: "👗", Sort: 7, Status: 1},
		{Name: "旅行", Description: "旅行风景", Icon: "✈️", Sort: 8, Status: 1},
		{Name: "科技", Description: "科技数码", Icon: "💻", Sort: 9, Status: 1},
		{Name: "教育", Description: "知识教育", Icon: "📚", Sort: 10, Status: 1},
	}

	if err := db.Create(&categories).Error; err != nil {
		log.Printf("创建分类失败: %v", err)
		return err
	}

	// 创建默认直播礼物
	gifts := []model.LiveGift{
		{Name: "点赞", Icon: "👍", Price: 1, Type: 1, Sort: 1, Status: 1},
		{Name: "玫瑰", Icon: "🌹", Price: 10, Type: 1, Sort: 2, Status: 1},
		{Name: "红心", Icon: "❤️", Price: 50, Type: 1, Sort: 3, Status: 1},
		{Name: "钻石", Icon: "💎", Price: 100, Type: 2, Sort: 4, Status: 1},
		{Name: "跑车", Icon: "🚗", Price: 500, Type: 2, Sort: 5, Status: 1},
		{Name: "火箭", Icon: "🚀", Price: 1000, Type: 2, Sort: 6, Status: 1},
	}

	if err := db.Create(&gifts).Error; err != nil {
		log.Printf("创建礼物失败: %v", err)
		return err
	}

	log.Println("初始数据填充完成")
	return nil
}
