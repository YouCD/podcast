package rag

import (
	"podcast/pkg/types"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func getMilvusClientConfig(cfg *types.Milvus) *milvusclient.ClientConfig {
	return &milvusclient.ClientConfig{
		Address: cfg.Endpoint,
		APIKey:  cfg.APIKey,
		DBName:  cfg.DBName,
	}
}
