package workflow

import (
	"context"
	"fmt"

	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/internal/database/models"
)

func saveToMilvus(ctx context.Context, rss *models.RssContent) error {
	llmPool := llm.NewLLMPool()
	llmInfo, err := llmPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("get llm error: %w", err)
	}
	defer llmPool.Put(ctx, llmInfo)
	engine, err := rag.NewEngine(ctx, llmInfo)
	if err != nil {
		return fmt.Errorf("new engine error: %w", err)
	}
	err = engine.AddRssContent(ctx, rss)
	if err != nil {
		return fmt.Errorf("insert error: %w", err)
	}
	return nil
}
