package workflow

import (
	"context"
	"fmt"

	"podcast/internal/ai/rag"
	"podcast/internal/database/models"
	"podcast/pkg/types"
)

func saveToPgVector(ctx context.Context, rss *models.RssContent, cfg *types.RagConfig) error {
	engine, err := rag.NewEngine(ctx, cfg)
	if err != nil {
		return fmt.Errorf("new engine error: %w", err)
	}
	defer engine.Close(ctx)
	err = engine.AddRssContent(ctx, rss)
	if err != nil {
		return fmt.Errorf("insert error: %w", err)
	}
	return nil
}
