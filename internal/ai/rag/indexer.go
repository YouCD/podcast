package rag

import (
	"context"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// MilvusIndexerConfig Milvus Indexer 配置
type MilvusIndexerConfig struct {
	ClientConfig *types.Milvus
	Collection   string
}

// NewMilvusIndexer 创建 redis indexer
func NewMilvusIndexer(ctx context.Context, embedder *dashscope.Embedder, cfg MilvusIndexerConfig) (*milvus2.Indexer, error) {
	indexer, err := milvus2.NewIndexer(ctx, &milvus2.IndexerConfig{
		ClientConfig: getMilvusClientConfig(cfg.ClientConfig),
		Collection:   cfg.Collection,
		Vector: &milvus2.VectorConfig{
			Dimension:    1024, // Match your embedding model dimension
			MetricType:   milvus2.COSINE,
			IndexBuilder: milvus2.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
		},
		Embedding: embedder,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("NewMilvusIndexer failed, init indexer err: %v", err)
		return nil, err
	}
	return indexer, nil
}

// InitMilvusVectorIndex 初始化向量索引
func InitMilvusVectorIndex(ctx context.Context, client *milvus2.Indexer, docs []*schema.Document) error {
	// Store每次最多支持10个文档，因此需要分批处理
	batchSize := 10
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		log.WithCtx(ctx).Infof("Processing batch [%d-%d] of %d documents", i, end-1, len(docs))

		ids, err := client.Store(ctx, batch)
		if err != nil {
			log.WithCtx(ctx).Errorw("InitMilvusVectorIndex failed on batch", "batch_start", i, "batch_end", end, "err", err)
			return err
		}
		log.WithCtx(ctx).Infof("Batch [%d-%d] stored successfully, got %v IDs", i, end-1, len(ids))
	}

	log.WithCtx(ctx).Infof("InitMilvusVectorIndex completed successfully, total processed: %d documents", len(docs))
	return nil
}
