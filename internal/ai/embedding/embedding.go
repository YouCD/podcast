package embedding

import (
	"context"
	"fmt"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/youcd/toolkit/log"
)

// Embedder Embedding 客户端
type Embedder struct {
	model string
	*dashscope.Embedder
}

// NewEmbedder 创建 ark embedder
func NewEmbedder(ctx context.Context, cfg *types.Embedding) (*Embedder, error) {
	d := int(cfg.Dimension)
	// 初始化 embedder
	embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		APIKey:     cfg.APIKey,
		Model:      cfg.Model,
		Dimensions: &d,
	})
	if err != nil {
		log.WithCtx(ctx).Error("NewQwenEmbedder failed, init embedder err: %v", err)
		return nil, err
	}
	return &Embedder{
		model:    cfg.Model,
		Embedder: embedder,
	}, nil
	//return embedder, nil
}

// CreateEmbeddings 创建嵌入向量
func (e *Embedder) CreateEmbeddings(ctx context.Context, content string) ([]float32, error) {
	var resultA []float32
	result, err := e.Embedder.EmbedStrings(ctx, []string{content})
	if err != nil {
		return nil, fmt.Errorf("embedding strings failed: %w", err)
	}
	if len(result) > 0 {
		for _, f := range result[0] {
			resultA = append(resultA, float32(f))
		}
	}
	return resultA, nil
}
