package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"podcast/config"

	"github.com/youcd/toolkit/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func GetDb() *gorm.DB {
	return db
}

// NewDB 根据配置创建 *gorm.DB，供依赖注入使用（不设置全局 db）
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil || cfg.Database == nil || cfg.Database.PostgreSQL == nil {
		return nil, fmt.Errorf("database config is nil")
	}
	p := cfg.Database.PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.DBName)

	logLevel := "info"
	if cfg.Global != nil {
		logLevel = cfg.Global.LogLevel
	}

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: log.NewGormLogger(time.Second, logLevel),
	})
	if err != nil {
		// 如果数据库不存在，尝试创建
		log.WithCtx(context.Background()).Warnf("连接数据库失败，尝试创建数据库: %v", err)
		if createErr := createDatabaseIfNotExists(context.Background(), p); createErr != nil {
			return nil, fmt.Errorf("创建数据库失败: %w", createErr)
		}
		// 重新连接
		conn, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: log.NewGormLogger(time.Second, logLevel),
		})
		if err != nil {
			return nil, fmt.Errorf("open postgresql: %w", err)
		}
	}

	return conn, nil
}

// createDatabaseIfNotExists 如果数据库不存在则创建
func createDatabaseIfNotExists(ctx context.Context, cfg *config.PostgreSQL) error {
	// 连接到默认的 postgres 数据库
	defaultDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password)

	conn, err := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接默认数据库失败: %w", err)
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	defer sqlDB.Close()

	// 检查数据库是否存在
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = '%s')", cfg.DBName)
	if err := conn.Raw(query).Scan(&exists).Error; err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	if exists {
		log.WithCtx(ctx).Infof("数据库 %s 已存在", cfg.DBName)
		return nil
	}

	// 创建数据库
	createSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
	if err := conn.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}

	log.WithCtx(ctx).Infof("数据库 %s 创建成功", cfg.DBName)
	return nil
}

// Init 根据配置初始化全局 DB（兼容现有调用方）；返回该 DB 供注入使用
func Init(cfg *config.Config) (*gorm.DB, error) {
	var err error
	once.Do(func() {
		db, err = NewDB(cfg)
		if err != nil {
			log.WithCtx(context.Background()).Error(err)
			return
		}
	})
	return db, err
}

// InitializationTable 根据传入的 db 执行表迁移（供 DI 使用）
func InitializationTable(ctx context.Context, db *gorm.DB) {
	if db == nil {
		log.WithCtx(ctx).Error("db is nil")
		return
	}
	err := db.AutoMigrate(&RssContent{}, &Report{}, &KeyInfo{}, &User{}, &ChatSession{}, &Message{})
	if err != nil {
		log.WithCtx(ctx).Error(err)
	}
}
