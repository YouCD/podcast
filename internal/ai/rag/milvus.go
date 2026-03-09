package rag

import (
	"podcast/config"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func getMlvusClientConfig() *milvusclient.ClientConfig {
	return &milvusclient.ClientConfig{
		Address: config.Cfg.Database.Milvus.Endpoint,
		APIKey:  config.Cfg.Database.Milvus.APIKey,
		DBName:  config.Cfg.Database.Milvus.DBName,
	}
}
