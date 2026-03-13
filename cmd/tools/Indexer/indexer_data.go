package main

import (
	"context"
	"os"
	"sync"

	"podcast/config"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/internal/database/models"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func main() {
	id := os.Args[1]
	ctx := context.Background()
	c, err := config.LoadAppConfig("/root/podcast/config.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(ctx).Infof("%#v", c)
	log.WithCtx(ctx).Infof("%#v", c)
	_, err = models.Init(c)
	if err != nil {
		panic(err)
	}

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
	llmInfo, err := llmPool.Get(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("Get error: %w", err)
		return
	}

	// 创建 RAG 引擎配置
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

	engine, err := rag.NewEngine(ctx, llmInfo, ragCfg)
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine error: %w", err)
		return
	}
	var posts []*models.RssContent

	err = models.GetDb().Model(&models.RssContent{}).Where("id > ?", id).Find(&posts).Error
	if err != nil {
		log.WithCtx(ctx).Errorf("FindAll error: %s", err)
		os.Exit(1)
		return
	}

	pChan := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for _, content := range posts {
		wg.Add(1)
		go func(content *models.RssContent) {
			pChan <- struct{}{}
			defer func() {
				<-pChan
				wg.Done()
			}()
			if err := engine.AddRssContent(ctx, content); err != nil {
				log.WithCtx(ctx).Errorw("AddRssContent", "id", content.ID, "err", err)
				return
			}
			log.WithCtx(ctx).Infow("AddRssContent", "成功", content.ID)
		}(content)

	}
	wg.Wait()
}
