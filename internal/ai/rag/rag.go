package rag

import (
	"context"
	"fmt"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/pgvector"

	"podcast/internal/database/models"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
	"gorm.io/gorm"
)

// Engine rag 引擎（负责 RSS 内容的分段向量化入库；
// 语义检索由 agent 的 PgVectorSearchTool 直接查库，不经过引擎）
type Engine struct {
	Dimension int64 // 嵌入维度
	pgVector  *pgvector.PgVector
	embedder  *embedding.Embedder
	cfg       *types.RagConfig
}

// NewEngine 创建 rag 引擎
func NewEngine(ctx context.Context, cfg *types.RagConfig) (*Engine, error) {
	// 初始化 embedder
	embedder, err := embedding.NewEmbedder(ctx, cfg.Embedding)
	if err != nil {
		return nil, err
	}

	// 初始化 pgvector
	pgVector := pgvector.NewPgVector(ctx, cfg.PgVector)

	// 创建 rss_vectors 表（如果不存在）
	if err := pgVector.CreateCollection(ctx, cfg.PgVector.RssCollection); err != nil {
		pgVector.Close(ctx)
		return nil, fmt.Errorf("创建 rss_vectors 表失败: %w", err)
	}

	return &Engine{
		Dimension: cfg.PgVector.Dimension,
		pgVector:  pgVector,
		embedder:  embedder,
		cfg:       cfg,
	}, nil
}

// AddRssContent 添加Rss内容（分段向量化：每段最多 DefaultChunkSize 字，相邻段重叠 DefaultChunkOverlap 字）
func (e *Engine) AddRssContent(ctx context.Context, rss *models.RssContent) error {
	log.WithCtx(ctx).Infow("AddRssContent", "title", rss.Title)

	chunks := ChunkContent(rss.Content, DefaultChunkSize, DefaultChunkOverlap)
	if len(chunks) == 0 {
		log.WithCtx(ctx).Warnw("AddRssContent 无内容可索引", "title", rss.Title)
		return nil
	}

	collectionName := e.cfg.PgVector.RssCollection
	db := e.pgVector.GetDB()

	err := db.Transaction(func(tx *gorm.DB) error {
		// 删除该文章的旧分段（重新向量化时分段数可能变化）
		if err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE md5 = $1`, collectionName), rss.MD5).Error; err != nil {
			return fmt.Errorf("删除旧分段失败: %w", err)
		}

		insertSQL := fmt.Sprintf(`
			INSERT INTO %s (title, md5, chunk_index, date, content, embedding)
			VALUES ($1, $2, $3, $4, $5, $6::halfvec)
		`, collectionName)

		for i, chunk := range chunks {
			chunkVector, err := e.embedder.CreateEmbeddings(ctx, chunk)
			if err != nil {
				return fmt.Errorf("生成第 %d 段向量失败: %w", i+1, err)
			}
			if err := tx.Exec(insertSQL, rss.Title, rss.MD5, i, rss.Date, chunk, pgvector.VectorToString(chunkVector)).Error; err != nil {
				return fmt.Errorf("插入第 %d 段失败: %w", i+1, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("插入数据失败: %w", err)
	}

	log.WithCtx(ctx).Infow("AddRssContent completed", "title", rss.Title, "chunks", len(chunks))
	return nil
}

// Close 关闭引擎资源
func (e *Engine) Close(ctx context.Context) {
	if e.pgVector != nil {
		e.pgVector.Close(ctx)
	}
}

// rss_vectors 表结构（用于存储 RSS 内容分段向量，一篇文章多个分段）
// CREATE TABLE IF NOT EXISTS rss_vectors (
//     id SERIAL PRIMARY KEY,
//     title VARCHAR(255) NOT NULL,
//     md5 VARCHAR(32) NOT NULL,
//     chunk_index INTEGER NOT NULL DEFAULT 0,
//     date TIMESTAMP NOT NULL,
//     content TEXT,
//     embedding halfvec(N),
//     UNIQUE (md5, chunk_index)
// );
