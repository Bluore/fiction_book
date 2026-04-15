package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"fiction-book/internal/model"
	"fiction-book/pkg/config"
)

// InitDB 初始化数据库连接
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	var logLevel logger.LogLevel
	if cfg.Server.Mode == "release" {
		logLevel = logger.Error
	} else {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// 自动迁移
	if err := db.AutoMigrate(&model.User{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Book{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.BookChapter{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.UserAsset{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.UserPurchasedBook{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.UserPurchasedChapter{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.UserBookshelf{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.UserHistory{}); err != nil {
		return nil, err
	}

	return db, nil
}
