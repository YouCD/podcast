package rag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/semantic"
	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/cloudwego/eino/components/document"
	"github.com/youcd/toolkit/log"
)

// NewMarkdownSplitter 创建 markdown 分割器
func NewMarkdownSplitter(ctx context.Context) (document.Transformer, error) {
	splitter, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers: map[string]string{
			"##": "title",
		},
		TrimHeaders: false,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("NewMarkdownSplitter err: %v", err)
		return nil, err
	}
	return splitter, nil
}

func NewSemanticSplitter(ctx context.Context, embedder *dashscope.Embedder) (document.Transformer, error) {
	// 初始化分割器
	splitter, err := semantic.NewSplitter(ctx, &semantic.Config{
		Embedding:    embedder,
		BufferSize:   2,
		MinChunkSize: 10,
		Separators:   []string{"\n", ".", "?", "!"},
		Percentile:   0.9,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("NewSemanticSplitter err: %v", err)
		return nil, err
	}
	return splitter, nil
}

func NewRecursiveSplitter(ctx context.Context) (document.Transformer, error) {
	// 初始化分割器
	splitter, err := recursive.NewSplitter(ctx, &recursive.Config{
		ChunkSize:   1000,
		OverlapSize: 200,
		Separators:  []string{"\n\n", "\n", "。", "！", "？"},
		KeepType:    recursive.KeepTypeEnd,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("NewRecursiveSplitter err: %v", err)

		return nil, fmt.Errorf("NewRecursiveSplitter err: %w", err)
	}

	return splitter, nil
}
