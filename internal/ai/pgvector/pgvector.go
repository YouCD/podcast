package pgvector

import (
	"context"
	"fmt"
	"time"

	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PgVector PostgreSQL + pgvector 向量库封装
type PgVector struct {
	db        *gorm.DB
	score     float32
	dimension int
}

// NewPgVector 创建 PgVector 客户端
func NewPgVector(ctx context.Context, cfg *types.PgVector) *PgVector {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// 如果数据库不存在，尝试连接默认 postgres 库并创建目标数据库
		log.WithCtx(ctx).Warnf("连接数据库失败，尝试创建数据库: %v", err)
		if createErr := createDatabaseIfNotExists(ctx, cfg); createErr != nil {
			log.WithCtx(ctx).Panic("创建数据库失败:", createErr)
		}
		// 重新连接
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.WithCtx(ctx).Panic("连接 PostgreSQL 失败:", err)
		}
	}

	// 启用 pgvector 扩展
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.WithCtx(ctx).Panic("启用 pgvector 扩展失败:", err)
	}

	return &PgVector{
		db:        db,
		score:     cfg.Score,
		dimension: int(cfg.Dimension),
	}
}

// createDatabaseIfNotExists 如果数据库不存在则创建
func createDatabaseIfNotExists(ctx context.Context, cfg *types.PgVector) error {
	// 连接到默认的 postgres 数据库
	defaultDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password)

	db, err := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接默认数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	defer sqlDB.Close()

	// 检查数据库是否存在
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = '%s')", cfg.DBName)
	if err := db.Raw(query).Scan(&exists).Error; err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	if exists {
		log.WithCtx(ctx).Infof("数据库 %s 已存在", cfg.DBName)
		return nil
	}

	// 创建数据库
	createSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
	if err := db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}

	log.WithCtx(ctx).Infof("数据库 %s 创建成功", cfg.DBName)
	return nil
}

// Close 关闭连接
func (p *PgVector) Close(ctx context.Context) {
	sqlDB, err := p.db.DB()
	if err != nil {
		log.WithCtx(ctx).Error("获取数据库连接失败:", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.WithCtx(ctx).Error("关闭数据库连接失败:", err)
	}
}

// GetDB 获取 gorm.DB 实例
func (p *PgVector) GetDB() *gorm.DB {
	return p.db
}

// DedupSearch 向量去重搜索
func (p *PgVector) DedupSearch(ctx context.Context, queryVector []float32, collectionName string) (bool, string, float32, error) {
	endT := time.Now()
	startT := endT.Add(-time.Hour * 24)

	var results []struct {
		Title string  `gorm:"column:title"`
		Score float32 `gorm:"column:score"`
	}

	// 使用 pgvector 的 cosine similarity 搜索
	query := fmt.Sprintf(`
		SELECT title, 1 - (embedding <=> $1::vector) as score
		FROM %s
		WHERE date >= $2 AND date <= $3
		ORDER BY embedding <=> $1::vector
		LIMIT 1
	`, collectionName)

	err := p.db.Raw(query, vectorToString(queryVector), startT, endT).Scan(&results).Error
	if err != nil {
		return false, "", 0, fmt.Errorf("搜索失败: %w", err)
	}

	if len(results) > 0 && results[0].Score > p.score {
		return true, results[0].Title, results[0].Score, nil
	}
	return false, "", 0, nil
}

// Query24HData 查询24小时内的数据
func (p *PgVector) Query24HData(ctx context.Context, collectionName string) (map[string][]float32, error) {
	endT := time.Now()
	startT := endT.Add(-time.Hour * 24)

	var results []struct {
		Title     string `gorm:"column:title"`
		Embedding string `gorm:"column:embedding"`
	}

	query := fmt.Sprintf(`
		SELECT title, embedding::text
		FROM %s
		WHERE date >= $1 AND date <= $2
	`, collectionName)

	err := p.db.Raw(query, startT, endT).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	data := make(map[string][]float32)
	for _, r := range results {
		vector, err := stringToVector(r.Embedding)
		if err != nil {
			log.WithCtx(ctx).Errorw("解析向量失败", "title", r.Title, "error", err)
			continue
		}
		data[r.Title] = vector
	}

	return data, nil
}

// CreateCollection 创建集合（表）
func (p *PgVector) CreateCollection(ctx context.Context, collectionName string) error {
	// 检查表是否已存在
	var exists bool
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = '%s'
		)
	`, collectionName)
	err := p.db.Raw(query).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("检查表是否存在失败: %w", err)
	}

	if exists {
		log.WithCtx(ctx).Infof("集合 %s 已存在，跳过创建", collectionName)
		return nil
	}

	// 根据表名创建不同的表结构
	var createTableSQL string
	if collectionName == "rss_vectors" {
		// rss_vectors 表需要 content 字段
		createTableSQL = fmt.Sprintf(`
			CREATE TABLE %s (
				id SERIAL PRIMARY KEY,
				title VARCHAR(255) NOT NULL,
				md5 VARCHAR(32) NOT NULL UNIQUE,
				date TIMESTAMP NOT NULL,
				content TEXT,
				embedding vector(%d)
			)
		`, collectionName, p.dimension)
	} else {
		// dedup_title 等其他表
		createTableSQL = fmt.Sprintf(`
			CREATE TABLE %s (
				id SERIAL PRIMARY KEY,
				title VARCHAR(255) NOT NULL,
				md5 VARCHAR(32) NOT NULL,
				date TIMESTAMP NOT NULL,
				embedding vector(%d)
			)
		`, collectionName, p.dimension)
	}

	if err := p.db.Exec(createTableSQL).Error; err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	// 创建索引
	indexSQL := fmt.Sprintf(`
		CREATE INDEX idx_%s_embedding ON %s USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)
	`, collectionName, collectionName)

	if err := p.db.Exec(indexSQL).Error; err != nil {
		log.WithCtx(ctx).Warnf("创建索引失败（可能数据量不足）: %v", err)
	}

	return nil
}

// Insert 插入数据
func (p *PgVector) Insert(ctx context.Context, date time.Time, md5, title string, vectors []float32, collectionName string) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (title, md5, date, embedding)
		VALUES ($1, $2, $3, $4::vector)
	`, collectionName)

	err := p.db.Exec(query, title, md5, date, vectorToString(vectors)).Error
	if err != nil {
		return fmt.Errorf("插入数据失败: %w", err)
	}

	log.WithCtx(ctx).Infow("pgvector", "insert", title, "md5", md5)
	return nil
}

// vectorToString 将向量转换为 PostgreSQL 格式
func vectorToString(vector []float32) string {
	result := "["
	for i, v := range vector {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", v)
	}
	result += "]"
	return result
}

// stringToVector 将 PostgreSQL 格式转换为向量
func stringToVector(s string) ([]float32, error) {
	var vector []float32
	_, err := fmt.Sscanf(s, "[%f]", &vector)
	if err != nil {
		// 尝试另一种格式
		vector = make([]float32, 0)
		for _, v := range splitVectorString(s) {
			var f float32
			fmt.Sscanf(v, "%f", &f)
			vector = append(vector, f)
		}
	}
	return vector, nil
}

// splitVectorString 分割向量字符串
func splitVectorString(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == '[' || c == ']' {
			continue
		}
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
