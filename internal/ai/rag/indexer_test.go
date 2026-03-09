package rag

import (
	"context"
	"log"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestNewMilvusIndexer(t *testing.T) {
	ctx := context.Background()
	// 初始化 embedder
	embedder, err := NewQwenEmbedder(ctx)
	if err != nil {
		t.Log(err)
		return
	}
	indexer, err := NewMilvusIndexer(ctx, embedder)
	if err != nil {
		t.Log(err)
		return
	}

	// 存储文档
	docs := []*schema.Document{
		{
			ID:      "doc1",
			Content: "Milvus is an open-source vector database",
			MetaData: map[string]any{
				"category": "database",
				"year":     2021,
			},
		},
		{
			ID:      "doc2",
			Content: "EINO is a framework for building AI applications",
		},
	}
	ids, err := indexer.Store(ctx, docs)
	if err != nil {
		log.Fatalf("Failed to store: %v", err)
		return
	}
	log.Printf("Store success, ids: %v", ids)
}
