package workflow

import (
	"context"
	"fmt"

	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/internal/database/models"
	"podcast/pkg/types"
)

func saveToMilvus(ctx context.Context, llmPool *llm.LLMPool, rss *models.RssContent, cfg *types.RagConfig) error {
	llmInfo, err := llmPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("get llm error: %w", err)
	}
	defer llmPool.Put(ctx, llmInfo)
	engine, err := rag.NewEngine(ctx, llmInfo, cfg)
	if err != nil {
		return fmt.Errorf("new engine error: %w", err)
	}
	err = engine.AddRssContent(ctx, rss)
	if err != nil {
		return fmt.Errorf("insert error: %w", err)
	}
	return nil
}
