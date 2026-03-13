package rag

import (
	"context"

	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	milvus2Ret "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/youcd/toolkit/log"
)

// NewMilvusRetriever 创建 redis retriever
func NewMilvusRetriever(ctx context.Context, embedder *dashscope.Embedder, topK int, cfg *types.Milvus) (*milvus2Ret.Retriever, error) {
	// 创建 retriever
	retriever, err := milvus2Ret.NewRetriever(ctx, &milvus2Ret.RetrieverConfig{
		ClientConfig: getMilvusClientConfig(cfg),
		Collection:   cfg.RssCollection,
		TopK:         topK,
		SearchMode:   search_mode.NewApproximate(milvus2Ret.COSINE),
		Embedding:    embedder,
	})
	if err != nil {
		log.WithCtx(ctx).Fatalf("Failed to create retriever: %v", err)
		return nil, err
	}
	return retriever, nil
}
