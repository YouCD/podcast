package main

import (
	"context"
	"fmt"
	"io"
	"podcast/config"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func main() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.local.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)

	ctx := context.Background()
	// 转换 LLM 配置
	llmInfos := make([]*types.LLMInfo, len(c.LLM))
	for i, llmCfg := range c.LLM {
		llmInfos[i] = &types.LLMInfo{
			ID:      int64(i),
			ApiKey:  llmCfg.ApiKey,
			Model:   llmCfg.Model,
			BaseURL: llmCfg.BaseURL,
		}
	}
	llmPool := llm.NewLLMPool(llmInfos)
	get, err := llmPool.Get(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("Get error: %w", err)
		return
	}
	ragCfg := &types.RagConfig{
		Milvus: &types.Milvus{
			Endpoint:      c.Database.Milvus.Endpoint,
			APIKey:        c.Database.Milvus.APIKey,
			DBName:        c.Database.Milvus.DBName,
			RssCollection: c.Database.Milvus.RssCollection,
		},
		Embedding: &types.Embedding{
			APIKey: c.Vector.Embedding.APIKey,
			Model:  c.Vector.Embedding.Model,
		},
	}
	ragCli, err := rag.NewEngine(ctx, get, ragCfg)
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine error: %w", err)
		return
	}

	// 检索
	var query string
	for {
		_, _ = fmt.Scan(&query)
		rsp, err := ragCli.Query(ctx, query)
		if err != nil {
			log.WithCtx(ctx).Errorf("Query error: %w", err)
			return
		}

		// 流式输出
		for {
			output, err := rsp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithCtx(ctx).Errorf("Recv error: %w", err)
			}
			fmt.Print(output.Content)
		}
	}
}
