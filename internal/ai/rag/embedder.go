package rag

import (
	"context"
	"podcast/config"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/youcd/toolkit/log"
)

// NewQwenEmbedder 创建 ark embedder
func NewQwenEmbedder(ctx context.Context) (*dashscope.Embedder, error) {
	// 初始化 embedder
	embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		APIKey: config.Cfg.Vector.Embedding.APIKey,
		Model:  config.Cfg.Vector.Embedding.Model,
	})
	if err != nil {
		log.WithCtx(ctx).Error("NewQwenEmbedder failed, init embedder err: %v", err)
		return nil, err
	}
	return embedder, nil
}
