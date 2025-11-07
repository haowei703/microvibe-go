package main

import (
	"log"
	"microvibe-go/internal/config"
	"microvibe-go/internal/database"
)

func main() {
	log.Println("=== 数据库迁移工具 ===")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := database.InitPostgres(cfg)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 执行迁移
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 填充初始数据
	if err := database.SeedData(db); err != nil {
		log.Fatalf("填充初始数据失败: %v", err)
	}

	log.Println("=== 迁移完成 ===")
}
