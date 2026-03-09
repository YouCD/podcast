package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"podcast/config"

	"github.com/youcd/toolkit/log"
	"gorm.io/driver/mysql"
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
	if cfg == nil || cfg.Database == nil || cfg.Database.Mysql == nil {
		return nil, fmt.Errorf("database config is nil")
	}
	m := cfg.Database.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local", m.User, m.Password, m.Host, m.Port)
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 ", m.DBName)
	if err := conn.Exec(sql).Error; err != nil {
		return nil, fmt.Errorf("create database: %w", err)
	}
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", m.User, m.Password, m.Host, m.Port, m.DBName)
	logLevel := "info"
	if cfg.Global != nil {
		logLevel = cfg.Global.LogLevel
	}
	conn, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: log.NewGormLogger(time.Second, logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql db: %w", err)
	}
	return conn, nil
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
	err := db.Set("gorm:table_options", "CHARSET=utf8mb4").AutoMigrate(&RssContent{}, &Report{}, &KeyInfo{}, &User{}, &ChatSession{}, &Message{})
	if err != nil {
		log.WithCtx(ctx).Error(err)
	}
}
